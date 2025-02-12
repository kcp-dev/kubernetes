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

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	applyconfigurationscoordinationv1 "k8s.io/client-go/applyconfigurations/coordination/v1"
	gentype "k8s.io/client-go/gentype"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

// LeasesGetter has a method to return a LeaseInterface.
// A group's client should implement this interface.
type LeasesGetter interface {
	Leases(namespace string) LeaseInterface
}

// LeaseInterface has methods to work with Lease resources.
type LeaseInterface interface {
	Create(ctx context.Context, lease *coordinationv1.Lease, opts metav1.CreateOptions) (*coordinationv1.Lease, error)
	Update(ctx context.Context, lease *coordinationv1.Lease, opts metav1.UpdateOptions) (*coordinationv1.Lease, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*coordinationv1.Lease, error)
	List(ctx context.Context, opts metav1.ListOptions) (*coordinationv1.LeaseList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *coordinationv1.Lease, err error)
	Apply(ctx context.Context, lease *applyconfigurationscoordinationv1.LeaseApplyConfiguration, opts metav1.ApplyOptions) (result *coordinationv1.Lease, err error)
	LeaseExpansion
}

// leases implements LeaseInterface
type leases struct {
	*gentype.ClientWithListAndApply[*coordinationv1.Lease, *coordinationv1.LeaseList, *applyconfigurationscoordinationv1.LeaseApplyConfiguration]
}

// newLeases returns a Leases
func newLeases(c *CoordinationV1Client, namespace string) *leases {
	return &leases{
		gentype.NewClientWithListAndApply[*coordinationv1.Lease, *coordinationv1.LeaseList, *applyconfigurationscoordinationv1.LeaseApplyConfiguration](
			"leases",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *coordinationv1.Lease { return &coordinationv1.Lease{} },
			func() *coordinationv1.LeaseList { return &coordinationv1.LeaseList{} },
		),
	}
}
