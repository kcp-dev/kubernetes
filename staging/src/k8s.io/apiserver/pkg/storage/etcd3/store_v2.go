/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"reflect"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd3/metrics"
	"k8s.io/apiserver/pkg/storage/value"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"
)

type storeV2 struct {
	*store

	preconditions []PreconditionFactory
	guards        []TransactionGuardFactory
}

// NewV2 returns an etcd3 implementation of storage.Interface.
func NewV2(c *clientv3.Client, codec runtime.Codec, newFunc func() runtime.Object, prefix string, transformer value.Transformer, pagingEnabled bool, leaseManagerConfig LeaseManagerConfig) storage.Interface {
	return newStoreV2(c, codec, newFunc, prefix, transformer, pagingEnabled, leaseManagerConfig)
}

func newStoreV2(c *clientv3.Client, codec runtime.Codec, newFunc func() runtime.Object, prefix string, transformer value.Transformer, pagingEnabled bool, leaseManagerConfig LeaseManagerConfig) *storeV2 {
	result := &storeV2{
		store:         newStore(c, codec, newFunc, prefix, transformer, pagingEnabled, leaseManagerConfig),
		preconditions: []PreconditionFactory{NewQuotaPrecondition},
		guards:        []TransactionGuardFactory{NewMutable},
	}
	return result
}

func (s *storeV2) additional(ctx context.Context, key string) ([]Precondition, []TransactionGuard) {
	var preconditions []Precondition
	for _, f := range s.preconditions {
		preconditions = append(preconditions, f(ctx, s.pathPrefix, key))
	}
	var guards []TransactionGuard
	for _, f := range s.guards {
		guards = append(guards, f(ctx, s.pathPrefix, key))
	}
	return preconditions, guards
}

// Create implements storage.Interface.Create.
func (s *storeV2) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	clusterName, err := genericapirequest.ClusterNameFrom(ctx)
	if err != nil {
		klog.Errorf("No cluster defined in Create action for key %s : %s", key, err.Error())
	}

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
	key = path.Join(s.pathPrefix, key)

	opts, err := s.ttlOpts(ctx, int64(ttl))
	if err != nil {
		return err
	}

	newData, err := s.transformer.TransformToStorage(data, authenticatedDataString(key))
	if err != nil {
		return storage.NewInternalError(err.Error())
	}

	ctx = WithInitialObject(ctx, &StoredObject{}) // no initial value
	ctx = WithFinalObject(ctx, &StoredObject{
		Object:   obj,
		Metadata: nil,
		Data:     newData,
		Stale:    false,
	})

	preconditions, guards := s.additional(ctx, key)
	txnResp, err := transact(ctx, s.client, "create", getTypeName(obj),
		preconditions,
		[]*PotentiallyCachedResponse{
			{}, // TODO: feed cached quota values?
		},
		append(guards, &NotFound{
			key: key,
		}),
		clientv3.OpPut(key, string(newData), opts...),
	)
	if err != nil {
		return err
	}

	if out != nil {
		putResp := txnResp.Responses[0].GetResponsePut()
		return decode(s.codec, s.versioner, data, out, putResp.Header.Revision, clusterName)
	}
	return nil
}

// Delete implements storage.Interface.Delete.
func (s *storeV2) Delete(
	ctx context.Context, key string, out runtime.Object, preconditions *storage.Preconditions,
	validateDeletion storage.ValidateObjectFunc, cachedExistingObject runtime.Object) error {
	v, err := conversion.EnforcePtr(out)
	if err != nil {
		return fmt.Errorf("unable to convert output object to pointer: %v", err)
	}
	key = path.Join(s.pathPrefix, key)
	return s.conditionalDelete(ctx, key, out, v, preconditions, validateDeletion, cachedExistingObject)
}

