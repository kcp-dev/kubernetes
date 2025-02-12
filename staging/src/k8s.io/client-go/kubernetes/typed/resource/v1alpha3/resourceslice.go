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

package v1alpha3

import (
	context "context"

	resourcev1alpha3 "k8s.io/api/resource/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	applyconfigurationsresourcev1alpha3 "k8s.io/client-go/applyconfigurations/resource/v1alpha3"
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
	Create(ctx context.Context, resourceSlice *resourcev1alpha3.ResourceSlice, opts v1.CreateOptions) (*resourcev1alpha3.ResourceSlice, error)
	Update(ctx context.Context, resourceSlice *resourcev1alpha3.ResourceSlice, opts v1.UpdateOptions) (*resourcev1alpha3.ResourceSlice, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*resourcev1alpha3.ResourceSlice, error)
	List(ctx context.Context, opts v1.ListOptions) (*resourcev1alpha3.ResourceSliceList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *resourcev1alpha3.ResourceSlice, err error)
	Apply(ctx context.Context, resourceSlice *applyconfigurationsresourcev1alpha3.ResourceSliceApplyConfiguration, opts v1.ApplyOptions) (result *resourcev1alpha3.ResourceSlice, err error)
	ResourceSliceExpansion
}

// resourceSlices implements ResourceSliceInterface
type resourceSlices struct {
	*gentype.ClientWithListAndApply[*resourcev1alpha3.ResourceSlice, *resourcev1alpha3.ResourceSliceList, *applyconfigurationsresourcev1alpha3.ResourceSliceApplyConfiguration]
}

// newResourceSlices returns a ResourceSlices
func newResourceSlices(c *ResourceV1alpha3Client) *resourceSlices {
	return &resourceSlices{
		gentype.NewClientWithListAndApply[*resourcev1alpha3.ResourceSlice, *resourcev1alpha3.ResourceSliceList, *applyconfigurationsresourcev1alpha3.ResourceSliceApplyConfiguration](
			"resourceslices",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *resourcev1alpha3.ResourceSlice { return &resourcev1alpha3.ResourceSlice{} },
			func() *resourcev1alpha3.ResourceSliceList { return &resourcev1alpha3.ResourceSliceList{} },
		),
	}
}
