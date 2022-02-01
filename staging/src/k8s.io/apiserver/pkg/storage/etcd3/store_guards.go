package etcd3

import (
	"context"
	"path"

	etcdclient "go.etcd.io/etcd/client/v3"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"
)

// TransactionGuardFactory knows how to add guards to mutating requests for keys
type TransactionGuardFactory func(ctx context.Context, pathPrefix, key string) TransactionGuard

// TransactionGuard is a precondition that can be expressed within the etcdv3 transaction
// gRPC call. etcd does not expose which precondition failed (or why) when a transaction fails,
// so we must read the value during the transaction and post-process the resulting data to determine
// why our transaction failed.
type TransactionGuard interface {
	// If exposes a comparison to be added to a transactional precondition. The key will be fetched
	// during the Then() portion of the transaction, for post-hoc Handle()ing.
	If() etcdclient.Cmp
	// Handle determines if the transactional precondition failed, given the data it operated over.
	// Returning a non-nil error here will cause the overall storage.Interface call to fail.
	Handle(response *etcdclient.GetResponse) error
}

// NotFound asserts that a key does not yet have a value in etcd
type NotFound struct {
	key string
}

func (n *NotFound) If() etcdclient.Cmp {
	return etcdclient.Compare(etcdclient.ModRevision(n.key), "=", 0)
}

func (n *NotFound) Handle(response *etcdclient.GetResponse) error {
	switch l := len(response.Kvs); l {
	case 0:
		return nil
	case 1:
		return storage.NewKeyExistsError(n.key, response.Kvs[0].ModRevision)
	default:
		return storage.NewInternalErrorf("expected one key holding object, got %d: %#v", l, response.Kvs)
	}
}

var _ TransactionGuard = &NotFound{}

func NewMutable(ctx context.Context, pathPrefix, key string) TransactionGuard {
	clusterName, err := genericapirequest.ClusterNameFrom(ctx)
	if err != nil {
		klog.Errorf("No cluster defined for key %s : %s", key, err.Error())
		return nil
	}
	metaPrefix := path.Join(pathPrefix, "clusterdata", clusterName)
	return &Mutable{
		key: path.Join(metaPrefix, "readOnly"),
	}
}

var _ TransactionGuardFactory = NewMutable

// Mutable asserts that the key belongs to a mutable key-range.
type Mutable struct {
	key string
}

func (m *Mutable) If() etcdclient.Cmp {
	return etcdclient.Compare(etcdclient.Value(m.key), "!=", "true")
}

func (m *Mutable) Handle(response *etcdclient.GetResponse) error {
	switch l := len(response.Kvs); l {
	case 1:
		if v := response.Kvs[0]; string(v.Value) == "true" {
			return storage.NewReadOnlyError(m.key, v.ModRevision)
		}
		return nil
	default:
		return storage.NewInternalErrorf("expected one key holding object, got %d: %#v", l, response.Kvs)
	}
}

var _ TransactionGuard = &Mutable{}

// AtModRevision asserts that a key has not been modified from a specific revision
type AtModRevision struct {
	key         string
	modRevision int64

	consumeUpdatedData func(response *etcdclient.GetResponse) error
}

func (a *AtModRevision) If() etcdclient.Cmp {
	return etcdclient.Compare(etcdclient.ModRevision(a.key), "=", a.modRevision)
}

func (a *AtModRevision) Handle(response *etcdclient.GetResponse) error {
	switch l := len(response.Kvs); l {
	case 1:
		if v := response.Kvs[0]; v.ModRevision != a.modRevision {
			return a.consumeUpdatedData(response)
		}
		return nil
	default:
		return storage.NewInternalErrorf("expected one key holding object, got %d: %#v", l, response.Kvs)
	}
}

var _ TransactionGuard = &AtModRevision{}
