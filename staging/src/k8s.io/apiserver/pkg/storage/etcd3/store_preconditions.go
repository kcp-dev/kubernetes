package etcd3

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"path"
	"strconv"

	etcdclient "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"
)

// PreconditionFactory knows how to add preconditions to mutating requests for keys
type PreconditionFactory func(ctx context.Context, pathPrefix, key string) Precondition

// PreconditionKVClient is the non-mutating subset of KV client calls which guards have
// access to.
type PreconditionKVClient interface {
	Get(ctx context.Context, key string, opts ...etcdclient.OpOption) (*etcdclient.GetResponse, error)
}

// Precondition is a function that needs to be evaluated on data fetched from the store asynchronously,
// therefore also requiring that the state of the underlying data not change when the main transaction
// is committed.
type Precondition interface {
	// Check uses the client to fetch appropriate data, returning an error if the precondition does
	// not hold across the data. A RequiresLiveReadErr must wrap any error that requires a live read
	// from the client to be valid.
	Check(ctx context.Context, client PreconditionKVClient) error
	// Update exposes any mutating changes that should be made during the transaction that this
	// precondition guards.
	Update() []etcdclient.Op
}

type RequiresLiveReadErr struct {
	reason error
}

func (r *RequiresLiveReadErr) Error() string {
	return r.reason.Error()
}

func (r *RequiresLiveReadErr) Is(target error) bool {
	if target == nil {
		return false
	}
	_, ok := target.(*RequiresLiveReadErr)
	return ok
}

func (r *RequiresLiveReadErr) Unwrap() error {
	return r.reason
}

var _ error = &RequiresLiveReadErr{}

// DeletePrecondition asserts that the guards hold and that the object is valid to delete
type DeletePrecondition struct {
	key string

	preconditions    *storage.Preconditions
	validateDeletion storage.ValidateObjectFunc

	decode func(response *etcdclient.GetResponse) (runtime.Object, error)
}

func (d *DeletePrecondition) Check(ctx context.Context, client PreconditionKVClient) error {
	response, err := client.Get(ctx, d.key)
	if err != nil {
		return err
	}

	obj, err := d.decode(response)
	if err != nil {
		return err
	}

	if err := d.preconditions.Check(d.key, obj); err != nil {
		return &RequiresLiveReadErr{reason: err}
	}

	if err := d.validateDeletion(ctx, obj); err != nil {
		return &RequiresLiveReadErr{reason: err}
	}

	return nil
}

func (d *DeletePrecondition) Update() []etcdclient.Op {
	return nil
}

var _ Precondition = &DeletePrecondition{}

// NewQuotaPrecondition creates a new quota precondition for the key in question
func NewQuotaPrecondition(ctx context.Context, pathPrefix, key string) Precondition {
	clusterName, err := genericapirequest.ClusterNameFrom(ctx)
	if err != nil {
		klog.Errorf("No cluster defined for key %s : %s", key, err.Error())
		return nil
	}
	metaPrefix := path.Join(pathPrefix, "clusterdata", clusterName)
	return &QuotaPrecondition{
		keyBeingMutated:  key,
		quotaPrefixKey:   path.Join(metaPrefix, "quota"),
		stripesPrefixKey: "stripes",
		numStripesKey:    "numStripes",
		limitKey:         "limit",
	}
}

var _ PreconditionFactory = NewQuotaPrecondition

// QuotaPrecondition asserts that the change in object size due to the mutating request does not violate
// storage footprint quota. In order to not create write bottlenecks on mutating requests, storage footprint
// is recorded in a striped fashion.
type QuotaPrecondition struct {
	keyBeingMutated string

	quotaPrefixKey   string
	stripesPrefixKey string
	numStripesKey    string
	limitKey         string

	quotaUpate etcdclient.Op
}

func (q *QuotaPrecondition) Check(ctx context.Context, client PreconditionKVClient) error {
	// TODO: store more intelligently, using formatter + encryption, etc
	quotaLimit, err := fetchUint(ctx, client, path.Join(q.quotaPrefixKey, q.limitKey))
	if err != nil {
		return err
	}

	numStripes, err := fetchUint(ctx, client, path.Join(q.quotaPrefixKey, q.numStripesKey))
	if err != nil {
		return err
	}

	stripe := binary.BigEndian.Uint64(sha256.New().Sum([]byte(q.keyBeingMutated))) % numStripes

	quotaResponse, err := client.Get(ctx, path.Join(q.quotaPrefixKey, q.stripesPrefixKey), etcdclient.WithPrefix())
	if err != nil {
		return err
	}
	if len(quotaResponse.Kvs) != int(numStripes) {
		return storage.NewInternalErrorf("invalid quota stripes: expected %d responses, got %d", numStripes, len(quotaResponse.Kvs))
	}
	quota := make([]uint64, numStripes)
	for _, kv := range quotaResponse.Kvs {
		thisStripe, err := strconv.ParseUint(string(kv.Key), 10, 64)
		if err != nil {
			return storage.NewInternalErrorf("invalid stripe name at %q: %v", kv.Key, err)
		}
		thisQuota, err := strconv.ParseUint(string(kv.Value), 10, 64)
		if err != nil {
			return storage.NewInternalErrorf("invalid stripe value at %q: %v", kv.Key, err)
		}
		if thisStripe >= numStripes {
			return storage.NewInternalErrorf("invalid quota stripes: expected [0,%d), got %d", numStripes, thisStripe)
		}
		quota[thisStripe] = thisQuota
	}

	initial, ok := InitialObjectFrom(ctx)
	if !ok {
		return storage.NewInternalError("no initial object found")
	}
	final, ok := FinalObjectFrom(ctx)
	if !ok {
		return storage.NewInternalError("no initial object found")
	}
	quota[stripe] += uint64(len(final.Data)) - uint64(len(initial.Data))

	var totalQuota uint64
	for _, quotum := range quota {
		totalQuota += quotum
	}
	if totalQuota > quotaLimit {
		return &RequiresLiveReadErr{reason: storage.NewInsufficientQuotaError(q.keyBeingMutated, quotaResponse.Header.Revision)}
	}

	q.quotaUpate = etcdclient.OpPut(
		path.Join(q.quotaPrefixKey, q.stripesPrefixKey, strconv.Itoa(int(stripe))),
		strconv.Itoa(int(quota[stripe])),
	)

	return nil
}

func (q *QuotaPrecondition) Update() []etcdclient.Op {
	return []etcdclient.Op{q.quotaUpate}
}

func fetchUint(ctx context.Context, client PreconditionKVClient, key string) (uint64, error) {
	response, err := client.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	if len(response.Kvs) != 1 {
		return 0, storage.NewInternalErrorf("invalid gRPC response from etcd: expected 1 response, got %d", len(response.Kvs))
	}
	value, err := strconv.ParseUint(string(response.Kvs[0].Value), 10, 64)
	if err != nil {
		return 0, storage.NewInternalErrorf("invalid value at %q: %v", key, err)
	}
	return value, nil
}

var _ Precondition = &QuotaPrecondition{}
