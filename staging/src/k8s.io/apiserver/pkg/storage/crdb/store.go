package crdb

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/cockroachdb/apd"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/value"
	"k8s.io/apiserver/pkg/storage/versioner"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/klog/v2"
)

func New(c pool, codec runtime.Codec, newFunc func() runtime.Object, prefix string, groupResource schema.GroupResource, transformer value.Transformer, pagingEnabled bool) storage.Interface {
	return newStore(c, codec, newFunc, prefix, groupResource, transformer, pagingEnabled)
}

func newStore(c pool, codec runtime.Codec, newFunc func() runtime.Object, prefix string, groupResource schema.GroupResource, transformer value.Transformer, pagingEnabled bool) *store {
	objectVersioner := versioner.APIObjectVersioner{}
	return &store{
		client:        c,
		codec:         codec,
		versioner:     objectVersioner,
		transformer:   transformer,
		pagingEnabled: pagingEnabled,
		// for compatibility with etcd2 impl.
		// no-op for default prefix of '/registry'.
		// keeps compatibility with etcd2 impl for custom prefixes that don't start with '/'
		pathPrefix:          path.Join("/", prefix),
		groupResource:       groupResource,
		groupResourceString: groupResource.String(),
		watcher:             newWatcher(c, codec, newFunc, objectVersioner, transformer),
	}
}

// pool holds the methods we need, so we can mock this out in tests
type pool interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	BeginTxFunc(ctx context.Context, txOptions pgx.TxOptions, f func(pgx.Tx) error) error
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

type store struct {
	client              pool
	codec               runtime.Codec
	versioner           storage.Versioner
	transformer         value.Transformer
	pathPrefix          string
	groupResource       schema.GroupResource
	groupResourceString string
	watcher             *watcher
	pagingEnabled       bool
}

func (s *store) Versioner() storage.Versioner {
	return s.versioner
}

func (s *store) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	cluster := request.ClusterFrom(ctx)
	if cluster == nil {
		klog.Errorf(storage.NewInternalError("no cluster metadata found").Error())
		cluster = &request.Cluster{
			Name:     "admin",
			Parents:  nil,
			Wildcard: false,
		}
	}
	key = path.Join(s.pathPrefix, key)

	if version, err := s.versioner.ObjectResourceVersion(obj); err == nil && version != 0 {
		return errors.New("resourceVersion should not be set on objects to be created")
	}
	if err := s.versioner.PrepareObjectForStorage(obj); err != nil {
		return fmt.Errorf("PrepareObjectForStorage failed: %v", err)
	}
	data, err := runtime.Encode(s.codec, obj)
	if err != nil {
		return err
	}
	newData, err := s.transformer.TransformToStorage(data, authenticatedDataString(key))
	if err != nil {
		return storage.NewInternalError(err.Error())
	}

	resourceVersion, err := create(ctx, s.client, key, newData)
	if err != nil {
		return err
	}
	return decode(s.codec, s.versioner, data, out, resourceVersion, cluster.Name)
}

func create(ctx context.Context, client pool, key string, data []byte) (int64, error) {
	cluster := request.ClusterFrom(ctx)
	if cluster == nil {
		return 0, storage.NewInternalError("no cluster metadata found")
	}
	requestInfo, haveRequestInfo := request.RequestInfoFrom(ctx)
	if !haveRequestInfo {
		klog.Errorf(storage.NewInternalError("no request info metadata found").Error())
	}

	// we can't use INSERT ... RETURNING crdb_internal_mvcc_timestamp
	// so we need to use a transaction here to ensure we do not race
	var mvccTimestamp apd.Decimal
	if err := client.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var execErr error
		if haveRequestInfo {
			_, execErr = tx.Exec(ctx, `INSERT INTO k8s (key, value, cluster, namespace, name, api_group, api_version, api_resource) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`, key, data, cluster.Name, requestInfo.Namespace, requestInfo.Name, requestInfo.APIGroup, requestInfo.APIVersion, requestInfo.Resource)
		} else {
			_, execErr = tx.Exec(ctx, `INSERT INTO k8s (key, value, cluster) VALUES ($1, $2, $3);`, key, data, cluster.Name)
		}
		if execErr != nil {
			return execErr
		}

		if err := tx.QueryRow(ctx, `SELECT crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1 AND cluster=$2;`, key, cluster.Name).Scan(&hybridLogicalTimestamp); err != nil {
			return err
		}
		return nil
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return 0, storage.NewKeyExistsError(key, 0)
			}
		}
		return 0, storage.NewInternalError(err.Error())
	}

	return toResourceVersion(mvccTimestamp)
}

func toResourceVersion(mvccTimestamp apd.Decimal) (int64, error) {
	return mvccTimestamp.Int64() // TODO: this effectively truncates the logical part of the HLC - what's the implication?
}

func (s *store) Delete(ctx context.Context, key string, out runtime.Object, preconditions *storage.Preconditions, validateDeletion storage.ValidateObjectFunc, cachedExistingObject runtime.Object) error {
	v, err := conversion.EnforcePtr(out)
	if err != nil {
		return fmt.Errorf("unable to convert output object to pointer: %v", err)
	}
	key = path.Join(s.pathPrefix, key)
	return s.conditionalDelete(ctx, key, out, v, preconditions, validateDeletion, cachedExistingObject)
}

