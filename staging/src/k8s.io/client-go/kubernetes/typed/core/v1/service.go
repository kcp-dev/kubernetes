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

// ServicesGetter has a method to return a ServiceInterface.
// A group's client should implement this interface.
type ServicesGetter interface {
	Services(namespace string) ServiceInterface
}

// ServiceInterface has methods to work with Service resources.
type ServiceInterface interface {
	Create(ctx context.Context, service *corev1.Service, opts metav1.CreateOptions) (*corev1.Service, error)
	Update(ctx context.Context, service *corev1.Service, opts metav1.UpdateOptions) (*corev1.Service, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, service *corev1.Service, opts metav1.UpdateOptions) (*corev1.Service, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Service, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.ServiceList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Service, err error)
	Apply(ctx context.Context, service *applyconfigurationscorev1.ServiceApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Service, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, service *applyconfigurationscorev1.ServiceApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Service, err error)
	ServiceExpansion
}

// services implements ServiceInterface
type services struct {
	*gentype.ClientWithListAndApply[*corev1.Service, *corev1.ServiceList, *applyconfigurationscorev1.ServiceApplyConfiguration]
}

// newServices returns a Services
func newServices(c *CoreV1Client, namespace string) *services {
	return &services{
		gentype.NewClientWithListAndApply[*corev1.Service, *corev1.ServiceList, *applyconfigurationscorev1.ServiceApplyConfiguration](
			"services",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *corev1.Service { return &corev1.Service{} },
			func() *corev1.ServiceList { return &corev1.ServiceList{} },
		),
	}
}
