/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	context "context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	gentype "k8s.io/client-go/gentype"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

// EventsGetter has a method to return a EventInterface.
// A group's client should implement this interface.
type EventsGetter interface {
	Events(namespace string) EventInterface
}

// EventInterface has methods to work with Event resources.
type EventInterface interface {
	Create(ctx context.Context, event *corev1.Event, opts metav1.CreateOptions) (*corev1.Event, error)
	Update(ctx context.Context, event *corev1.Event, opts metav1.UpdateOptions) (*corev1.Event, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Event, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Event, err error)
	Apply(ctx context.Context, event *applyconfigurationscorev1.EventApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Event, err error)
	EventExpansion
}

// events implements EventInterface
type events struct {
	*gentype.ClientWithListAndApply[*corev1.Event, *corev1.EventList, *applyconfigurationscorev1.EventApplyConfiguration]
}

// newEvents returns a Events
func newEvents(c *CoreV1Client, namespace string) *events {
	return &events{
		gentype.NewClientWithListAndApply[*corev1.Event, *corev1.EventList, *applyconfigurationscorev1.EventApplyConfiguration](
			"events",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *corev1.Event { return &corev1.Event{} },
			func() *corev1.EventList { return &corev1.EventList{} },
		),
	}
}