func (s *store) conditionalDelete(ctx context.Context, key string, out runtime.Object, v reflect.Value, preconditions *storage.Preconditions, validateDeletion storage.ValidateObjectFunc, cachedExistingObject runtime.Object) error {
	cluster := request.ClusterFrom(ctx)
	if cluster == nil {
		klog.Errorf(storage.NewInternalError("no cluster metadata found").Error())
		cluster = &request.Cluster{
			Name:     "admin",
			Parents:  nil,
			Wildcard: false,
		}
	}

	getCurrentState := func() (*objState, error) {
		var mvccTimestamp apd.Decimal
		var blob []byte
		err := s.client.QueryRow(ctx, `SELECT value, crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1 AND cluster=$2;`, key, cluster.Name).Scan(&blob, &mvccTimestamp)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, storage.NewKeyNotFoundError(key, 0)
			}
			return nil, storage.NewInternalError(err.Error())
		}
		resourceVersion, rvErr := toResourceVersion(mvccTimestamp)
		if rvErr != nil {
			return nil, rvErr
		}
		return s.getState(blob, resourceVersion, key, v, errors.Is(err, pgx.ErrNoRows), false, cluster.Name)
	}

	var origState *objState
	var origStateIsCurrent bool
	var err error
	if cachedExistingObject != nil {
		origState, err = s.getStateFromObject(cachedExistingObject)
	} else {
		origState, err = getCurrentState()
		origStateIsCurrent = true
	}
	if err != nil {
		return err
	}

	for {
		if preconditions != nil {
			if err := preconditions.Check(key, origState.obj); err != nil {
				if origStateIsCurrent {
					return err
				}

				// It's possible we're working with stale data.
				// Remember the revision of the potentially stale data and the resulting update error
				cachedRev := origState.rev
				cachedUpdateErr := err

				// Actually fetch
				origState, err = getCurrentState()
				if err != nil {
					return err
				}
				origStateIsCurrent = true

				// it turns out our cached data was not stale, return the error
				if cachedRev == origState.rev {
					return cachedUpdateErr
				}

				// Retry
				continue
			}
		}
		if err := validateDeletion(ctx, origState.obj); err != nil {
			if origStateIsCurrent {
				return err
			}

			// It's possible we're working with stale data.
			// Remember the revision of the potentially stale data and the resulting update error
			cachedRev := origState.rev
			cachedUpdateErr := err

			// Actually fetch
			origState, err = getCurrentState()
			if err != nil {
				return err
			}
			origStateIsCurrent = true

			// it turns out our cached data was not stale, return the error
			if cachedRev == origState.rev {
				return cachedUpdateErr
			}

			// Retry
			continue
		}

		var retry bool
		var mvccTimestamp apd.Decimal
		var blob []byte
		if err := s.client.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `DELETE FROM k8s WHERE key=$1 AND cluster=$2 AND crdb_internal_mvcc_timestamp=$3;`, key, cluster.Name, origState.rev)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				retry = true
				// we didn't delete anything, either because the value we used for our precondition was too old,
				// or because someone raced with us to delete the thing - so, check to see if there's some updated
				// value we can use for preconditions next time around
				return tx.QueryRow(ctx, `SELECT value, crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1 AND cluster=$2;`, key, cluster.Name).Scan(&blob, &mvccTimestamp)
			}
			return nil
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return storage.NewKeyNotFoundError(key, 0)
			}
			return storage.NewInternalError(err.Error())
		}
		if retry {
			resourceVersion, rvErr := toResourceVersion(mvccTimestamp)
			if rvErr != nil {
				return rvErr
			}
			origState, err = s.getState(blob, resourceVersion, key, v, errors.Is(err, pgx.ErrNoRows), false, cluster.Name)
			if err != nil {
				return err
			}
			origStateIsCurrent = true
			continue
		}
		return decode(s.codec, s.versioner, origState.data, out, origState.rev, cluster.Name)
	}
}

func (s *store) Watch(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	return s.watch(ctx, key, opts, false)
}

func (s *store) WatchList(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	return s.watch(ctx, key, opts, true)
}

func (s *store) watch(ctx context.Context, key string, opts storage.ListOptions, recursive bool) (watch.Interface, error) {
	rev, err := s.versioner.ParseResourceVersion(opts.ResourceVersion)
	if err != nil {
		return nil, err
	}
	key = path.Join(s.pathPrefix, key)

	// HACK: would need to be an argument to storage (or a change to how decoding works for key structure)
	cluster, err := request.ValidClusterFrom(ctx)
	if err != nil {
		return nil, storage.NewInternalError(fmt.Sprintf("Invalid cluster for key %s : %v", key, err))
	}
	clusterName := cluster.Name
	if cluster.Wildcard {
		clusterName = "*"
	}

	return s.watcher.Watch(ctx, key, int64(rev), recursive, clusterName, opts.ProgressNotify, opts.Predicate)
}

func (s *store) Get(ctx context.Context, key string, opts storage.GetOptions, out runtime.Object) error {
	cluster := request.ClusterFrom(ctx)
	if cluster == nil {
		klog.Errorf(storage.NewInternalError("no cluster metadata found").Error())
		cluster = &request.Cluster{
			Name:     "admin",
			Parents:  nil,
			Wildcard: false,
		}
	}
	key = path.Join(s.pathPrefix, key)

	// NOTE: the k8s GET semantics are weird - when passing a resourceVersion in a GET parameter,
	// you are only ever going to get something that is no older than the resourceVersion, and the
	// check is done against the logical time *of the read*, not of the last write to the data...
	var data []byte
	var mvccTimestamp apd.Decimal
	var clusterTimestamp apd.Decimal
	if err := s.client.QueryRow(ctx, `SELECT value, crdb_internal_mvcc_timestamp, cluster_logical_timestamp() FROM k8s WHERE key=$1 AND cluster=$2;`, key, cluster.Name).Scan(&data, &mvccTimestamp, &clusterTimestamp); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if opts.IgnoreNotFound {
				return runtime.SetZeroValue(out)
			}
			return storage.NewKeyNotFoundError(key, 0)
		} else {
			return storage.NewInternalError(err.Error())
		}
	}

	latestResourceVersion, err := toResourceVersion(clusterTimestamp)
	if err != nil {
		return err
	}

	if err = s.validateMinimumResourceVersion(opts.ResourceVersion, uint64(latestResourceVersion)); err != nil {
		return err
	}

	objectResourceVersion, err := toResourceVersion(mvccTimestamp)
	if err != nil {
		return err
	}
	data, _, err = s.transformer.TransformFromStorage(data, authenticatedDataString(key))
	if err != nil {
		return storage.NewInternalError(err.Error())
	}

	return decode(s.codec, s.versioner, data, out, objectResourceVersion, cluster.Name)
}

