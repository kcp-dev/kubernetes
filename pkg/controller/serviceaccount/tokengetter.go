/*
Copyright 2014 The Kubernetes Authors.

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

// +kcp-code-generator:skip

package serviceaccount

import (
	"context"

	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"
	kcpadmissionregistrationv1listers "github.com/kcp-dev/client-go/listers/admissionregistration/v1"
	kcpcorev1listers "github.com/kcp-dev/client-go/listers/core/v1"
	"github.com/kcp-dev/logicalcluster/v3"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	admissionregistrationv1listers "k8s.io/client-go/listers/admissionregistration/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/serviceaccount"
)

func NewClusterGetterFromClient(client kcpkubernetesclientset.ClusterInterface, secretLister kcpcorev1listers.SecretClusterLister, serviceAccountLister kcpcorev1listers.ServiceAccountClusterLister /*podLister kcpcorev1listers.PodClusterLister*/, validatingWebhookConfigurationLister kcpadmissionregistrationv1listers.ValidatingWebhookConfigurationClusterLister, mutatingWebhookConfigurationLister kcpadmissionregistrationv1listers.MutatingWebhookConfigurationClusterLister) serviceaccount.ServiceAccountTokenClusterGetter {
	return &serviceAccountTokenClusterGetter{
		client:                               client,
		secretLister:                         secretLister,
		serviceAccountLister:                 serviceAccountLister,
		validatingWebhookConfigurationLister: validatingWebhookConfigurationLister,
		mutatingWebhookConfigurationLister:   mutatingWebhookConfigurationLister,
	}
}

type serviceAccountTokenClusterGetter struct {
	client                               kcpkubernetesclientset.ClusterInterface
	secretLister                         kcpcorev1listers.SecretClusterLister
	serviceAccountLister                 kcpcorev1listers.ServiceAccountClusterLister
	podLister                            kcpcorev1listers.PodClusterLister
	validatingWebhookConfigurationLister kcpadmissionregistrationv1listers.ValidatingWebhookConfigurationClusterLister
	mutatingWebhookConfigurationLister   kcpadmissionregistrationv1listers.MutatingWebhookConfigurationClusterLister
}

func (s *serviceAccountTokenClusterGetter) Cluster(name logicalcluster.Name) serviceaccount.ServiceAccountTokenGetter {
	var validatingWebhookConfigurationLister admissionregistrationv1listers.ValidatingWebhookConfigurationLister
	if s.validatingWebhookConfigurationLister != nil {
		validatingWebhookConfigurationLister = s.validatingWebhookConfigurationLister.Cluster(name)
	}
	var mutatingWebhookConfigurationLister admissionregistrationv1listers.MutatingWebhookConfigurationLister
	if s.mutatingWebhookConfigurationLister != nil {
		mutatingWebhookConfigurationLister = s.mutatingWebhookConfigurationLister.Cluster(name)
	}
	return NewGetterFromClient(
		s.client.Cluster(name.Path()),
		s.secretLister.Cluster(name),
		s.serviceAccountLister.Cluster(name),
		validatingWebhookConfigurationLister,
		mutatingWebhookConfigurationLister,
	)
}

// clientGetter implements ServiceAccountTokenGetter using a clientset.Interface
type clientGetter struct {
	client                               clientset.Interface
	secretLister                         v1listers.SecretLister
	serviceAccountLister                 v1listers.ServiceAccountLister
	podLister                            v1listers.PodLister
	nodeLister                           v1listers.NodeLister
	validatingWebhookConfigurationLister admissionregistrationv1listers.ValidatingWebhookConfigurationLister
	mutatingWebhookConfigurationLister   admissionregistrationv1listers.MutatingWebhookConfigurationLister
}

// NewGetterFromClient returns a ServiceAccountTokenGetter that
// uses the specified client to retrieve service accounts, pods, secrets and nodes.
// The client should NOT authenticate using a service account token
// the returned getter will be used to retrieve, or recursion will result.
// The webhook configuration listers may be nil, in which case the respective
// getters fail closed with NotFound.
func NewGetterFromClient(c clientset.Interface, secretLister v1listers.SecretLister, serviceAccountLister v1listers.ServiceAccountLister, validatingWebhookConfigurationLister admissionregistrationv1listers.ValidatingWebhookConfigurationLister, mutatingWebhookConfigurationLister admissionregistrationv1listers.MutatingWebhookConfigurationLister) serviceaccount.ServiceAccountTokenGetter {
	return clientGetter{
		client:                               c,
		secretLister:                         secretLister,
		serviceAccountLister:                 serviceAccountLister,
		validatingWebhookConfigurationLister: validatingWebhookConfigurationLister,
		mutatingWebhookConfigurationLister:   mutatingWebhookConfigurationLister,
	}
}

func (c clientGetter) GetServiceAccount(ctx context.Context, namespace, name string) (*v1.ServiceAccount, error) {
	if serviceAccount, err := c.serviceAccountLister.ServiceAccounts(namespace).Get(name); err == nil {
		return serviceAccount, nil
	}
	return c.client.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c clientGetter) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	if c.podLister == nil {
		return nil, apierrors.NewNotFound(v1.Resource("pods"), name)
	}
	return c.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c clientGetter) GetSecret(ctx context.Context, namespace, name string) (*v1.Secret, error) {
	if secret, err := c.secretLister.Secrets(namespace).Get(name); err == nil {
		return secret, nil
	}
	return c.client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c clientGetter) GetNode(ctx context.Context, name string) (*v1.Node, error) {
	// handle the case where the node lister isn't set due to feature being disabled
	if c.nodeLister == nil {
		return nil, apierrors.NewNotFound(v1.Resource("nodes"), name)
	}
	if node, err := c.nodeLister.Get(name); err == nil {
		return node, nil
	}
	return c.client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

func (c clientGetter) GetValidatingWebhookConfiguration(ctx context.Context, name string) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	// handle the case where no lister is plumbed, e.g. for kcp cluster getters
	if c.validatingWebhookConfigurationLister == nil {
		return nil, apierrors.NewNotFound(admissionregistrationv1.Resource("validatingwebhookconfigurations"), name)
	}
	if validatingWebhookCfg, err := c.validatingWebhookConfigurationLister.Get(name); err == nil {
		return validatingWebhookCfg, nil
	}
	return c.client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
}

func (c clientGetter) GetMutatingWebhookConfiguration(ctx context.Context, name string) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	// handle the case where no lister is plumbed, e.g. for kcp cluster getters
	if c.mutatingWebhookConfigurationLister == nil {
		return nil, apierrors.NewNotFound(admissionregistrationv1.Resource("mutatingwebhookconfigurations"), name)
	}
	if mutatingWebhookCfg, err := c.mutatingWebhookConfigurationLister.Get(name); err == nil {
		return mutatingWebhookCfg, nil
	}
	return c.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
}
