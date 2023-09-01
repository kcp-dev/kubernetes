/*
Copyright 2022 The Kubernetes Authors.

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

package validatingadmissionpolicy

import (
	"context"
	"errors"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/client-go/dynamic"
	admissionregistrationinformers "k8s.io/client-go/informers/admissionregistration/v1alpha1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/component-base/featuregate"

	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

////////////////////////////////////////////////////////////////////////////////
// Plugin Definition
////////////////////////////////////////////////////////////////////////////////

// Definition for CEL admission plugin. This is the entry point into the
// CEL admission control system.
//
// Each plugin is asked to validate every object update.

const (
	// PluginName indicates the name of admission plug-in
	PluginName = "ValidatingAdmissionPolicy"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPlugin()
	})
}

////////////////////////////////////////////////////////////////////////////////
// Plugin Initialization & Dependency Injection
////////////////////////////////////////////////////////////////////////////////

type CELAdmissionPlugin struct {
	*admission.Handler
	evaluator CELPolicyEvaluator

	inspectedFeatureGates bool
	enabled               bool

	// Injected Dependencies
	namespaceInformer                         coreinformers.NamespaceInformer
	validatingAdmissionPoliciesInformer       admissionregistrationinformers.ValidatingAdmissionPolicyInformer
	validatingAdmissionPolicyBindingsInformer admissionregistrationinformers.ValidatingAdmissionPolicyBindingInformer
	client                                    kubernetes.Interface
	restMapper                                meta.RESTMapper
	dynamicClient                             dynamic.Interface
	stopCh                                    <-chan struct{}
}

var _ initializer.WantsExternalKubeInformerFactory = &CELAdmissionPlugin{}
var _ initializer.WantsExternalKubeClientSet = &CELAdmissionPlugin{}
var _ initializer.WantsRESTMapper = &CELAdmissionPlugin{}
var _ initializer.WantsDynamicClient = &CELAdmissionPlugin{}
var _ initializer.WantsDrainedNotification = &CELAdmissionPlugin{}

var _ admission.InitializationValidator = &CELAdmissionPlugin{}
var _ admission.ValidationInterface = &CELAdmissionPlugin{}

func NewPlugin() (*CELAdmissionPlugin, error) {
	return &CELAdmissionPlugin{
		Handler: admission.NewHandler(admission.Connect, admission.Create, admission.Delete, admission.Update),
	}, nil
}

func (c *CELAdmissionPlugin) SetExternalKubeInformerFactory(f informers.SharedInformerFactory) {
	c.namespaceInformer = f.Core().V1().Namespaces()
	c.validatingAdmissionPoliciesInformer = f.Admissionregistration().V1alpha1().ValidatingAdmissionPolicies()
	c.validatingAdmissionPolicyBindingsInformer = f.Admissionregistration().V1alpha1().ValidatingAdmissionPolicyBindings()
}

func (c *CELAdmissionPlugin) SetExternalKubeClientSet(client kubernetes.Interface) {
	c.client = client
}

func (c *CELAdmissionPlugin) SetRESTMapper(mapper meta.RESTMapper) {
	c.restMapper = mapper
}

func (c *CELAdmissionPlugin) SetDynamicClient(client dynamic.Interface) {
	c.dynamicClient = client
}

func (c *CELAdmissionPlugin) SetDrainedNotification(stopCh <-chan struct{}) {
	c.stopCh = stopCh
}

func (c *CELAdmissionPlugin) InspectFeatureGates(featureGates featuregate.FeatureGate) {
	if featureGates.Enabled(features.ValidatingAdmissionPolicy) {
		c.enabled = true
	}
	c.inspectedFeatureGates = true
}

// ValidateInitialization - once clientset and informer factory are provided, creates and starts the admission controller
func (c *CELAdmissionPlugin) ValidateInitialization() error {
	if !c.inspectedFeatureGates {
		return fmt.Errorf("%s did not see feature gates", PluginName)
	}
	if !c.enabled {
		return nil
	}
	if c.namespaceInformer == nil {
		return errors.New("missing namespace informer")
	}
	if c.validatingAdmissionPoliciesInformer == nil {
		return errors.New("missing validating admission policies informer")
	}
	if c.validatingAdmissionPolicyBindingsInformer == nil {
		return errors.New("missing validating admission policy bindings informer")
	}
	if c.client == nil {
		return errors.New("missing kubernetes client")
	}
	if c.restMapper == nil {
		return errors.New("missing rest mapper")
	}
	if c.dynamicClient == nil {
		return errors.New("missing dynamic client")
	}
	if c.stopCh == nil {
		return errors.New("missing stop channel")
	}
	c.evaluator = NewAdmissionController(
		c.namespaceInformer,
		c.validatingAdmissionPoliciesInformer,
		c.validatingAdmissionPolicyBindingsInformer,
		c.client,
		c.restMapper,
		c.dynamicClient,
	)
	if err := c.evaluator.ValidateInitialization(); err != nil {
		return err
	}

	c.SetReadyFunc(c.evaluator.HasSynced)
	go c.evaluator.Run(c.stopCh)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// admission.ValidationInterface
////////////////////////////////////////////////////////////////////////////////

func (c *CELAdmissionPlugin) Handles(operation admission.Operation) bool {
	return true
}

func (c *CELAdmissionPlugin) Validate(
	ctx context.Context,
	a admission.Attributes,
	o admission.ObjectInterfaces,
) (err error) {
	if !c.enabled {
		return nil
	}

	// isPolicyResource determines if an admission.Attributes object is describing
	// the admission of a ValidatingAdmissionPolicy or a ValidatingAdmissionPolicyBinding
	if isPolicyResource(a) {
		return
	}

	if !c.WaitForReady() {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	return c.evaluator.Validate(ctx, a, o)
}

func isPolicyResource(attr admission.Attributes) bool {
	gvk := attr.GetResource()
	if gvk.Group == "admissionregistration.k8s.io" {
		if gvk.Resource == "validatingadmissionpolicies" || gvk.Resource == "validatingadmissionpolicybindings" {
			return true
		}
	}
	return false
}