func (s *store) GetToList(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	cluster := request.ClusterFrom(ctx)
	if cluster == nil {
		klog.Errorf(storage.NewInternalError("no cluster metadata found").Error())
		cluster = &request.Cluster{
			Name:     "admin",
			Parents:  nil,
			Wildcard: false,
		}
	}
	key = path.Join(s.pathPrefix, key)

	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return err
	}
	v, err := conversion.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		return fmt.Errorf("need ptr to slice: %v", err)
	}

	newItemFunc := getNewItemFunc(listObj, v)

	var clause string
	if opts.ResourceVersion != "" && opts.ResourceVersion != "0" && opts.ResourceVersionMatch == metav1.ResourceVersionMatchExact {
		// TODO: with a paid license for CRDB we can use `with_max_timestamp()` to get the exact same
		// semantics as today's k8s GET - without that, though, we can only enforce an exact match
		resourceVersion, err := s.versioner.ParseResourceVersion(opts.ResourceVersion)
		if err != nil {
			return err
		}
		providedTimestamp := apd.New(int64(resourceVersion), 0)
		// TODO: can we do something better here for the time? using a var makes:
		// ERROR: AS OF SYSTEM TIME: only constant expressions, with_min_timestamp, with_max_staleness, or follower_read_timestamp are allowed (SQLSTATE XXUUU)
		clause = fmt.Sprintf(`AS OF SYSTEM TIME %s `, providedTimestamp)
	}

	// NOTE: the k8s GETOLIST semantics are weird - when passing a resourceVersion in a GET parameter,
	// the check is done against the logical time *of the read*, not of the last write to the data...
	// however, unlike raw GET, we *do* want to be able to get the actual historical version of the data,
	// so we can't just SELECT ... crdb_internal_mvcc_timestamp, cluster_logical_timestamp() ... AS OF SYSTEM TIME
	// as we need the *current* cluster_logical_timestamp()
	var data []byte
	var mvccTimestamp apd.Decimal
	var clusterTimestamp apd.Decimal
	found := true
	if err := s.client.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// we need to read the cluster timestamp even if the data read fails with a NotFound
		// we furthermore need to do the AS OF SYSTEM TIME query first:
		// ERROR: internal error: cannot set fixed timestamp, txn "sql txn" meta={id=6fd70624 pri=0.03834544 epo=0 ts=1645025006.198715497,0 min=1645025006.198715497,0 seq=0} lock=false stat=PENDING rts=1645025006.198715497,0 wto=false gul=1645025006.698715497,0 already performed reads (SQLSTATE XX000)
		dataReadErr := tx.QueryRow(ctx, fmt.Sprintf(`SELECT value, crdb_internal_mvcc_timestamp FROM k8s %sWHERE key=$1 AND cluster=$2;`, clause), key, cluster.Name).Scan(&data, &mvccTimestamp)
		timestampReadErr := tx.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&clusterTimestamp)
		if dataReadErr != nil {
			return dataReadErr // prefer data read err if we failed both
		}
		return timestampReadErr // otherwise, return in order
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			found = false // this is *not* an error for this call
		} else {
			return storage.NewInternalError(err.Error())
		}
	}

	latestResourceVersion, err := toResourceVersion(clusterTimestamp)
	if err != nil {
		return err
	}

	if err = s.validateMinimumResourceVersion(opts.ResourceVersion, uint64(latestResourceVersion)); err != nil {
		return err
	}

	if found {
		objectResourceVersion, err := toResourceVersion(mvccTimestamp)
		if err != nil {
			return err
		}
		data, _, err = s.transformer.TransformFromStorage(data, authenticatedDataString(key))
		if err != nil {
			return storage.NewInternalError(err.Error())
		}
		if err := appendListItem(v, data, uint64(objectResourceVersion), opts.Predicate, s.codec, s.versioner, newItemFunc, cluster.Name); err != nil {
			return err
		}
	}

	// update version with cluster level revision
	return s.versioner.UpdateList(listObj, uint64(latestResourceVersion), "", nil)
}

const and = " AND "

// whereClause is meant to append `WHERE a=b AND c=d` style clauses to a sql query with positional placeholders for the args
func whereClause(offset int, cluster *request.Cluster, info *request.RequestInfo) (clause string, args []interface{}) {
	var clauses []string
	if offset == 0 {
		offset++
	}
	if cluster != nil && cluster.Name != "" && !cluster.Wildcard {
		clauses = append(clauses, fmt.Sprintf("cluster=$%d", offset))
		args = append(args, cluster.Name)
		offset++
	}
	// TODO: request.RequestInfo only seems to contain name on field selectors, not creates...
	//if info != nil && info.Name != "" {
	//	clauses = append(clauses, fmt.Sprintf("name=$%d", offset))
	//	args = append(args, info.Name)
	//	offset++
	//}
	if info != nil && info.Namespace != metav1.NamespaceAll {
		clauses = append(clauses, fmt.Sprintf("namespace=$%d", offset))
		args = append(args, info.Namespace)
		offset++
	}
	if info != nil && info.APIVersion != "" {
		clauses = append(clauses, fmt.Sprintf("api_version=$%d", offset))
		args = append(args, info.APIVersion)
		offset++
	}
	if info != nil && info.APIGroup != "" {
		clauses = append(clauses, fmt.Sprintf("api_group=$%d", offset))
		args = append(args, info.APIGroup)
		offset++
	}
	if info != nil && info.Resource != "" {
		clauses = append(clauses, fmt.Sprintf("api_resource=$%d", offset))
		args = append(args, info.Resource)
		offset++
	}
	var prefix string
	if len(clauses) > 0 {
		prefix = and
	}
	return prefix + strings.Join(clauses, and), args
}

