//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

package resource

import (
	"github.com/kcp-dev/client-go/informers/internalinterfaces"
	"github.com/kcp-dev/client-go/informers/resource/v1alpha2"
)

type ClusterInterface interface {
	// V1alpha2 provides access to the shared informers in V1alpha2.
	V1alpha2() v1alpha2.ClusterInterface
}

type group struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new ClusterInterface.
func New(f internalinterfaces.SharedInformerFactory, tweakListOptions internalinterfaces.TweakListOptionsFunc) ClusterInterface {
	return &group{factory: f, tweakListOptions: tweakListOptions}
}

// V1alpha2 returns a new v1alpha2.ClusterInterface.
func (g *group) V1alpha2() v1alpha2.ClusterInterface {
	return v1alpha2.New(g.factory, g.tweakListOptions)
}
