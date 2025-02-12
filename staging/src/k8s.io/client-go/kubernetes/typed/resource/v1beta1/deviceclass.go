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

// DeviceClassesGetter has a method to return a DeviceClassInterface.
// A group's client should implement this interface.
type DeviceClassesGetter interface {
	DeviceClasses() DeviceClassInterface
}

// DeviceClassInterface has methods to work with DeviceClass resources.
type DeviceClassInterface interface {
	Create(ctx context.Context, deviceClass *resourcev1beta1.DeviceClass, opts v1.CreateOptions) (*resourcev1beta1.DeviceClass, error)
	Update(ctx context.Context, deviceClass *resourcev1beta1.DeviceClass, opts v1.UpdateOptions) (*resourcev1beta1.DeviceClass, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*resourcev1beta1.DeviceClass, error)
	List(ctx context.Context, opts v1.ListOptions) (*resourcev1beta1.DeviceClassList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *resourcev1beta1.DeviceClass, err error)
	Apply(ctx context.Context, deviceClass *applyconfigurationsresourcev1beta1.DeviceClassApplyConfiguration, opts v1.ApplyOptions) (result *resourcev1beta1.DeviceClass, err error)
	DeviceClassExpansion
}

// deviceClasses implements DeviceClassInterface
type deviceClasses struct {
	*gentype.ClientWithListAndApply[*resourcev1beta1.DeviceClass, *resourcev1beta1.DeviceClassList, *applyconfigurationsresourcev1beta1.DeviceClassApplyConfiguration]
}

// newDeviceClasses returns a DeviceClasses
func newDeviceClasses(c *ResourceV1beta1Client) *deviceClasses {
	return &deviceClasses{
		gentype.NewClientWithListAndApply[*resourcev1beta1.DeviceClass, *resourcev1beta1.DeviceClassList, *applyconfigurationsresourcev1beta1.DeviceClassApplyConfiguration](
			"deviceclasses",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *resourcev1beta1.DeviceClass { return &resourcev1beta1.DeviceClass{} },
			func() *resourcev1beta1.DeviceClassList { return &resourcev1beta1.DeviceClassList{} },
		),
	}
}
