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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/apis/example"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
)

type IndexedClientFactory func(indexers storage.IndexerFuncs) InternalTestClient

func RunTestListUsingSecondaryIndex(t *testing.T, clientFactory IndexedClientFactory) {
	defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.RemainingItemCount, true)()
	indexers := map[string]storage.IndexerFunc{
		"metadata.name": func(obj runtime.Object) string {
			return obj.(*example.Pod).ObjectMeta.Name
		},
		"spec.nodeName": func(obj runtime.Object) string {
			return obj.(*example.Pod).Spec.NodeName
		},
		"spec.hostname": func(obj runtime.Object) string {
			return obj.(*example.Pod).Spec.Hostname
		},
	}
	ctx, store := testSetup(clientFactory(indexers))
	store.indexers = indexers

	objects := []struct {
		key                  string
		object, storedObject *example.Pod
	}{
		{
			key:    "/some/prefix/foo-1",
			object: &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo-1"}, Spec: example.PodSpec{NodeName: "first"}},
		},
		{
			key:    "/some/prefix/foo-2",
			object: &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo-2"}, Spec: example.PodSpec{NodeName: "second"}},
		},
		{
			key:    "/some/prefix/foo-3",
			object: &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo-3"}, Spec: example.PodSpec{NodeName: "first", Hostname: "localhost"}},
		},
		{
			key:    "/some/prefix/foo-4",
			object: &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo-4"}, Spec: example.PodSpec{NodeName: "second", Hostname: "localhost"}},
		},
	}

	for i, obj := range objects {
		objects[i].storedObject = &example.Pod{}
		err := store.Create(ctx, obj.key, obj.object, objects[i].storedObject, 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	for _, testCase := range []struct {
		name     string
		selector fields.Selector
		fieldSet fields.Set
		expected []example.Pod
	}{
		{
			name: "selecting on one index",
			fieldSet: fields.Set{"spec.nodeName": "first"},
			expected: []example.Pod{*objects[0].storedObject, *objects[2].storedObject},
		},
		{
			name: "selecting on a different value for the index",
			fieldSet: fields.Set{"spec.nodeName": "second"},
			expected: []example.Pod{*objects[1].storedObject, *objects[3].storedObject},
		},
		{
			name: "selecting on a non-matching value for the index",
			fieldSet: fields.Set{"spec.nodeName": "other"},
		},
		{
			name: "selecting on a different index",
			fieldSet: fields.Set{"spec.hostname": "localhost"},
			expected: []example.Pod{*objects[2].storedObject, *objects[3].storedObject},
		},
		{
			name: "selecting on an intersection of two indices",
			fieldSet: fields.Set{"spec.nodeName": "first", "spec.hostname": "localhost"},
			expected: []example.Pod{*objects[2].storedObject},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			storageOpts := storage.ListOptions{
				ResourceVersion: "0",
				Predicate: storage.SelectionPredicate{
					Label: labels.Everything(),
					Field: &matchingSelector{Selector: fields.SelectorFromSet(testCase.fieldSet)},
					GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
						f := fields.Set{}
						for index, indexFunc := range store.indexers {
							f[index] = indexFunc(obj)
						}
						return nil, f, nil
					},
					IndexFields: []string{"spec.nodeName", "spec.hostname"},
				},
				Recursive: true,
			}

			list := &example.PodList{}
			if err := store.GetList(ctx, "/some/prefix", storageOpts, list); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			expectNoDiff(t, "invalid data returned", testCase.expected, list.Items)
		})
	}
}

// matchingSelector matches everything, ensuring that selectivity in a query was due to indexing at the storage layer
type matchingSelector struct {
	fields.Selector
}

func (s *matchingSelector) Matches(_ fields.Fields) bool {
	return true
}
