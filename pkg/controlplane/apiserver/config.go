/*
Copyright 2023 The Kubernetes Authors.

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

package apiserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/kcp-dev/client-go/dynamic"
	kcpinformers "github.com/kcp-dev/client-go/informers"
	clientset "github.com/kcp-dev/client-go/kubernetes"
	kcpclient "github.com/kcp-dev/client-go/kubernetes"
	"github.com/kcp-dev/logicalcluster/v3"
	oteltrace "go.opentelemetry.io/otel/trace"

	"k8s.io/apimachinery/pkg/runtime"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/clientsethack"
	"k8s.io/apiserver/pkg/dynamichack"
	"k8s.io/apiserver/pkg/endpoints/discovery/aggregated"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericfeatures "k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/informerfactoryhack"
	"k8s.io/apiserver/pkg/reconcilers"
	peerreconcilers "k8s.io/apiserver/pkg/reconcilers"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/egressselector"
	"k8s.io/apiserver/pkg/server/filters"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/apiserver/pkg/util/openapi"
	utilpeerproxy "k8s.io/apiserver/pkg/util/peerproxy"
	clientgoinformers "k8s.io/client-go/informers"
	clientgoclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/transport"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/component-base/version"
	aggregatorapiserver "k8s.io/kube-aggregator/pkg/apiserver"
	openapicommon "k8s.io/kube-openapi/pkg/common"

	"k8s.io/kubernetes/pkg/api/legacyscheme"
	controlplaneadmission "k8s.io/kubernetes/pkg/controlplane/apiserver/admission"
	"k8s.io/kubernetes/pkg/controlplane/apiserver/options"
	"k8s.io/kubernetes/pkg/controlplane/controller/clusterauthenticationtrust"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/kubeapiserver"
	"k8s.io/kubernetes/pkg/kubeapiserver/authorizer/modes"
	rbacrest "k8s.io/kubernetes/pkg/registry/rbac/rest"
	"k8s.io/kubernetes/pkg/serviceaccount"
)

// LocalAdminCluster is the default logical cluster that kube-apiserver's
// objects, e.g. the RBAC bootstrap policy land in.
var LocalAdminCluster = logicalcluster.Name("system:admin")

// Config defines configuration for the master
type Config struct {
	Generic *genericapiserver.Config
	Extra
}

type Extra struct {
	ClusterAuthenticationInfo clusterauthenticationtrust.ClusterAuthenticationInfo

	APIResourceConfigSource serverstorage.APIResourceConfigSource
	StorageFactory          serverstorage.StorageFactory
	EventTTL                time.Duration

	EnableLogsSupport bool
	ProxyTransport    *http.Transport

	// PeerProxy, if not nil, sets proxy transport between kube-apiserver peers for requests
	// that can not be served locally
	PeerProxy utilpeerproxy.Interface
	// PeerEndpointReconcileInterval defines how often the endpoint leases are reconciled in etcd.
	PeerEndpointReconcileInterval time.Duration
	// PeerEndpointLeaseReconciler updates the peer endpoint leases
	PeerEndpointLeaseReconciler peerreconcilers.PeerEndpointLeaseReconciler
	// PeerAdvertiseAddress is the IP for this kube-apiserver which is used by peer apiservers to route a request
	// to this apiserver. This happens in cases where the peer is not able to serve the request due to
	// version skew. If unset, AdvertiseAddress/BindAddress will be used.
	PeerAdvertiseAddress peerreconcilers.PeerAdvertiseAddress

	ServiceAccountIssuer        serviceaccount.TokenGenerator
	ServiceAccountMaxExpiration time.Duration
	ExtendExpiration            bool

	// ServiceAccountIssuerDiscovery
	ServiceAccountIssuerURL  string
	ServiceAccountJWKSURI    string
	ServiceAccountPublicKeys []interface{}

	SystemNamespaces []string

	VersionedInformers kcpinformers.SharedInformerFactory
}

// BuildGenericConfig takes the generic controlplane apiserver options and produces
// the genericapiserver.Config associated with it. The genericapiserver.Config is
// often shared between multiple delegated apiservers.
func BuildGenericConfig(
	s options.CompletedOptions,
	schemes []*runtime.Scheme,
	resourceConfig *serverstorage.ResourceConfig,
	getOpenAPIDefinitions func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition,
) (
	genericConfig *genericapiserver.Config,
	versionedInformers kcpinformers.SharedInformerFactory,
	storageFactory *serverstorage.DefaultStorageFactory,
	lastErr error,
) {
	genericConfig = genericapiserver.NewConfig(legacyscheme.Codecs)
	genericConfig.MergedResourceConfig = resourceConfig

	if lastErr = s.GenericServerRunOptions.ApplyTo(genericConfig); lastErr != nil {
		return
	}

	if lastErr = s.SecureServing.ApplyTo(&genericConfig.SecureServing, &genericConfig.LoopbackClientConfig); lastErr != nil {
		return
	}

	// Disable compression for self-communication, since we are going to be
	// on a fast local network
	genericConfig.LoopbackClientConfig.DisableCompression = true

	kubeClientConfig := genericConfig.LoopbackClientConfig
	clientgoExternalClient, err := clientgoclientset.NewForConfig(kubeClientConfig)
	if err != nil {
		lastErr = fmt.Errorf("failed to create real external clientset: %w", err)
		return
	}
	versionedInformers = clientgoinformers.NewSharedInformerFactory(clientgoExternalClient, 10*time.Minute)

	if lastErr = s.Features.ApplyTo(genericConfig, clientgoExternalClient, versionedInformers); lastErr != nil {
		return
	}
	if lastErr = s.APIEnablement.ApplyTo(genericConfig, resourceConfig, legacyscheme.Scheme); lastErr != nil {
		return
	}
	if lastErr = s.EgressSelector.ApplyTo(genericConfig); lastErr != nil {
		return
	}
	if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.APIServerTracing) {
		if lastErr = s.Traces.ApplyTo(genericConfig.EgressSelector, genericConfig); lastErr != nil {
			return
		}
	}
	// wrap the definitions to revert any changes from disabled features
	getOpenAPIDefinitions = openapi.GetOpenAPIDefinitionsWithoutDisabledFeatures(getOpenAPIDefinitions)
	namer := openapinamer.NewDefinitionNamer(schemes...)
	genericConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(getOpenAPIDefinitions, namer)
	genericConfig.OpenAPIConfig.Info.Title = "Kubernetes"
	genericConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(getOpenAPIDefinitions, namer)
	genericConfig.OpenAPIV3Config.Info.Title = "Kubernetes"

	genericConfig.LongRunningFunc = filters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString("attach", "exec", "proxy", "log", "portforward"),
	)

	kubeVersion := version.Get()
	genericConfig.Version = &kubeVersion

	if genericConfig.EgressSelector != nil {
		s.Etcd.StorageConfig.Transport.EgressLookup = genericConfig.EgressSelector.Lookup
	}
	if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.APIServerTracing) {
		s.Etcd.StorageConfig.Transport.TracerProvider = genericConfig.TracerProvider
	} else {
		s.Etcd.StorageConfig.Transport.TracerProvider = oteltrace.NewNoopTracerProvider()
	}

	storageFactoryConfig := kubeapiserver.NewStorageFactoryConfig()
	storageFactoryConfig.APIResourceConfig = genericConfig.MergedResourceConfig
	storageFactory, lastErr = storageFactoryConfig.Complete(s.Etcd).New()
	if lastErr != nil {
		return
	}
	if lastErr = s.Etcd.ApplyWithStorageFactoryTo(storageFactory, genericConfig); lastErr != nil {
		return
	}

	ctx := wait.ContextForChannel(genericConfig.DrainedNotify())
	// Disable compression for self-communication, since we are going to be
	// on a fast local network
	genericConfig.LoopbackClientConfig.DisableCompression = true

	kubeClientConfig := genericConfig.LoopbackClientConfig
	clusterClient, err := kcpclient.NewForConfig(kubeClientConfig)
	if err != nil {
		lastErr = fmt.Errorf("failed to create cluster clientset: %v", err)
		return
	}
	versionedInformers = kcpinformers.NewSharedInformerFactory(clusterClient, 10*time.Minute)

	// Authentication.ApplyTo requires already applied OpenAPIConfig and EgressSelector if present
	if lastErr = s.Authentication.ApplyTo(ctx, &genericConfig.Authentication, genericConfig.SecureServing, genericConfig.EgressSelector, genericConfig.OpenAPIConfig, genericConfig.OpenAPIV3Config, clusterClient, versionedInformers, genericConfig.APIServerID); lastErr != nil {
		return
	}

	var enablesRBAC bool
	genericConfig.Authorization.Authorizer, genericConfig.RuleResolver, enablesRBAC, err = BuildAuthorizer(
		ctx,
		s,
		genericConfig.EgressSelector,
		genericConfig.APIServerID,
		informerfactoryhack.Wrap(versionedInformers),
	)
	if err != nil {
		lastErr = fmt.Errorf("invalid authorization config: %w", err)
		return
	}
	if s.Authorization != nil && !enablesRBAC {
		genericConfig.DisabledPostStartHooks.Insert(rbacrest.PostStartHookName)
	}

	lastErr = s.Audit.ApplyTo(genericConfig)
	if lastErr != nil {
		return
	}

	if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.APIPriorityAndFairness) && s.GenericServerRunOptions.EnablePriorityAndFairness {
		genericConfig.FlowControl, lastErr = BuildPriorityAndFairness(s, clusterClient.Cluster(LocalAdminCluster.Path()), informerfactoryhack.Wrap(versionedInformers))
	}

	if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.AggregatedDiscoveryEndpoint) {
		genericConfig.AggregatedDiscoveryGroupManager = aggregated.NewResourceManager("apis")
	}

	return
}

// BuildAuthorizer constructs the authorizer. If authorization is not set in s, it returns nil, nil, false, nil
func BuildAuthorizer(ctx context.Context, s options.CompletedOptions, egressSelector *egressselector.EgressSelector, apiserverID string, versionedInformers clientgoinformers.SharedInformerFactory) (authorizer.Authorizer, authorizer.RuleResolver, bool, error) {
	authorizationConfig, err := s.Authorization.ToAuthorizationConfig(versionedInformers)
	if err != nil {
		return nil, nil, false, err
	}
	if authorizationConfig == nil {
		return nil, nil, false, nil
	}

	if egressSelector != nil {
		egressDialer, err := egressSelector.Lookup(egressselector.ControlPlane.AsNetworkContext())
		if err != nil {
			return nil, nil, false, err
		}
		authorizationConfig.CustomDial = egressDialer
	}

	enablesRBAC := false
	for _, a := range authorizationConfig.AuthorizationConfiguration.Authorizers {
		if string(a.Type) == modes.ModeRBAC {
			enablesRBAC = true
			break
		}
	}

	authorizer, ruleResolver, err := authorizationConfig.New(ctx, apiserverID)

	return authorizer, ruleResolver, enablesRBAC, err
}

// CreateConfig takes the generic controlplane apiserver options and
// creates a config for the generic Kube APIs out of it.
func CreateConfig(
	opts options.CompletedOptions,
	genericConfig *genericapiserver.Config,
	versionedInformers kcpinformers.SharedInformerFactory,
	storageFactory *serverstorage.DefaultStorageFactory,
	serviceResolver aggregatorapiserver.ServiceResolver,
	additionalInitializers []admission.PluginInitializer,
) (
	*Config,
	[]admission.PluginInitializer,
	error,
) {
	proxyTransport := CreateProxyTransport()

	opts.Metrics.Apply()
	serviceaccount.RegisterMetrics()

	config := &Config{
		Generic: genericConfig,
		Extra: Extra{
			APIResourceConfigSource: storageFactory.APIResourceConfigSource,
			StorageFactory:          storageFactory,
			EventTTL:                opts.EventTTL,
			EnableLogsSupport:       opts.EnableLogsHandler,
			ProxyTransport:          proxyTransport,
			SystemNamespaces:        opts.SystemNamespaces,

			ServiceAccountIssuer:        opts.ServiceAccountIssuer,
			ServiceAccountMaxExpiration: opts.ServiceAccountTokenMaxExpiration,
			ExtendExpiration:            opts.Authentication.ServiceAccounts.ExtendExpiration,

			VersionedInformers: versionedInformers,
		},
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.UnknownVersionInteroperabilityProxy) {
		var err error
		config.PeerEndpointLeaseReconciler, err = CreatePeerEndpointLeaseReconciler(*genericConfig, storageFactory)
		if err != nil {
			return nil, nil, err
		}
		// build peer proxy config only if peer ca file exists
		if opts.PeerCAFile != "" {
			config.PeerProxy, err = BuildPeerProxy(informerfactoryhack.Wrap(versionedInformers), genericConfig.StorageVersionManager, opts.ProxyClientCertFile,
				opts.ProxyClientKeyFile, opts.PeerCAFile, opts.PeerAdvertiseAddress, genericConfig.APIServerID, config.Extra.PeerEndpointLeaseReconciler, config.Generic.Serializer)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	clientCAProvider, err := opts.Authentication.ClientCert.GetClientCAContentProvider()
	if err != nil {
		return nil, nil, err
	}
	config.ClusterAuthenticationInfo.ClientCA = clientCAProvider

	requestHeaderConfig, err := opts.Authentication.RequestHeader.ToAuthenticationRequestHeaderConfig()
	if err != nil {
		return nil, nil, err
	}
	if requestHeaderConfig != nil {
		config.ClusterAuthenticationInfo.RequestHeaderCA = requestHeaderConfig.CAContentProvider
		config.ClusterAuthenticationInfo.RequestHeaderAllowedNames = requestHeaderConfig.AllowedClientNames
		config.ClusterAuthenticationInfo.RequestHeaderExtraHeaderPrefixes = requestHeaderConfig.ExtraHeaderPrefixes
		config.ClusterAuthenticationInfo.RequestHeaderGroupHeaders = requestHeaderConfig.GroupHeaders
		config.ClusterAuthenticationInfo.RequestHeaderUsernameHeaders = requestHeaderConfig.UsernameHeaders
	}

	// setup admission
	genericAdmissionConfig := controlplaneadmission.Config{
		ExternalInformers:    informerfactoryhack.Wrap(versionedInformers),
		LoopbackClientConfig: genericConfig.LoopbackClientConfig,
	}
	genericInitializers, err := genericAdmissionConfig.New(proxyTransport, genericConfig.EgressSelector, serviceResolver, genericConfig.TracerProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create admission plugin initializer: %w", err)
	}
	clientgoExternalClient, err := clientgoclientset.NewForConfig(genericConfig.LoopbackClientConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create real client-go external client: %w", err)
	}
	dynamicExternalClient, err := dynamic.NewForConfig(genericConfig.LoopbackClientConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create real dynamic external client: %w", err)
	}
	err = opts.Admission.ApplyTo(
		genericConfig,
		informerfactoryhack.Wrap(versionedInformers),
		clientsethack.Wrap(clientgoExternalClient),
		dynamichack.Wrap(dynamicExternalClient),
		utilfeature.DefaultFeatureGate,
		append(genericInitializers, additionalInitializers...)...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to apply admission: %w", err)
	}

	// Load and set the public keys.
	var pubKeys []interface{}
	for _, f := range opts.Authentication.ServiceAccounts.KeyFiles {
		keys, err := keyutil.PublicKeysFromFile(f)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse key file %q: %w", f, err)
		}
		pubKeys = append(pubKeys, keys...)
	}
	config.ServiceAccountIssuerURL = opts.Authentication.ServiceAccounts.Issuers[0]
	config.ServiceAccountJWKSURI = opts.Authentication.ServiceAccounts.JWKSURI
	config.ServiceAccountPublicKeys = pubKeys

	return config, genericInitializers, nil
}

// CreateProxyTransport creates the dialer infrastructure to connect to the nodes.
func CreateProxyTransport() *http.Transport {
	var proxyDialerFn utilnet.DialFunc
	// Proxying to pods and services is IP-based... don't expect to be able to verify the hostname
	proxyTLSClientConfig := &tls.Config{InsecureSkipVerify: true}
	proxyTransport := utilnet.SetTransportDefaults(&http.Transport{
		DialContext:     proxyDialerFn,
		TLSClientConfig: proxyTLSClientConfig,
	})
	return proxyTransport
}