func (s *storeV2) conditionalDelete(
	ctx context.Context, key string, out runtime.Object, v reflect.Value, preconditions *storage.Preconditions,
	validateDeletion storage.ValidateObjectFunc, cachedExistingObject runtime.Object) error {
	var err error
	var clusterName string
	clusterName, err = genericapirequest.ClusterNameFrom(ctx)
	if err != nil {
		klog.Errorf("No cluster defined in conditionalDelete action for key %s : %s", key, err.Error())
	}

	var initial *StoredObject
	if cachedExistingObject != nil {
		var err error
		initial, err = s.storedObjectFrom(cachedExistingObject)
		if err != nil {
			return err
		}
	}
	ctx = WithInitialObject(ctx, initial)
	ctx = WithFinalObject(ctx, &StoredObject{}) // no final value

	txnPreconditions, guards := s.additional(ctx, key)
	_, err = transact(ctx, s.client, "delete", getTypeName(out),
		append(txnPreconditions, &DeletePrecondition{
			key:              key,
			preconditions:    preconditions,
			validateDeletion: validateDeletion,
			decode: func(response *clientv3.GetResponse) (runtime.Object, error) {
				err = s.updateStoredObjectWith(initial, response, key, v, false, clusterName)
				return initial.Object, err
			},
		}),
		[]*PotentiallyCachedResponse{
			{}, // TODO: feed cached quota values?
			{}, // TODO: get cached object into this, loooool
		},
		guards,
		clientv3.OpDelete(key),
	)
	if err != nil {
		return err
	}

	return decode(s.codec, s.versioner, initial.Data, out, int64(initial.Metadata.ResourceVersion), clusterName)
}

// GuaranteedUpdate implements storage.Interface.GuaranteedUpdate.
func (s *storeV2) GuaranteedUpdate(
	ctx context.Context, key string, out runtime.Object, ignoreNotFound bool,
	preconditions *storage.Preconditions, tryUpdate storage.UpdateFunc, cachedExistingObject runtime.Object) error {
	trace := utiltrace.New("GuaranteedUpdate etcd3", utiltrace.Field{"type", getTypeName(out)})
	defer trace.LogIfLong(500 * time.Millisecond)

	clusterName, err := genericapirequest.ClusterNameFrom(ctx)
	if err != nil {
		klog.Errorf("No cluster defined in GuaranteedUpdate action for key %s : %s", key, err.Error())
	}

	v, err := conversion.EnforcePtr(out)
	if err != nil {
		return fmt.Errorf("unable to convert output object to pointer: %v", err)
	}
	key = path.Join(s.pathPrefix, key)

	getCurrentState := func() (*objState, error) {
		startTime := time.Now()
		getResp, err := s.client.KV.Get(ctx, key)
		metrics.RecordEtcdRequestLatency("get", getTypeName(out), startTime)
		if err != nil {
			return nil, err
		}
		return s.getState(getResp, key, v, ignoreNotFound, clusterName)
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
	trace.Step("initial value restored")

	transformContext := authenticatedDataString(key)
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

		ret, ttl, err := s.updateState(origState, tryUpdate)
		if err != nil {
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

		data, err := runtime.Encode(s.codec, ret)
		if err != nil {
			return err
		}
		if !origState.stale && bytes.Equal(data, origState.data) {
			// if we skipped the original Get in this loop, we must refresh from
			// etcd in order to be sure the data in the store is equivalent toz
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
				return decode(s.codec, s.versioner, origState.data, out, origState.rev, clusterName)
			}
		}

		newData, err := s.transformer.TransformToStorage(data, transformContext)
		if err != nil {
			return storage.NewInternalError(err.Error())
		}

		opts, err := s.ttlOpts(ctx, int64(ttl))
		if err != nil {
			return err
		}
		trace.Step("Transaction prepared")

		startTime := time.Now()
		txnResp, err := s.client.KV.Txn(ctx).If(
			clientv3.Compare(clientv3.ModRevision(key), "=", origState.rev),
		).Then(
			clientv3.OpPut(key, string(newData), opts...),
		).Else(
			clientv3.OpGet(key),
		).Commit()
		metrics.RecordEtcdRequestLatency("update", getTypeName(out), startTime)
		if err != nil {
			return err
		}
		trace.Step("Transaction committed")
		if !txnResp.Succeeded {
			getResp := (*clientv3.GetResponse)(txnResp.Responses[0].GetResponseRange())
			klog.V(4).Infof("GuaranteedUpdate of %s failed because of a conflict, going to retry", key)
			origState, err = s.getState(getResp, key, v, ignoreNotFound, clusterName)
			if err != nil {
				return err
			}
			trace.Step("Retry value restored")
			origStateIsCurrent = true
			continue
		}
		putResp := txnResp.Responses[0].GetResponsePut()

		return decode(s.codec, s.versioner, data, out, putResp.Header.Revision, clusterName)
	}
}

