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

package v1beta1

import (
	context "context"

	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	applyconfigurationsresourcev1beta1 "k8s.io/client-go/applyconfigurations/resource/v1beta1"
	gentype "k8s.io/client-go/gentype"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

// ResourceSlicesGetter has a method to return a ResourceSliceInterface.
// A group's client should implement this interface.
type ResourceSlicesGetter interface {
	ResourceSlices() ResourceSliceInterface
}

// ResourceSliceInterface has methods to work with ResourceSlice resources.
type ResourceSliceInterface interface {
	Create(ctx context.Context, resourceSlice *resourcev1beta1.ResourceSlice, opts v1.CreateOptions) (*resourcev1beta1.ResourceSlice, error)
	Update(ctx context.Context, resourceSlice *resourcev1beta1.ResourceSlice, opts v1.UpdateOptions) (*resourcev1beta1.ResourceSlice, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*resourcev1beta1.ResourceSlice, error)
	List(ctx context.Context, opts v1.ListOptions) (*resourcev1beta1.ResourceSliceList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *resourcev1beta1.ResourceSlice, err error)
	Apply(ctx context.Context, resourceSlice *applyconfigurationsresourcev1beta1.ResourceSliceApplyConfiguration, opts v1.ApplyOptions) (result *resourcev1beta1.ResourceSlice, err error)
	ResourceSliceExpansion
}

// resourceSlices implements ResourceSliceInterface
type resourceSlices struct {
	*gentype.ClientWithListAndApply[*resourcev1beta1.ResourceSlice, *resourcev1beta1.ResourceSliceList, *applyconfigurationsresourcev1beta1.ResourceSliceApplyConfiguration]
}

// newResourceSlices returns a ResourceSlices
func newResourceSlices(c *ResourceV1beta1Client) *resourceSlices {
	return &resourceSlices{
		gentype.NewClientWithListAndApply[*resourcev1beta1.ResourceSlice, *resourcev1beta1.ResourceSliceList, *applyconfigurationsresourcev1beta1.ResourceSliceApplyConfiguration](
			"resourceslices",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *resourcev1beta1.ResourceSlice { return &resourcev1beta1.ResourceSlice{} },
			func() *resourcev1beta1.ResourceSliceList { return &resourcev1beta1.ResourceSliceList{} },
		),
	}
}
