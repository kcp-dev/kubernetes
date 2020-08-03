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

package genericcontrolplane

import (
	"errors"
	"fmt"

	"k8s.io/apiserver/pkg/server/egressselector"
	openapicommon "k8s.io/kube-openapi/pkg/common"

	"k8s.io/kubernetes/pkg/kubeapiserver/options"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

// AuthenticationApplyTo requires already applied OpenAPIConfig and EgressSelector if present.
func AuthenticationApplyTo(o *options.BuiltInAuthenticationOptions, authInfo *genericapiserver.AuthenticationInfo, secureServing *genericapiserver.SecureServingInfo, egressSelector *egressselector.EgressSelector, openAPIConfig *openapicommon.Config) error {
	if o == nil {
		return nil
	}

	if openAPIConfig == nil {
		return errors.New("uninitialized OpenAPIConfig")
	}

	authenticatorConfig, err := o.ToAuthenticationConfig()
	if err != nil {
		return err
	}

	if authenticatorConfig.ClientCAContentProvider != nil {
		if err = authInfo.ApplyClientCert(authenticatorConfig.ClientCAContentProvider, secureServing); err != nil {
			return fmt.Errorf("unable to load client CA file: %v", err)
		}
	}
	if authenticatorConfig.RequestHeaderConfig != nil && authenticatorConfig.RequestHeaderConfig.CAContentProvider != nil {
		if err = authInfo.ApplyClientCert(authenticatorConfig.RequestHeaderConfig.CAContentProvider, secureServing); err != nil {
			return fmt.Errorf("unable to load client CA file: %v", err)
		}
	}

	authInfo.APIAudiences = o.APIAudiences
	// if o.ServiceAccounts != nil && len(o.ServiceAccounts.Issuers) != 0 && len(o.APIAudiences) == 0 {
	// 	authInfo.APIAudiences = authenticator.Audiences(o.ServiceAccounts.Issuers)
	// }

	// authenticatorConfig.ServiceAccountTokenGetter = serviceaccountcontroller.NewGetterFromClient(
	// 	extclient,
	// 	versionedInformer.Core().V1().Secrets().Lister(),
	// 	versionedInformer.Core().V1().ServiceAccounts().Lister(),
	// 	versionedInformer.Core().V1().Pods().Lister(),
	// )

	// authenticatorConfig.BootstrapTokenAuthenticator = bootstrap.NewTokenAuthenticator(
	// 	versionedInformer.Core().V1().Secrets().Lister().Secrets(metav1.NamespaceSystem),
	// )

	if egressSelector != nil {
		egressDialer, err := egressSelector.Lookup(egressselector.ControlPlane.AsNetworkContext())
		if err != nil {
			return err
		}
		authenticatorConfig.CustomDial = egressDialer
	}

	authInfo.Authenticator, openAPIConfig.SecurityDefinitions, err = authenticatorConfig.New()
	if err != nil {
		return err
	}

	return nil
}
