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

// ResourceQuotasGetter has a method to return a ResourceQuotaInterface.
// A group's client should implement this interface.
type ResourceQuotasGetter interface {
	ResourceQuotas(namespace string) ResourceQuotaInterface
}

// ResourceQuotaInterface has methods to work with ResourceQuota resources.
type ResourceQuotaInterface interface {
	Create(ctx context.Context, resourceQuota *corev1.ResourceQuota, opts metav1.CreateOptions) (*corev1.ResourceQuota, error)
	Update(ctx context.Context, resourceQuota *corev1.ResourceQuota, opts metav1.UpdateOptions) (*corev1.ResourceQuota, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, resourceQuota *corev1.ResourceQuota, opts metav1.UpdateOptions) (*corev1.ResourceQuota, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.ResourceQuota, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.ResourceQuotaList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.ResourceQuota, err error)
	Apply(ctx context.Context, resourceQuota *applyconfigurationscorev1.ResourceQuotaApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.ResourceQuota, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, resourceQuota *applyconfigurationscorev1.ResourceQuotaApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.ResourceQuota, err error)
	ResourceQuotaExpansion
}

// resourceQuotas implements ResourceQuotaInterface
type resourceQuotas struct {
	*gentype.ClientWithListAndApply[*corev1.ResourceQuota, *corev1.ResourceQuotaList, *applyconfigurationscorev1.ResourceQuotaApplyConfiguration]
}

// newResourceQuotas returns a ResourceQuotas
func newResourceQuotas(c *CoreV1Client, namespace string) *resourceQuotas {
	return &resourceQuotas{
		gentype.NewClientWithListAndApply[*corev1.ResourceQuota, *corev1.ResourceQuotaList, *applyconfigurationscorev1.ResourceQuotaApplyConfiguration](
			"resourcequotas",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *corev1.ResourceQuota { return &corev1.ResourceQuota{} },
			func() *corev1.ResourceQuotaList { return &corev1.ResourceQuotaList{} },
		),
	}
}
