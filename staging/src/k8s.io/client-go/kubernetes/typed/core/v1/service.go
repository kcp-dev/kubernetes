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
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	scheme "k8s.io/client-go/kubernetes/scheme"
	rest "k8s.io/client-go/rest"
)

// ServicesGetter has a method to return a ServiceInterface.
// A group's client should implement this interface.
type ServicesGetter interface {
	Services(namespace string) ServiceInterface
}

// ServiceInterface has methods to work with Service resources.
type ServiceInterface interface {
	Create(ctx context.Context, service *v1.Service, opts metav1.CreateOptions) (*v1.Service, error)
	Update(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (*v1.Service, error)
	UpdateStatus(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (*v1.Service, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Service, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ServiceList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Service, err error)
	Apply(ctx context.Context, service *corev1.ServiceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Service, err error)
	ApplyStatus(ctx context.Context, service *corev1.ServiceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Service, err error)
	ServiceExpansion
}

// services implements ServiceInterface
type services struct {
	client  rest.Interface
	cluster string
	ns      string
}

// newServices returns a Services
func newServices(c *CoreV1Client, namespace string) *services {
	return &services{
		client:  c.RESTClient(),
		cluster: c.cluster,
		ns:      namespace,
	}
}

// Get takes name of the service, and returns the corresponding service object, and an error if there is any.
func (c *services) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Service, err error) {
	result = &v1.Service{}
	err = c.client.Get().
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Services that match those selectors.
func (c *services) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ServiceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ServiceList{}
	err = c.client.Get().
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested services.
func (c *services) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("services").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a service and creates it.  Returns the server's representation of the service, and an error, if there is any.
func (c *services) Create(ctx context.Context, service *v1.Service, opts metav1.CreateOptions) (result *v1.Service, err error) {
	result = &v1.Service{}
	err = c.client.Post().
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(service).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a service and updates it. Returns the server's representation of the service, and an error, if there is any.
func (c *services) Update(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (result *v1.Service, err error) {
	result = &v1.Service{}
	err = c.client.Put().
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		Name(service.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(service).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *services) UpdateStatus(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (result *v1.Service, err error) {
	result = &v1.Service{}
	err = c.client.Put().
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		Name(service.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(service).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the service and deletes it. Returns an error if one occurs.
func (c *services) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched service.
func (c *services) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Service, err error) {
	result = &v1.Service{}
	err = c.client.Patch(pt).
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied service.
func (c *services) Apply(ctx context.Context, service *corev1.ServiceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Service, err error) {
	if service == nil {
		return nil, fmt.Errorf("service provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(service)
	if err != nil {
		return nil, err
	}
	name := service.Name
	if name == nil {
		return nil, fmt.Errorf("service.Name must be provided to Apply")
	}
	result = &v1.Service{}
	err = c.client.Patch(types.ApplyPatchType).
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *services) ApplyStatus(ctx context.Context, service *corev1.ServiceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Service, err error) {
	if service == nil {
		return nil, fmt.Errorf("service provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(service)
	if err != nil {
		return nil, err
	}

	name := service.Name
	if name == nil {
		return nil, fmt.Errorf("service.Name must be provided to Apply")
	}

	result = &v1.Service{}
	err = c.client.Patch(types.ApplyPatchType).
		Cluster(c.cluster).
		Namespace(c.ns).
		Resource("services").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
