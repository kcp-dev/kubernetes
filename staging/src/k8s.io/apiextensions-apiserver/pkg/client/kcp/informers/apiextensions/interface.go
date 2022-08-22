//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by kcp code-generator. DO NOT EDIT.

package apiextensions

import (
	internalinterfaces "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions/internalinterfaces"
	v1 "k8s.io/apiextensions-apiserver/pkg/client/kcp/informers/apiextensions/v1"
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/client/kcp/informers/apiextensions/v1beta1"
)

type Interface struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return Interface{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// V1 returns a new v1.Interface.
func (g Interface) V1() v1.Interface {
	return v1.New(g.factory, g.namespace, g.tweakListOptions)
}

// V1beta1 returns a new v1beta1.Interface.
func (g Interface) V1beta1() v1beta1.Interface {
	return v1beta1.New(g.factory, g.namespace, g.tweakListOptions)
}
