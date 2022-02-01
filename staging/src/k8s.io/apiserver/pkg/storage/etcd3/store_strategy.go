/*
Copyright 2022 The Kubernetes Authors.

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
	"context"
	"errors"
	"fmt"
	"time"

	etcdclient "go.etcd.io/etcd/client/v3"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd3/metrics"
)

// Transactions to etcd allow us to enforce invariants on the state of the underlying store
// before we commit, but the built-in transaction implementation in the gRPC calls only allows
// for simple comparisons (on revisions or data). If we want to assert something about a field
// in the value of a stored key, we need to read the key, assert on the data, and furthermore
// assert that the key did not change when we're executing the transaction. The framework in
// this file allows for composition of such complex comparisons, as well as simple ones.
// Correctly issuing the correct gRPC calls to etcd requires some subtle invariants, which this
// framework enforces for the user:
// - native transactional guards (If) on a key are paired with reads on the key on failure
//   (Then) so that a client issuing transactions against etcd may determine which precondition
//   failed and why
// - only non-mutating requests may be done to acquire data necessary for non-native/asynchronous
//   guards
// - all keys fetched for non-native/asynchronous guards must have native transactional
//   guards to ensure their state has not changed before the transaction is committed
// - furthermore, all such keys must be re-fetched in the failure case (Then) of a transaction
//   so that the entire flow may be re-tried on conflict

// transact executes the operations given the guards and preconditions.
func transact(ctx context.Context, client *etcdclient.Client, verb, resource string, preconditions []Precondition, state []*PotentiallyCachedResponse, transactionGuards []TransactionGuard, operation etcdclient.Op) (*etcdclient.TxnResponse, error) {
	// TODO: somehow better align the cached state with the preconditions?
	preconditionClient := &preconditionKVClient{delegate: client}
	var txnResponse *etcdclient.TxnResponse
	var txnError error
attempt: // TODO: clean up this label
	for {
		var ops []etcdclient.Op
		for i, precondition := range preconditions {
			preconditionCtx := context.WithValue(ctx, contextKeyCache, state[i])
			if err := precondition.Check(preconditionCtx, preconditionClient); err != nil {
				if !errors.Is(err, &RequiresLiveReadErr{}) {
					// some errors are fatal to the transaction regardless, bubble them up straight away
					return nil, err
				}
				// some errors require that the data used to generate them is up-to-date
				if !state[i].cached {
					return nil, err
				}
				// if we failed on cached data, we need to start over and do a live read next time
				state[i].needsUpdate = true
				break attempt
			}
			// if the precondition needs to mutate, record it
			ops = append(ops, precondition.Update()...)
		}

		// if all preconditions succeeded, go forth with the transaction
		gTxn := &GuardedTransaction{
			ctx:        ctx,
			client:     client,
			verb:       verb,
			resource:   resource,
			guards:     append(preconditionClient.Guards(), transactionGuards...),
			operations: ops,
		}
		txnResponse, txnError = gTxn.Then(operation).Commit()
		if txnError != nil {
			// errors transacting or failed guards terminate this transaction
			return nil, txnError
		}
		if txnResponse.Succeeded {
			// successful transaction, we're done
			break
		}
		// no errors during the transaction, but we failed some guards - retry, using fresh data from the transaction
	}

	return txnResponse, nil
}

// contextKey allows for unique context value keys, as this type is unexported from this package
type contextKey string

const (
	// contextKeyCache is the key under which we store a potentially cached object and metadata
	contextKeyCache = contextKey("cached")
)

// PotentiallyCachedResponse is a response from etcd which may or may not be stale
type PotentiallyCachedResponse struct {
	response *etcdclient.GetResponse
	cached      bool
	needsUpdate bool
}

// preconditionKVClient is a client that knows how to read values from etcd and ensure that they are up-to-date
// when future transactions are done assuming those values.
type preconditionKVClient struct {
	delegate *etcdclient.Client

	guards []TransactionGuard
}

// Get serves potentially cached data, ensuring that all kvs exposed to the precondition calling this are recorded
// in modRevision guards so that we know the data used to evaluate the precondition does not change before the
// transaction is committed.
func (p *preconditionKVClient) Get(ctx context.Context, key string, opts ...etcdclient.OpOption) (*etcdclient.GetResponse, error) {
	item := ctx.Value(contextKeyCache)
	if item == nil {
		return nil, storage.NewInternalErrorf("found no cached item under key %q in context", contextKeyCache)
	}
	cachedResponse, ok := item.(*PotentiallyCachedResponse)
	if !ok {
		return nil, storage.NewInternalErrorf("found invalid cached item %T under key %q in context", item, contextKeyCache)
	}
	// if we don't have a cached response to return, or we need to update our cached version, fetch from etcd
	if cachedResponse.response == nil || (cachedResponse.cached && cachedResponse.needsUpdate) {
		response, err := p.delegate.Get(ctx, key, opts...)
		if err != nil {
			return response, err
		}
		(*cachedResponse).response = response
		(*cachedResponse).cached = false
		(*cachedResponse).needsUpdate = false
	}
	for i, kv := range cachedResponse.response.Kvs {
		p.guards = append(p.guards, &AtModRevision{
			key:         string(kv.Key),
			modRevision: kv.ModRevision,
			consumeUpdatedData: func(response *etcdclient.GetResponse) error {
				if len(response.Kvs) != 1 {
					return storage.NewInternalErrorf("invalid gRPC response from etcd: expected 1 response, got %d", len(response.Kvs))
				}
				(*cachedResponse).response.Header = response.Header
				(*cachedResponse).response.Kvs[i] = response.Kvs[0]
				(*cachedResponse).cached = false
				return nil
			},
		})
	}
	return cachedResponse.response, nil
}

func (p *preconditionKVClient) Guards() []TransactionGuard {
	return p.guards
}

var _ PreconditionKVClient = &preconditionKVClient{}

// GuardedTransaction closes over the process of running a transaction against etcd with guards,
// fetching all the necessary data to evaluate which guards failed, and returning an appropriate
// error
type GuardedTransaction struct {
	ctx    context.Context
	client *etcdclient.Client
	guards []TransactionGuard

	// TODO: is this the correct place for these?
	verb, resource string

	operations []etcdclient.Op
}

// Then records which operations should execute if the transaction guards succeed
func (g *GuardedTransaction) Then(ops ...etcdclient.Op) *GuardedTransaction {
	g.operations = append(g.operations, ops...)
	return g
}

// Commit executes the transaction against etcd
func (g *GuardedTransaction) Commit() (*etcdclient.TxnResponse, error) {
	var ifs []etcdclient.Cmp
	var elses []etcdclient.Op
	for _, guard := range g.guards {
		precondition := guard.If()
		ifs = append(ifs, precondition)
		elses = append(elses, etcdclient.OpGet(string(precondition.Key)))
	}
	startTime := time.Now()
	response, err := g.client.KV.Txn(g.ctx).If(ifs...).Then(g.operations...).Else(elses...).Commit()
	metrics.RecordEtcdRequestLatency(g.verb, g.resource, startTime)
	if err != nil {
		return response, err
	}
	if actual, expected := len(response.Responses), len(elses); actual != expected {
		return response, fmt.Errorf("invalid gRPC response from etcd: expected %d responses, got %d", expected, actual)
	}
	for i, guard := range g.guards {
		rangeResponse := response.Responses[i].GetResponseRange()
		if rangeResponse == nil {
			return response, storage.NewInternalErrorf("expected a RangeResponse for operation %d, got %T", i, response.Responses[i])
		}
		if err := guard.Handle((*etcdclient.GetResponse)(rangeResponse)); err != nil {
			return response, err
		}
	}
	return response, nil
}
