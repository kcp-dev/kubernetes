/*
Copyright The KCP Authors.

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

// Code generated by kcp code-generator. DO NOT EDIT.

package autoscaling

import (
	"github.com/kcp-dev/client-go/informers/autoscaling/v1"
	"github.com/kcp-dev/client-go/informers/autoscaling/v2"
	"github.com/kcp-dev/client-go/informers/autoscaling/v2beta1"
	"github.com/kcp-dev/client-go/informers/autoscaling/v2beta2"
	"github.com/kcp-dev/client-go/informers/internalinterfaces"
)

type ClusterInterface interface {
	// V1 provides access to the shared informers in V1.
	V1() v1.ClusterInterface
	// V2 provides access to the shared informers in V2.
	V2() v2.ClusterInterface
	// V2beta1 provides access to the shared informers in V2beta1.
	V2beta1() v2beta1.ClusterInterface
	// V2beta2 provides access to the shared informers in V2beta2.
	V2beta2() v2beta2.ClusterInterface
}

type group struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new ClusterInterface.
func New(f internalinterfaces.SharedInformerFactory, tweakListOptions internalinterfaces.TweakListOptionsFunc) ClusterInterface {
	return &group{factory: f, tweakListOptions: tweakListOptions}
}

// V1 returns a new v1.ClusterInterface.
func (g *group) V1() v1.ClusterInterface {
	return v1.New(g.factory, g.tweakListOptions)
}

// V2 returns a new v2.ClusterInterface.
func (g *group) V2() v2.ClusterInterface {
	return v2.New(g.factory, g.tweakListOptions)
}

// V2beta1 returns a new v2beta1.ClusterInterface.
func (g *group) V2beta1() v2beta1.ClusterInterface {
	return v2beta1.New(g.factory, g.tweakListOptions)
}

// V2beta2 returns a new v2beta2.ClusterInterface.
func (g *group) V2beta2() v2beta2.ClusterInterface {
	return v2beta2.New(g.factory, g.tweakListOptions)
}
