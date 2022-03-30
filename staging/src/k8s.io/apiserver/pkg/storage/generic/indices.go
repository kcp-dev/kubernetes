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

package generic

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
)

type contextKey int

// indexFieldsContextKey is used to store index matchers in the request context
const indexFieldsContextKey contextKey = iota

func IndexFieldsFromContext(ctx context.Context) ([]storage.MatchValue, bool) {
	indexFields, ok := ctx.Value(indexFieldsContextKey).([]storage.MatchValue)
	return indexFields, ok
}

func ContextWithIndexFields(ctx context.Context, indexFields []storage.MatchValue) context.Context {
	return context.WithValue(ctx, indexFieldsContextKey, indexFields)
}

func ContextWithComputedIndexFields(ctx context.Context, obj runtime.Object, indexers storage.IndexerFuncs) context.Context {
	var indexFields []storage.MatchValue
	for index, indexFunc := range indexers {
		indexFields = append(indexFields, storage.MatchValue{
			IndexName: storage.FieldIndex(index),
			Value:     indexFunc(obj),
		})
	}
	return context.WithValue(ctx, indexFieldsContextKey, indexFields)
}