func (s *storeV2) updateStoredObjectWith(storedObject *StoredObject, response *clientv3.GetResponse, key string, v reflect.Value, ignoreNotFound bool, clusterName string) error {
	if u, ok := v.Addr().Interface().(runtime.Unstructured); ok {
		storedObject.Object = u.NewEmptyInstance()
	} else {
		storedObject.Object = reflect.New(v.Type()).Interface().(runtime.Object)
	}

	if len(response.Kvs) == 0 {
		if !ignoreNotFound {
			return storage.NewKeyNotFoundError(key, 0)
		}
		if err := runtime.SetZeroValue(storedObject.Object); err != nil {
			return err
		}
	} else {
		if storedObject.Metadata.ResourceVersion == uint64(response.Kvs[0].ModRevision) {
			// no need to transform or decode, we're already up-to-date
			return nil
		}
		data, stale, err := s.transformer.TransformFromStorage(response.Kvs[0].Value, authenticatedDataString(key))
		if err != nil {
			return storage.NewInternalError(err.Error())
		}
		storedObject.Metadata.ResourceVersion = uint64(response.Kvs[0].ModRevision)
		storedObject.Data = data
		storedObject.Stale = stale
		if err := decode(s.codec, s.versioner, storedObject.Data, storedObject.Object, int64(storedObject.Metadata.ResourceVersion), clusterName); err != nil {
			return err
		}
	}
	return nil
}

func (s *storeV2) getState(getResp *clientv3.GetResponse, key string, v reflect.Value, ignoreNotFound bool, clusterName string) (*objState, error) {
	state := &objState{
		meta: &storage.ResponseMeta{},
	}

	if u, ok := v.Addr().Interface().(runtime.Unstructured); ok {
		state.obj = u.NewEmptyInstance()
	} else {
		state.obj = reflect.New(v.Type()).Interface().(runtime.Object)
	}

	if len(getResp.Kvs) == 0 {
		if !ignoreNotFound {
			return nil, storage.NewKeyNotFoundError(key, 0)
		}
		if err := runtime.SetZeroValue(state.obj); err != nil {
			return nil, err
		}
	} else {
		data, stale, err := s.transformer.TransformFromStorage(getResp.Kvs[0].Value, authenticatedDataString(key))
		if err != nil {
			return nil, storage.NewInternalError(err.Error())
		}
		state.rev = getResp.Kvs[0].ModRevision
		state.meta.ResourceVersion = uint64(state.rev)
		state.data = data
		state.stale = stale
		if err := decode(s.codec, s.versioner, state.data, state.obj, state.rev, clusterName); err != nil {
			return nil, err
		}
	}
	return state, nil
}

func (s *storeV2) storedObjectFrom(obj runtime.Object) (*StoredObject, error) {
	storedObject := &StoredObject{
		Object:   obj,
		Metadata: &storage.ResponseMeta{},
		Stale:    false,
	}

	rv, err := s.versioner.ObjectResourceVersion(obj)
	if err != nil {
		return nil, fmt.Errorf("couldn't get resource version: %v", err)
	}
	storedObject.Metadata.ResourceVersion = rv

	// Compute the serialized form - for that we need to temporarily clean
	// its resource version field (those are not stored in etcd).
	if err := s.versioner.PrepareObjectForStorage(obj); err != nil {
		return nil, fmt.Errorf("PrepareObjectForStorage failed: %v", err)
	}
	storedObject.Data, err = runtime.Encode(s.codec, obj)
	if err != nil {
		return nil, err
	}
	if err := s.versioner.UpdateObject(storedObject.Object, rv); err != nil {
		klog.Errorf("failed to update object version: %v", err)
	}
	return storedObject, nil
}

func (s *storeV2) updateState(st *objState, userUpdate storage.UpdateFunc) (runtime.Object, uint64, error) {
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

// ttlOpts returns client options based on given ttl.
// ttl: if ttl is non-zero, it will attach the key to a lease with ttl of roughly the same length
func (s *storeV2) ttlOpts(ctx context.Context, ttl int64) ([]clientv3.OpOption, error) {
	if ttl == 0 {
		return nil, nil
	}
	id, err := s.leaseManager.GetLease(ctx, ttl)
	if err != nil {
		return nil, err
	}
	return []clientv3.OpOption{clientv3.WithLease(id)}, nil
}
