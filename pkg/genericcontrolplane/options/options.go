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

package options

import (
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/apiserver/pkg/admission/plugin/webhook/mutating"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/validating"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/metrics"
	"k8s.io/klog/v2"

	kubeoptions "k8s.io/kubernetes/pkg/kubeapiserver/options"
	"k8s.io/kubernetes/pkg/serviceaccount"
)

// InsecurePortFlags are dummy flags, they are kept only for compatibility and will be removed in v1.24.
// TODO: remove these flags in v1.24.
var InsecurePortFlags = []string{"insecure-port", "port"}

// ServerRunOptions runs a kubernetes api server.
type ServerRunOptions struct {
	GenericServerRunOptions *genericoptions.ServerRunOptions
	Etcd                    *genericoptions.EtcdOptions
	SecureServing           *genericoptions.SecureServingOptionsWithLoopback
	Audit                   *genericoptions.AuditOptions
	Features                *genericoptions.FeatureOptions
	Admission               *genericoptions.AdmissionOptions
	Authentication          *kubeoptions.BuiltInAuthenticationOptions
	APIEnablement           *genericoptions.APIEnablementOptions
	EgressSelector          *genericoptions.EgressSelectorOptions
	Metrics                 *metrics.Options
	Logs                    *logs.Options
	Traces                  *genericoptions.TracingOptions

	EnableLogsHandler        bool
	EventTTL                 time.Duration
	MaxConnectionBytesPerSec int64

	ProxyClientCertFile string
	ProxyClientKeyFile  string

	EnableAggregatorRouting bool

	IdentityLeaseDurationSeconds      int
	IdentityLeaseRenewIntervalSeconds int

	ServiceAccountSigningKeyFile     string
	ServiceAccountIssuer             serviceaccount.TokenGenerator
	ServiceAccountTokenMaxExpiration time.Duration

	ShowHiddenMetricsForVersion string
}

// completedServerRunOptions is a private wrapper that enforces a call of Complete() before Run can be invoked.
type completedServerRunOptions struct {
	ServerRunOptions
}

type CompletedServerRunOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedServerRunOptions
}

// NewServerRunOptions creates a new ServerRunOptions object with default parameters
func NewServerRunOptions() *ServerRunOptions {
	s := ServerRunOptions{
		GenericServerRunOptions: genericoptions.NewServerRunOptions(),
		Etcd:                    genericoptions.NewEtcdOptions(storagebackend.NewDefaultConfig(kubeoptions.DefaultEtcdPathPrefix, nil)),
		SecureServing:           kubeoptions.NewSecureServingOptions().WithLoopback(),
		Audit:                   genericoptions.NewAuditOptions(),
		Features:                genericoptions.NewFeatureOptions(),
		Admission:               genericoptions.NewAdmissionOptions(),
		Authentication:          kubeoptions.NewBuiltInAuthenticationOptions().WithAll(),
		APIEnablement:           genericoptions.NewAPIEnablementOptions(),
		EgressSelector:          genericoptions.NewEgressSelectorOptions(),
		Metrics:                 metrics.NewOptions(),
		Logs:                    logs.NewOptions(),
		Traces:                  genericoptions.NewTracingOptions(),

		EnableLogsHandler: true,
		EventTTL:          1 * time.Hour,

		IdentityLeaseDurationSeconds:      3600,
		IdentityLeaseRenewIntervalSeconds: 10,
	}

	// disable the watch cache
	s.Etcd.EnableWatchCache = false

	// TODO: turn off the admission webhooks for now
	s.Admission.DefaultOffPlugins.Insert(validating.PluginName, mutating.PluginName)

	// Overwrite the default for storage data format.
	s.Etcd.DefaultStorageMediaType = "application/vnd.kubernetes.protobuf"

	return &s
}

// Complete defaults missing field values. It mutates the receiver.
func (o *ServerRunOptions) Complete() (CompletedServerRunOptions, error) {
	if err := o.GenericServerRunOptions.DefaultAdvertiseAddress(o.SecureServing.SecureServingOptions); err != nil {
		return CompletedServerRunOptions{}, err
	}

	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts(o.GenericServerRunOptions.AdvertiseAddress.String(), nil, nil); err != nil {
		return CompletedServerRunOptions{}, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	if len(o.GenericServerRunOptions.ExternalHost) == 0 {
		if len(o.GenericServerRunOptions.AdvertiseAddress) > 0 {
			o.GenericServerRunOptions.ExternalHost = o.GenericServerRunOptions.AdvertiseAddress.String()
		} else {
			if hostname, err := os.Hostname(); err == nil {
				o.GenericServerRunOptions.ExternalHost = hostname
			} else {
				return CompletedServerRunOptions{}, fmt.Errorf("error finding host name: %v", err)
			}
		}
		klog.Infof("external host was not specified, using %v", o.GenericServerRunOptions.ExternalHost)
	}

	for key, value := range o.APIEnablement.RuntimeConfig {
		if key == "v1" || strings.HasPrefix(key, "v1/") ||
			key == "api/v1" || strings.HasPrefix(key, "api/v1/") {
			delete(o.APIEnablement.RuntimeConfig, key)
			o.APIEnablement.RuntimeConfig["/v1"] = value
		}
		if key == "api/legacy" {
			delete(o.APIEnablement.RuntimeConfig, key)
		}
	}

	return CompletedServerRunOptions{
		&completedServerRunOptions{
			ServerRunOptions: *o,
		},
	}, nil
}
