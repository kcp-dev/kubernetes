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

package v2beta2

import (
	context "context"

	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	applyconfigurationsautoscalingv2beta2 "k8s.io/client-go/applyconfigurations/autoscaling/v2beta2"
	gentype "k8s.io/client-go/gentype"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

// HorizontalPodAutoscalersGetter has a method to return a HorizontalPodAutoscalerInterface.
// A group's client should implement this interface.
type HorizontalPodAutoscalersGetter interface {
	HorizontalPodAutoscalers(namespace string) HorizontalPodAutoscalerInterface
}

// HorizontalPodAutoscalerInterface has methods to work with HorizontalPodAutoscaler resources.
type HorizontalPodAutoscalerInterface interface {
	Create(ctx context.Context, horizontalPodAutoscaler *autoscalingv2beta2.HorizontalPodAutoscaler, opts v1.CreateOptions) (*autoscalingv2beta2.HorizontalPodAutoscaler, error)
	Update(ctx context.Context, horizontalPodAutoscaler *autoscalingv2beta2.HorizontalPodAutoscaler, opts v1.UpdateOptions) (*autoscalingv2beta2.HorizontalPodAutoscaler, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, horizontalPodAutoscaler *autoscalingv2beta2.HorizontalPodAutoscaler, opts v1.UpdateOptions) (*autoscalingv2beta2.HorizontalPodAutoscaler, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*autoscalingv2beta2.HorizontalPodAutoscaler, error)
	List(ctx context.Context, opts v1.ListOptions) (*autoscalingv2beta2.HorizontalPodAutoscalerList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *autoscalingv2beta2.HorizontalPodAutoscaler, err error)
	Apply(ctx context.Context, horizontalPodAutoscaler *applyconfigurationsautoscalingv2beta2.HorizontalPodAutoscalerApplyConfiguration, opts v1.ApplyOptions) (result *autoscalingv2beta2.HorizontalPodAutoscaler, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, horizontalPodAutoscaler *applyconfigurationsautoscalingv2beta2.HorizontalPodAutoscalerApplyConfiguration, opts v1.ApplyOptions) (result *autoscalingv2beta2.HorizontalPodAutoscaler, err error)
	HorizontalPodAutoscalerExpansion
}

// horizontalPodAutoscalers implements HorizontalPodAutoscalerInterface
type horizontalPodAutoscalers struct {
	*gentype.ClientWithListAndApply[*autoscalingv2beta2.HorizontalPodAutoscaler, *autoscalingv2beta2.HorizontalPodAutoscalerList, *applyconfigurationsautoscalingv2beta2.HorizontalPodAutoscalerApplyConfiguration]
}

// newHorizontalPodAutoscalers returns a HorizontalPodAutoscalers
func newHorizontalPodAutoscalers(c *AutoscalingV2beta2Client, namespace string) *horizontalPodAutoscalers {
	return &horizontalPodAutoscalers{
		gentype.NewClientWithListAndApply[*autoscalingv2beta2.HorizontalPodAutoscaler, *autoscalingv2beta2.HorizontalPodAutoscalerList, *applyconfigurationsautoscalingv2beta2.HorizontalPodAutoscalerApplyConfiguration](
			"horizontalpodautoscalers",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *autoscalingv2beta2.HorizontalPodAutoscaler {
				return &autoscalingv2beta2.HorizontalPodAutoscaler{}
			},
			func() *autoscalingv2beta2.HorizontalPodAutoscalerList {
				return &autoscalingv2beta2.HorizontalPodAutoscalerList{}
			},
		),
	}
}
