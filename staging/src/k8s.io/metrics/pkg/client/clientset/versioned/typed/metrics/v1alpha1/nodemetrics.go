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

package v1alpha1

import (
	context "context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
	metricsv1alpha1 "k8s.io/metrics/pkg/apis/metrics/v1alpha1"
	scheme "k8s.io/metrics/pkg/client/clientset/versioned/scheme"
)

// NodeMetricsesGetter has a method to return a NodeMetricsInterface.
// A group's client should implement this interface.
type NodeMetricsesGetter interface {
	NodeMetricses() NodeMetricsInterface
}

// NodeMetricsInterface has methods to work with NodeMetrics resources.
type NodeMetricsInterface interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*metricsv1alpha1.NodeMetrics, error)
	List(ctx context.Context, opts v1.ListOptions) (*metricsv1alpha1.NodeMetricsList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	NodeMetricsExpansion
}

// nodeMetricses implements NodeMetricsInterface
type nodeMetricses struct {
	*gentype.ClientWithList[*metricsv1alpha1.NodeMetrics, *metricsv1alpha1.NodeMetricsList]
}

// newNodeMetricses returns a NodeMetricses
func newNodeMetricses(c *MetricsV1alpha1Client) *nodeMetricses {
	return &nodeMetricses{
		gentype.NewClientWithList[*metricsv1alpha1.NodeMetrics, *metricsv1alpha1.NodeMetricsList](
			"nodes",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *metricsv1alpha1.NodeMetrics { return &metricsv1alpha1.NodeMetrics{} },
			func() *metricsv1alpha1.NodeMetricsList { return &metricsv1alpha1.NodeMetricsList{} },
		),
	}
}
