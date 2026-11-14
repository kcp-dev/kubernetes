/*
Copyright 2026 The KCP Authors.

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

package handlers

import (
	"context"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func newWidget(annotations map[string]string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("example.io/v1")
	obj.SetKind("Widget")
	obj.SetName("w")
	obj.SetAnnotations(annotations)
	return obj
}

func TestSetKCPOriginalAPIVersionAnnotation(t *testing.T) {
	wildcard := request.WithCluster(context.Background(), request.Cluster{Wildcard: true})

	t.Run("sets the annotation without changing the original object", func(t *testing.T) {
		original := &metav1.ObjectMeta{Name: "w", Annotations: map[string]string{"existing": "value"}}
		partial := meta.AsPartialObjectMetadata(original)
		typed := &metav1.PartialObjectMetadata{TypeMeta: metav1.TypeMeta{APIVersion: "example.io/v1", Kind: "Widget"}}

		setKCPOriginalAPIVersionAnnotation(wildcard, typed, partial)

		if got := partial.Annotations[KCPOriginalAPIVersionAnnotation]; got != "example.io/v1" {
			t.Errorf("expected annotation example.io/v1 on the partial object, got %q", got)
		}
		if partial.Annotations["existing"] != "value" {
			t.Errorf("expected existing annotations to be kept, got %v", partial.Annotations)
		}
		if _, found := original.Annotations[KCPOriginalAPIVersionAnnotation]; found {
			t.Errorf("the original object's annotations must not be modified, got %v", original.Annotations)
		}
	})

	t.Run("keeps an annotation set by the converter", func(t *testing.T) {
		original := newWidget(map[string]string{KCPOriginalAPIVersionAnnotation: "example.io/v2"})
		partial := meta.AsPartialObjectMetadata(original)

		setKCPOriginalAPIVersionAnnotation(wildcard, original, partial)

		if got := partial.Annotations[KCPOriginalAPIVersionAnnotation]; got != "example.io/v2" {
			t.Errorf("expected the existing annotation to be kept, got %q", got)
		}
	})

	t.Run("does nothing for a request to a single cluster", func(t *testing.T) {
		ctx := request.WithCluster(context.Background(), request.Cluster{Name: "root"})
		original := newWidget(nil)
		partial := meta.AsPartialObjectMetadata(original)

		setKCPOriginalAPIVersionAnnotation(ctx, original, partial)

		if _, found := partial.Annotations[KCPOriginalAPIVersionAnnotation]; found {
			t.Errorf("expected no annotation, got %v", partial.Annotations)
		}
	})

	// Concurrent wildcard requests convert the same cached object. Run with -race, this used
	// to fail with "concurrent map writes" because every request wrote into the cached map.
	t.Run("concurrent conversions of a shared object", func(t *testing.T) {
		original := newWidget(map[string]string{"existing": "value"})
		annotations := original.GetAnnotations()
		objectMeta := &metav1.ObjectMeta{Name: "w", Annotations: annotations}

		var wg sync.WaitGroup
		for range 16 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				partial := &metav1.PartialObjectMetadata{ObjectMeta: *objectMeta}
				setKCPOriginalAPIVersionAnnotation(wildcard, original, partial)
			}()
		}
		wg.Wait()

		if _, found := annotations[KCPOriginalAPIVersionAnnotation]; found {
			t.Errorf("the shared annotations map must not be modified, got %v", annotations)
		}
	})
}
