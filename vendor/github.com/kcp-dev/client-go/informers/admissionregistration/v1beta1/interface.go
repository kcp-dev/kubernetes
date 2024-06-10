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

package v1beta1

import (
	"github.com/kcp-dev/client-go/informers/internalinterfaces"
)

type ClusterInterface interface {
	// ValidatingAdmissionPolicies returns a ValidatingAdmissionPolicyClusterInformer
	ValidatingAdmissionPolicies() ValidatingAdmissionPolicyClusterInformer
	// ValidatingAdmissionPolicyBindings returns a ValidatingAdmissionPolicyBindingClusterInformer
	ValidatingAdmissionPolicyBindings() ValidatingAdmissionPolicyBindingClusterInformer
	// ValidatingWebhookConfigurations returns a ValidatingWebhookConfigurationClusterInformer
	ValidatingWebhookConfigurations() ValidatingWebhookConfigurationClusterInformer
	// MutatingWebhookConfigurations returns a MutatingWebhookConfigurationClusterInformer
	MutatingWebhookConfigurations() MutatingWebhookConfigurationClusterInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new ClusterInterface.
func New(f internalinterfaces.SharedInformerFactory, tweakListOptions internalinterfaces.TweakListOptionsFunc) ClusterInterface {
	return &version{factory: f, tweakListOptions: tweakListOptions}
}

// ValidatingAdmissionPolicies returns a ValidatingAdmissionPolicyClusterInformer
func (v *version) ValidatingAdmissionPolicies() ValidatingAdmissionPolicyClusterInformer {
	return &validatingAdmissionPolicyClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ValidatingAdmissionPolicyBindings returns a ValidatingAdmissionPolicyBindingClusterInformer
func (v *version) ValidatingAdmissionPolicyBindings() ValidatingAdmissionPolicyBindingClusterInformer {
	return &validatingAdmissionPolicyBindingClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ValidatingWebhookConfigurations returns a ValidatingWebhookConfigurationClusterInformer
func (v *version) ValidatingWebhookConfigurations() ValidatingWebhookConfigurationClusterInformer {
	return &validatingWebhookConfigurationClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// MutatingWebhookConfigurations returns a MutatingWebhookConfigurationClusterInformer
func (v *version) MutatingWebhookConfigurations() MutatingWebhookConfigurationClusterInformer {
	return &mutatingWebhookConfigurationClusterInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
