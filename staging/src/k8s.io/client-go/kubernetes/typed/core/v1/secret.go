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

package v1

import (
	context "context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	gentype "k8s.io/client-go/gentype"
	scheme "k8s.io/client-go/kubernetes/scheme"
)

// SecretsGetter has a method to return a SecretInterface.
// A group's client should implement this interface.
type SecretsGetter interface {
	Secrets(namespace string) SecretInterface
}

// SecretInterface has methods to work with Secret resources.
type SecretInterface interface {
	Create(ctx context.Context, secret *corev1.Secret, opts metav1.CreateOptions) (*corev1.Secret, error)
	Update(ctx context.Context, secret *corev1.Secret, opts metav1.UpdateOptions) (*corev1.Secret, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Secret, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.SecretList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *corev1.Secret, err error)
	Apply(ctx context.Context, secret *applyconfigurationscorev1.SecretApplyConfiguration, opts metav1.ApplyOptions) (result *corev1.Secret, err error)
	SecretExpansion
}

// secrets implements SecretInterface
type secrets struct {
	*gentype.ClientWithListAndApply[*corev1.Secret, *corev1.SecretList, *applyconfigurationscorev1.SecretApplyConfiguration]
}

// newSecrets returns a Secrets
func newSecrets(c *CoreV1Client, namespace string) *secrets {
	return &secrets{
		gentype.NewClientWithListAndApply[*corev1.Secret, *corev1.SecretList, *applyconfigurationscorev1.SecretApplyConfiguration](
			"secrets",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *corev1.Secret { return &corev1.Secret{} },
			func() *corev1.SecretList { return &corev1.SecretList{} },
		),
	}
}
