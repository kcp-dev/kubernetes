package validatingadmissionpolicy

import (
	admissionregistrationinformers "k8s.io/client-go/informers/admissionregistration/v1alpha1"
	coreinformers "k8s.io/client-go/informers/core/v1"
)

func (c *CELAdmissionPlugin) SetNamespaceInformer(i coreinformers.NamespaceInformer) {
	c.namespaceInformer = i
}

func (c *CELAdmissionPlugin) SetValidatingAdmissionPoliciesInformer(i admissionregistrationinformers.ValidatingAdmissionPolicyInformer) {
	c.validatingAdmissionPoliciesInformer = i
}

func (c *CELAdmissionPlugin) SetValidatingAdmissionPolicyBindingsInformer(i admissionregistrationinformers.ValidatingAdmissionPolicyBindingInformer) {
	c.validatingAdmissionPolicyBindingsInformer = i
}
