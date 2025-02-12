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

package v1beta1

import (
	"github.com/kcp-dev/client-go/informers/internalinterfaces"
)

type ClusterInterface interface {
	// Roles returns a RoleClusterInformer
	Roles() RoleClusterInformer
	// RoleBindings returns a RoleBindingClusterInformer
	RoleBindings() RoleBindingClusterInformer
	// ClusterRoles returns a ClusterRoleClusterInformer
	ClusterRoles() ClusterRoleClusterInformer
	// ClusterRoleBindings returns a ClusterRoleBindingClusterInformer
	ClusterRoleBindings() ClusterRoleBindingClusterInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new ClusterInterface.
func New(f internalinterfaces.SharedInformerFactory, tweakListOptions internalinterfaces.TweakListOptionsFunc) ClusterInterface {
	return &version{factory: f, tweakListOptions: tweakListOptions}
}

// Roles returns a RoleClusterInformer
func (v *version) Roles() RoleClusterInformer {
	return &roleClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// RoleBindings returns a RoleBindingClusterInformer
func (v *version) RoleBindings() RoleBindingClusterInformer {
	return &roleBindingClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterRoles returns a ClusterRoleClusterInformer
func (v *version) ClusterRoles() ClusterRoleClusterInformer {
	return &clusterRoleClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ClusterRoleBindings returns a ClusterRoleBindingClusterInformer
func (v *version) ClusterRoleBindings() ClusterRoleBindingClusterInformer {
	return &clusterRoleBindingClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