func (s *store) List(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	cluster := request.ClusterFrom(ctx)
	if cluster == nil {
		klog.Errorf(storage.NewInternalError("no cluster metadata found").Error())
		cluster = &request.Cluster{Name: "admin"}
	}
	requestInfo, haveRequestInfo := request.RequestInfoFrom(ctx)
	if !haveRequestInfo {
		klog.Errorf(storage.NewInternalError("no request info metadata found").Error())
	}

	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return err
	}
	v, err := conversion.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		return fmt.Errorf("need ptr to slice: %v", err)
	}

	if s.pathPrefix != "" {
		key = path.Join(s.pathPrefix, key)
	}
	// We need to make sure the key ended with "/" so that we only get children "directories".
	// e.g. if we have key "/a", "/a/b", "/ab", getting keys with prefix "/a" will return all three,
	// while with prefix "/a/" will return only "/a/b" which is the correct answer.
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	keyPrefix := key

	// NOTE: the whole clientv3.WithRange(rangeEnd) prefixing thing here basically
	// exists to limit a LIST call to the "subdirectories" for a given GVR, which
	// we get for free with our indexing/column selectors, so we can disregard the
	// range-end option

	// set the appropriate options to filter the returned data set
	var revisionClause string
	var limitClause string
	var paging bool
	if s.pagingEnabled && opts.Predicate.Limit > 0 {
		paging = true
		limitClause = fmt.Sprintf(` LIMIT %d`, opts.Predicate.Limit)
	}

	newItemFunc := getNewItemFunc(listObj, v)

	var fromRV *uint64
	if len(opts.ResourceVersion) > 0 {
		parsedRV, err := s.versioner.ParseResourceVersion(opts.ResourceVersion)
		if err != nil {
			return apierrors.NewBadRequest(fmt.Sprintf("invalid resource version: %v", err))
		}
		fromRV = &parsedRV
	}

	var returnedRV, continueRV, withRev int64
	var continueKey string
	switch {
	case s.pagingEnabled && len(opts.Predicate.Continue) > 0:
		continueKey, continueRV, err = DecodeContinue(opts.Predicate.Continue, keyPrefix)
		if err != nil {
			return apierrors.NewBadRequest(fmt.Sprintf("invalid continue token: %v", err))
		}

		if len(opts.ResourceVersion) > 0 && opts.ResourceVersion != "0" {
			return apierrors.NewBadRequest("specifying resource version is not allowed when using continue")
		}
		key = continueKey

		// If continueRV > 0, the LIST request needs a specific resource version.
		// continueRV==0 is invalid.
		// If continueRV < 0, the request is for the latest resource version.
		if continueRV > 0 {
			withRev = continueRV
			returnedRV = continueRV
		}
	case s.pagingEnabled && opts.Predicate.Limit > 0:
		if fromRV != nil {
			switch opts.ResourceVersionMatch {
			case metav1.ResourceVersionMatchNotOlderThan:
				// The not older than constraint is checked after we get a response from etcd,
				// and returnedRV is then set to the revision we get from the etcd response.
			case metav1.ResourceVersionMatchExact:
				returnedRV = int64(*fromRV)
				withRev = returnedRV
			case "": // legacy case
				if *fromRV > 0 {
					returnedRV = int64(*fromRV)
					withRev = returnedRV
				}
			default:
				return fmt.Errorf("unknown ResourceVersionMatch value: %v", opts.ResourceVersionMatch)
			}
		}
	default:
		if fromRV != nil {
			switch opts.ResourceVersionMatch {
			case metav1.ResourceVersionMatchNotOlderThan:
				// The not older than constraint is checked after we get a response from etcd,
				// and returnedRV is then set to the revision we get from the etcd response.
			case metav1.ResourceVersionMatchExact:
				returnedRV = int64(*fromRV)
				withRev = returnedRV
			case "": // legacy case
			default:
				return fmt.Errorf("unknown ResourceVersionMatch value: %v", opts.ResourceVersionMatch)
			}
		}
	}

	// loop until we have filled the requested limit from crdb or there are no more results
	var lastKey []byte
	var hasMore bool
	var numFetched int
	var numEvald int
	var remaining int64
	for {
		if withRev != 0 {
			providedTimestamp := apd.New(withRev, 0)
			// TODO: can we do something better here for the time? using a var makes:
			// ERROR: AS OF SYSTEM TIME: only constant expressions, with_min_timestamp, with_max_staleness, or follower_read_timestamp are allowed (SQLSTATE XXUUU)
			revisionClause = fmt.Sprintf(`AS OF SYSTEM TIME %s `, providedTimestamp)
		}

		var sliceFull bool
		var finalKey []byte
		var clusterTimestamp apd.Decimal
		// as always, the resourceVersion parameter the client passes in is compared to the
		// latest possible system clock, not the last write to the data, so we need to use a
		// transaction here to read that ...
		if err := s.client.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			// TODO: perhaps just for tests, we need WHERE key LIKE prefix% since those tests expect real prefixing ... but that is not ideal for production
			// we need to read the cluster timestamp even if the data read fails with a NotFound
			if err := tx.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&clusterTimestamp); err != nil {
				return err
			}

			// TODO: perhaps just for tests, we need WHERE key LIKE prefix% since those tests expect real prefixing ... but that is not ideal for production
			clauses, args := whereClause(2, cluster, requestInfo)
			args = append([]interface{}{key}, args...)
			if err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*), MAX(key) FROM k8s WHERE key LIKE '%s%%' AND key >= $1%s;`, keyPrefix, clauses), args...).Scan(&remaining, &finalKey); err != nil {
				return err
			}

			query := fmt.Sprintf(`SELECT key, value, crdb_internal_mvcc_timestamp FROM k8s WHERE key LIKE '%s%%' AND key >= $1%s ORDER BY key%s;`, keyPrefix, clauses, limitClause)
			rows, err := tx.Query(ctx, query, args...)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			defer func() {
				rows.Close()
			}()

			// we can be certain that we will read up to our limit, if the data is present
			numRows := remaining
			if paging && opts.Predicate.Limit < remaining {
				numRows = opts.Predicate.Limit
			}
			// avoid small allocations for the result slice, since this can be called in many
			// different contexts and we don't know how significantly the result will be filtered
			if opts.Predicate.Empty() {
				growSlice(v, int(numRows))
			} else {
				growSlice(v, 2048, int(numRows))
			}
			for rows.Next() {
				if err := rows.Err(); err != nil {
					return err
				}

				if paging && int64(v.Len()) >= opts.Predicate.Limit {
					hasMore = true
					sliceFull = true
					return nil
				}

				numFetched += 1
				remaining -= 1

				var data []byte
				var mvccTimestamp apd.Decimal
				if err := rows.Scan(&lastKey, &data, &mvccTimestamp); err != nil {
					return err
				}

				clusterName := cluster.Name
				if cluster.Wildcard {
					sub := strings.TrimPrefix(string(lastKey), keyPrefix)
					if i := strings.Index(sub, "/"); i != -1 {
						clusterName = sub[:i]
					}
					if clusterName == "" {
						klog.Errorf("the cluster name of extracted object should not be empty for key %q", string(lastKey))
					}
				}

				objectResourceVersion, err := toResourceVersion(mvccTimestamp)
				if err != nil {
					return err
				}

				data, _, err = s.transformer.TransformFromStorage(data, authenticatedDataString(lastKey))
				if err != nil {
					return storage.NewInternalErrorf("unable to transform key %q: %v", lastKey, err)
				}

				if err := appendListItem(v, data, uint64(objectResourceVersion), opts.Predicate, s.codec, s.versioner, newItemFunc, clusterName); err != nil {
					return err
				}
				numEvald++
			}

			return nil
		}); err != nil {
			return interpretListError(err, len(opts.Predicate.Continue) > 0, continueKey, keyPrefix)
		}

		if sliceFull {
			break
		}

		latestResourceVersion, err := toResourceVersion(clusterTimestamp)
		if err != nil {
			return err
		}

		if err = s.validateMinimumResourceVersion(opts.ResourceVersion, uint64(latestResourceVersion)); err != nil {
			return err
		}

		hasMore = !bytes.Equal(lastKey, finalKey)

		// indicate to the client which resource version was returned
		if returnedRV == 0 {
			returnedRV = latestResourceVersion
		}

		// no more results remain or we didn't request paging
		if !hasMore || !paging {
			break
		}
		// we're paging but we have filled our bucket
		if int64(v.Len()) >= opts.Predicate.Limit {
			break
		}
		key = string(lastKey) + "\x00"
		if withRev == 0 {
			withRev = returnedRV
		}
	}

	// instruct the client to begin querying from immediately after the last key we returned
	// we never return a key that the client wouldn't be allowed to see
	if hasMore {
		// we want to start immediately after the last key
		next, err := EncodeContinue(string(lastKey)+"\x00", keyPrefix, returnedRV)
		if err != nil {
			return err
		}
		var remainingItemCount *int64
		// out remaining count considers all objects that do not match the opts.Predicate.
		// Instead of returning inaccurate count for non-empty selectors, we return nil.
		// Only set remainingItemCount if the predicate is empty.
		if utilfeature.DefaultFeatureGate.Enabled(features.RemainingItemCount) {
			if opts.Predicate.Empty() {
				remainingItemCount = &remaining
			}
		}
		return s.versioner.UpdateList(listObj, uint64(returnedRV), next, remainingItemCount)
	}

	// no continuation
	return s.versioner.UpdateList(listObj, uint64(returnedRV), "", nil)
}

func (s *store) GuaranteedUpdate(ctx context.Context, key string, out runtime.Object, ignoreNotFound bool, preconditions *storage.Preconditions, tryUpdate storage.UpdateFunc, cachedExistingObject runtime.Object) error {
	cluster := request.ClusterFrom(ctx)
	if cluster == nil {
		klog.Errorf(storage.NewInternalError("no cluster metadata found").Error())
		cluster = &request.Cluster{
			Name:     "admin",
			Parents:  nil,
			Wildcard: false,
		}
	}
	requestInfo, haveRequestInfo := request.RequestInfoFrom(ctx)
	if !haveRequestInfo {
		klog.Errorf(storage.NewInternalError("no request info metadata found").Error())
	}
	key = path.Join(s.pathPrefix, key)

	v, err := conversion.EnforcePtr(out)
	if err != nil {
		return fmt.Errorf("unable to convert output object to pointer: %v", err)
	}
	transformContext := authenticatedDataString(key)

	getCurrentState := func() (*objState, error) {
		var mvccTimestamp apd.Decimal
		var blob []byte
		err := s.client.QueryRow(ctx, `SELECT value, crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1 AND cluster=$2;`, key, cluster.Name).Scan(&blob, &mvccTimestamp)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.NewInternalError(err.Error())
		}
		resourceVersion, rvErr := toResourceVersion(mvccTimestamp)
		if rvErr != nil {
			return nil, rvErr
		}
		return s.getState(blob, resourceVersion, key, v, errors.Is(err, pgx.ErrNoRows), ignoreNotFound, cluster.Name)
	}

	var origState *objState
	var origStateIsCurrent bool
	if cachedExistingObject != nil {
		origState, err = s.getStateFromObject(cachedExistingObject)
	} else {
		origState, err = getCurrentState()
		origStateIsCurrent = true
	}
	if err != nil {
		return err
	}

	for {
		if err := preconditions.Check(key, origState.obj); err != nil {
			// If our data is already up to date, return the error
			if origStateIsCurrent {
				return err
			}

			// It's possible we were working with stale data
			// Actually fetch
			origState, err = getCurrentState()
			if err != nil {
				return err
			}
			origStateIsCurrent = true
			// Retry
			continue
		}

		ret, _, err := s.updateState(origState, tryUpdate) // TODO: ttl?
		if err != nil {
			// If our data is already up to date, return the error
			if origStateIsCurrent {
				return err
			}

			// It's possible we were working with stale data
			// Remember the revision of the potentially stale data and the resulting update error
			cachedRev := origState.rev
			cachedUpdateErr := err

			// Actually fetch
			origState, err = getCurrentState()
			if err != nil {
				return err
			}
			origStateIsCurrent = true

			// it turns out our cached data was not stale, return the error
			if cachedRev == origState.rev {
				return cachedUpdateErr
			}

			// Retry
			continue
		}

		data, err := runtime.Encode(s.codec, ret)
		if err != nil {
			return err
		}
		if !origState.stale && bytes.Equal(data, origState.data) {
			// if we skipped the original Get in this loop, we must refresh from
			// etcd in order to be sure the data in the store is equivalent to
			// our desired serialization
			if !origStateIsCurrent {
				origState, err = getCurrentState()
				if err != nil {
					return err
				}
				origStateIsCurrent = true
				if !bytes.Equal(data, origState.data) {
					// original data changed, restart loop
					continue
				}
			}
			// recheck that the data from etcd is not stale before short-circuiting a write
			if !origState.stale {
				return decode(s.codec, s.versioner, origState.data, out, origState.rev, cluster.Name)
			}
		}

		newData, err := s.transformer.TransformToStorage(data, transformContext)
		if err != nil {
			return storage.NewInternalError(err.Error())
		}

		var retry bool
		var mvccTimestamp apd.Decimal
		var blob []byte
		if err := s.client.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			var tag pgconn.CommandTag
			var execErr error
			if haveRequestInfo {
				tag, execErr = tx.Exec(ctx, `INSERT INTO k8s (key, value, cluster, namespace, name, api_group, api_version, api_resource) 
											VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
											ON CONFLICT (key, cluster) 
											DO UPDATE SET value=excluded.value
											WHERE crdb_internal_mvcc_timestamp=$9;`,
					key, newData, cluster.Name, requestInfo.Namespace, requestInfo.Name, requestInfo.APIGroup, requestInfo.APIVersion, requestInfo.Resource, previousHybridLogicalTimestamp,
				)
			} else {
				tag, execErr = tx.Exec(ctx, `INSERT INTO k8s (key, value, cluster) 
											VALUES ($1, $2, $3) 
											ON CONFLICT (key, cluster) 
											DO UPDATE SET value=excluded.value
											WHERE crdb_internal_mvcc_timestamp=$4;`,
					key, newData, cluster.Name, previousHybridLogicalTimestamp,
				)
			}
			if execErr != nil {
				return execErr
			}
			if tag.RowsAffected() == 0 {
				retry = true
				// we didn't update anything, either because the value we used for our precondition was too old,
				// or because someone raced with us to delete the thing - so, check to see if there's some updated
				// value we can use for preconditions next time around
				return tx.QueryRow(ctx, `SELECT value, crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1 AND cluster=$2;`, key, cluster.Name).Scan(&blob, &mvccTimestamp)
			}
			// read the updated timestamp so we know the new resourceVersion
			return tx.QueryRow(ctx, `SELECT crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1 AND cluster=$2;`, key, cluster.Name).Scan(&mvccTimestamp)
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return storage.NewKeyNotFoundError(key, origState.rev)
			}
			return storage.NewInternalError(err.Error())
		}
		updatedResourceVersion, rvErr := toResourceVersion(mvccTimestamp)
		if rvErr != nil {
			return rvErr
		}
		if retry {
			origState, err = s.getState(blob, updatedResourceVersion, key, v, errors.Is(err, pgx.ErrNoRows), ignoreNotFound, cluster.Name)
			if err != nil {
				return err
			}
			origStateIsCurrent = true
			continue
		}
		return decode(s.codec, s.versioner, data, out, updatedResourceVersion, cluster.Name)
	}
}

func (s *store) Count(key string) (int64, error) {
	panic("this is hard and requires that we understand k8s/etcd prefixing mechanisms .. who actually uses this?!")
}

// below copied from etcd

type objState struct {
	obj   runtime.Object
	meta  *storage.ResponseMeta
	rev   int64
	data  []byte
	stale bool
}

func (s *store) getState(blob []byte, resourceVersion int64, key string, v reflect.Value, wasNotFound, ignoreNotFound bool, clusterName string) (*objState, error) {
	state := &objState{
		meta: &storage.ResponseMeta{},
	}

	if u, ok := v.Addr().Interface().(runtime.Unstructured); ok {
		state.obj = u.NewEmptyInstance()
	} else {
		state.obj = reflect.New(v.Type()).Interface().(runtime.Object)
	}

	if wasNotFound {
		if !ignoreNotFound {
			return nil, storage.NewKeyNotFoundError(key, 0)
		}
		if err := runtime.SetZeroValue(state.obj); err != nil {
			return nil, err
		}
	} else {
		data, stale, err := s.transformer.TransformFromStorage(blob, authenticatedDataString(key))
		if err != nil {
			return nil, storage.NewInternalError(err.Error())
		}
		state.rev = resourceVersion
		state.meta.ResourceVersion = uint64(state.rev)
		state.data = data
		state.stale = stale
		if err := decode(s.codec, s.versioner, state.data, state.obj, state.rev, clusterName); err != nil {
			return nil, err
		}
	}
	return state, nil
}

// decode decodes value of bytes into object. It will also set the object resource version to rev.
// On success, objPtr would be set to the object.
func decode(codec runtime.Codec, versioner storage.Versioner, value []byte, objPtr runtime.Object, rev int64, clusterName string) error {
	if _, err := conversion.EnforcePtr(objPtr); err != nil {
		return fmt.Errorf("unable to convert output object to pointer: %v", err)
	}
	_, _, err := codec.Decode(value, nil, objPtr)
	if err != nil {
		return err
	}
	// being unable to set the version does not prevent the object from being extracted
	if err := versioner.UpdateObject(objPtr, uint64(rev)); err != nil {
		klog.Errorf("failed to update object version: %v", err)
	}
	// HACK: in order to support CRD tenancy, the clusterName, which is extracted from the object etcd key,
	// should be set on the decoded object.
	// This is done here since we want to set the logical cluster the object is part of,
	// without storing the clusterName inside the etcd object itself (as it has been until now).
	// The etcd key is ultimately the only thing that links us to a cluster
	if clusterName != "" {
		if s, ok := objPtr.(metav1.ObjectMetaAccessor); ok {
			s.GetObjectMeta().SetClusterName(clusterName)
		} else if s, ok := objPtr.(metav1.Object); ok {
			s.SetClusterName(clusterName)
		} else if s, ok := objPtr.(*unstructured.Unstructured); ok {
			s.SetClusterName(clusterName)
		} else {
			klog.Warningf("Could not set ClusterName %s in appendListItem on object: %T", clusterName, objPtr)
		}
	} else {
		klog.Errorf("Cluster should not be unknown")
	}

	return nil
}

func (s *store) getStateFromObject(obj runtime.Object) (*objState, error) {
	state := &objState{
		obj:  obj,
		meta: &storage.ResponseMeta{},
	}

	rv, err := s.versioner.ObjectResourceVersion(obj)
	if err != nil {
		return nil, fmt.Errorf("couldn't get resource version: %v", err)
	}
	state.rev = int64(rv)
	state.meta.ResourceVersion = uint64(state.rev)

	// Compute the serialized form - for that we need to temporarily clean
	// its resource version field (those are not stored in etcd).
	if err := s.versioner.PrepareObjectForStorage(obj); err != nil {
		return nil, fmt.Errorf("PrepareObjectForStorage failed: %v", err)
	}
	state.data, err = runtime.Encode(s.codec, obj)
	if err != nil {
		return nil, err
	}
	if err := s.versioner.UpdateObject(state.obj, uint64(rv)); err != nil {
		klog.Errorf("failed to update object version: %v", err)
	}
	return state, nil
}

func (s *store) updateState(st *objState, userUpdate storage.UpdateFunc) (runtime.Object, uint64, error) {
	ret, ttlPtr, err := userUpdate(st.obj, *st.meta)
	if err != nil {
		return nil, 0, err
	}

	if err := s.versioner.PrepareObjectForStorage(ret); err != nil {
		return nil, 0, fmt.Errorf("PrepareObjectForStorage failed: %v", err)
	}
	var ttl uint64
	if ttlPtr != nil {
		ttl = *ttlPtr
	}
	return ret, ttl, nil
}

// validateMinimumResourceVersion returns a 'too large resource' version error when the provided minimumResourceVersion is
// greater than the most recent actualRevision available from storage.
func (s *store) validateMinimumResourceVersion(minimumResourceVersion string, actualRevision uint64) error {
	if minimumResourceVersion == "" {
		return nil
	}
	minimumRV, err := s.versioner.ParseResourceVersion(minimumResourceVersion)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid resource version: %v", err))
	}
	// Enforce the storage.Interface guarantee that the resource version of the returned data
	// "will be at least 'resourceVersion'".
	if minimumRV > actualRevision {
		return storage.NewTooLargeResourceVersionError(minimumRV, actualRevision, 0)
	}
	return nil
}

// authenticatedDataString satisfies the value.Context interface. It uses the key to
// authenticate the stored data. This does not defend against reuse of previously
// encrypted values under the same key, but will prevent an attacker from using an
// encrypted value from a different key. A stronger authenticated data segment would
// include the etcd3 Version field (which is incremented on each write to a key and
// reset when the key is deleted), but an attacker with write access to etcd can
// force deletion and recreation of keys to weaken that angle.
type authenticatedDataString string

// AuthenticatedData implements the value.Context interface.
func (d authenticatedDataString) AuthenticatedData() []byte {
	return []byte(string(d))
}

var _ value.Context = authenticatedDataString("")

// TODO: different encoding

// ContinueToken is a simple structured object for encoding the state of a continue token.
// TODO: if we change the version of the encoded from, we can't start encoding the new version
// until all other servers are upgraded (i.e. we need to support rolling update)
// This is a public API struct and cannot change.
type ContinueToken struct {
	APIVersion      string `json:"v"`
	ResourceVersion int64  `json:"rv"`
	StartKey        string `json:"start"`
}

// DecodeContinue transforms an encoded predicate from into a versioned struct.
// TODO: return a typed error that instructs clients that they must relist
func DecodeContinue(continueValue, keyPrefix string) (fromKey string, rv int64, err error) {
	data, err := base64.RawURLEncoding.DecodeString(continueValue)
	if err != nil {
		return "", 0, fmt.Errorf("continue key is not valid: %v", err)
	}
	var c ContinueToken
	if err := json.Unmarshal(data, &c); err != nil {
		return "", 0, fmt.Errorf("continue key is not valid: %v", err)
	}
	switch c.APIVersion {
	case "meta.k8s.io/v1":
		if c.ResourceVersion == 0 {
			return "", 0, fmt.Errorf("continue key is not valid: incorrect encoded start resourceVersion (version meta.k8s.io/v1)")
		}
		if len(c.StartKey) == 0 {
			return "", 0, fmt.Errorf("continue key is not valid: encoded start key empty (version meta.k8s.io/v1)")
		}
		// defend against path traversal attacks by clients - path.Clean will ensure that startKey cannot
		// be at a higher level of the hierarchy, and so when we append the key prefix we will end up with
		// continue start key that is fully qualified and cannot range over anything less specific than
		// keyPrefix.
		key := c.StartKey
		if !strings.HasPrefix(key, "/") {
			key = "/" + key
		}
		cleaned := path.Clean(key)
		if cleaned != key {
			return "", 0, fmt.Errorf("continue key is not valid: %s", c.StartKey)
		}
		return keyPrefix + cleaned[1:], c.ResourceVersion, nil
	default:
		return "", 0, fmt.Errorf("continue key is not valid: server does not recognize this encoded version %q", c.APIVersion)
	}
}

// EncodeContinue returns a string representing the encoded continuation of the current query.
func EncodeContinue(key, keyPrefix string, resourceVersion int64) (string, error) {
	nextKey := strings.TrimPrefix(key, keyPrefix)
	if nextKey == key {
		return "", fmt.Errorf("unable to encode next field: the key (%s) and key prefix (%s) do not match", key, keyPrefix)
	}
	out, err := json.Marshal(&ContinueToken{APIVersion: "meta.k8s.io/v1", ResourceVersion: resourceVersion, StartKey: nextKey})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(out), nil
}

func getNewItemFunc(listObj runtime.Object, v reflect.Value) func() runtime.Object {
	// For unstructured lists with a target group/version, preserve the group/version in the instantiated list items
	if unstructuredList, isUnstructured := listObj.(*unstructured.UnstructuredList); isUnstructured {
		if apiVersion := unstructuredList.GetAPIVersion(); len(apiVersion) > 0 {
			return func() runtime.Object {
				return &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": apiVersion}}
			}
		}
	}

	// Otherwise just instantiate an empty item
	elem := v.Type().Elem()
	return func() runtime.Object {
		return reflect.New(elem).Interface().(runtime.Object)
	}
}

// appendListItem decodes and appends the object (if it passes filter) to v, which must be a slice.
func appendListItem(v reflect.Value, data []byte, rev uint64, pred storage.SelectionPredicate, codec runtime.Codec, versioner storage.Versioner, newItemFunc func() runtime.Object, clusterName string) error {
	obj, _, err := codec.Decode(data, nil, newItemFunc())
	if err != nil {
		return err
	}
	// being unable to set the version does not prevent the object from being extracted
	if err := versioner.UpdateObject(obj, rev); err != nil {
		klog.Errorf("failed to update object version: %v", err)
	}

	// HACK: in order to support CRD tenancy, the clusterName, which is extracted from the object etcd key,
	// should be set on the decoded object.
	// This is done here since we want to set the logical cluster the object is part of,
	// without storing the clusterName inside the etcd object itself (as it has been until now).
	// The etcd key is ultimately the only thing that links us to a cluster
	if clusterName != "" {
		if s, ok := obj.(metav1.ObjectMetaAccessor); ok {
			s.GetObjectMeta().SetClusterName(clusterName)
		} else if s, ok := obj.(metav1.Object); ok {
			s.SetClusterName(clusterName)
		} else if s, ok := obj.(*unstructured.Unstructured); ok {
			s.SetClusterName(clusterName)
		} else {
			klog.Warningf("Could not set ClusterName %s in appendListItem on object: %T", clusterName, obj)
		}
	} else {
		klog.Errorf("Cluster should not be unknown")
	}

	if matched, err := pred.Matches(obj); err == nil && matched {
		v.Set(reflect.Append(v, reflect.ValueOf(obj).Elem()))
	}
	return nil
}

// growSlice takes a slice value and grows its capacity up
// to the maximum of the passed sizes or maxCapacity, whichever
// is smaller. Above maxCapacity decisions about allocation are left
// to the Go runtime on append. This allows a caller to make an
// educated guess about the potential size of the total list while
// still avoiding overly aggressive initial allocation. If sizes
// is empty maxCapacity will be used as the size to grow.
func growSlice(v reflect.Value, maxCapacity int, sizes ...int) {
	cap := v.Cap()
	max := cap
	for _, size := range sizes {
		if size > max {
			max = size
		}
	}
	if len(sizes) == 0 || max > maxCapacity {
		max = maxCapacity
	}
	if max <= cap {
		return
	}
	if v.Len() > 0 {
		extra := reflect.MakeSlice(v.Type(), 0, max)
		reflect.Copy(extra, v)
		v.Set(extra)
	} else {
		extra := reflect.MakeSlice(v.Type(), 0, max)
		v.Set(extra)
	}
}
