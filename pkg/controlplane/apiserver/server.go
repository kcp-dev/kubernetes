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
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	coordinationapiv1 "k8s.io/api/coordination/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	apiserverfeatures "k8s.io/apiserver/pkg/features"
	peerreconcilers "k8s.io/apiserver/pkg/reconcilers"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	clientgoinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/component-helpers/apimachinery/lease"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"

	"k8s.io/kubernetes/pkg/controlplane/controller/apiserverleasegc"
	"k8s.io/kubernetes/pkg/controlplane/controller/clusterauthenticationtrust"
	"k8s.io/kubernetes/pkg/controlplane/controller/legacytokentracking"
	"k8s.io/kubernetes/pkg/controlplane/controller/systemnamespaces"
	"k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/routes"
	"k8s.io/kubernetes/pkg/serviceaccount"
)

var (
	// IdentityLeaseGCPeriod is the interval which the lease GC controller checks for expired leases
	// IdentityLeaseGCPeriod is exposed so integration tests can tune this value.
	IdentityLeaseGCPeriod = 3600 * time.Second
	// IdentityLeaseDurationSeconds is the duration of kube-apiserver lease in seconds
	// IdentityLeaseDurationSeconds is exposed so integration tests can tune this value.
	IdentityLeaseDurationSeconds = 3600
	// IdentityLeaseRenewIntervalPeriod is the interval of kube-apiserver renewing its lease in seconds
	// IdentityLeaseRenewIntervalPeriod is exposed so integration tests can tune this value.
	IdentityLeaseRenewIntervalPeriod = 10 * time.Second
	// IdentityLeaseComponentLabelKey is used to apply a component label to identity lease objects, indicating:
	//   1. the lease is an identity lease (different from leader election leases)
	//   2. which component owns this lease
	IdentityLeaseComponentLabelKey = "apiserver.kubernetes.io/identity"

	// DeprecatedKubeAPIServerIdentityLeaseLabelSelector selects kube-apiserver identity leases
	DeprecatedKubeAPIServerIdentityLeaseLabelSelector = "k8s.io/component=kube-apiserver"
)

// Server is a struct that contains a generic control plane apiserver instance
// that can be run to start serving the APIs.
type Server struct {
	GenericAPIServer *genericapiserver.GenericAPIServer

	APIResourceConfigSource   serverstorage.APIResourceConfigSource
	RESTOptionsGetter         genericregistry.RESTOptionsGetter
	ClusterAuthenticationInfo clusterauthenticationtrust.ClusterAuthenticationInfo
	VersionedInformers        clientgoinformers.SharedInformerFactory
}

// New returns a new instance of Master from the given config.
// Certain config fields will be set to a default value if unset.
// Certain config fields must be specified, including:
// KubeletClientConfig
func (c completedConfig) New(name string, delegationTarget genericapiserver.DelegationTarget) (*Server, error) {
	generic, err := c.Generic.New(name, delegationTarget)
	if err != nil {
		return nil, err
	}

	if c.EnableLogsSupport {
		routes.Logs{}.Install(generic.Handler.GoRestfulContainer)
	}

	// Metadata and keys are expected to only change across restarts at present,
	// so we just marshal immediately and serve the cached JSON bytes.
	md, err := serviceaccount.NewOpenIDMetadata(
		c.ServiceAccountIssuerURL,
		c.ServiceAccountJWKSURI,
		c.Generic.ExternalAddress,
		c.ServiceAccountPublicKeys,
	)
	if err != nil {
		// If there was an error, skip installing the endpoints and log the
		// error, but continue on. We don't return the error because the
		// metadata responses require additional, backwards incompatible
		// validation of command-line options.
		msg := fmt.Sprintf("Could not construct pre-rendered responses for"+
			" ServiceAccountIssuerDiscovery endpoints. Endpoints will not be"+
			" enabled. Error: %v", err)
		if c.ServiceAccountIssuerURL != "" {
			// The user likely expects this feature to be enabled if issuer URL is
			// set and the feature gate is enabled. In the future, if there is no
			// longer a feature gate and issuer URL is not set, the user may not
			// expect this feature to be enabled. We log the former case as an Error
			// and the latter case as an Info.
			klog.Error(msg)
		} else {
			klog.Info(msg)
		}
	} else {
		routes.NewOpenIDMetadataServer(md.ConfigJSON, md.PublicKeysetJSON).
			Install(generic.Handler.GoRestfulContainer)
	}

	s := &Server{
		GenericAPIServer: generic,

		APIResourceConfigSource:   c.APIResourceConfigSource,
		RESTOptionsGetter:         c.Generic.RESTOptionsGetter,
		ClusterAuthenticationInfo: c.ClusterAuthenticationInfo,
		VersionedInformers:        c.VersionedInformers,
	}

	client := kubernetes.NewForConfigOrDie(s.GenericAPIServer.LoopbackClientConfig)
	if len(c.SystemNamespaces) > 0 {
		s.GenericAPIServer.AddPostStartHookOrDie("start-system-namespaces-controller", func(hookContext genericapiserver.PostStartHookContext) error {
			go systemnamespaces.NewController(c.SystemNamespaces, client, s.VersionedInformers.Core().V1().Namespaces()).Run(hookContext.StopCh)
			return nil
		})
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.UnknownVersionInteroperabilityProxy) {
		_, publicServicePort, err := c.Generic.SecureServing.HostPort()
		if err != nil {
			return nil, fmt.Errorf("failed to get listener address: %w", err)
		}
		peeraddress := getPeerAddress(c.Extra.PeerAdvertiseAddress, c.Generic.PublicAddress, publicServicePort)
		peerEndpointCtrl := peerreconcilers.New(
			c.Generic.APIServerID,
			peeraddress,
			c.Extra.PeerEndpointLeaseReconciler,
			c.Extra.PeerEndpointReconcileInterval,
			client)
		if err != nil {
			return nil, fmt.Errorf("failed to create peer endpoint lease controller: %w", err)
		}
		s.GenericAPIServer.AddPostStartHookOrDie("peer-endpoint-reconciler-controller",
			func(hookContext genericapiserver.PostStartHookContext) error {
				peerEndpointCtrl.Start(hookContext.StopCh)
				return nil
			})
		s.GenericAPIServer.AddPreShutdownHookOrDie("peer-endpoint-reconciler-controller",
			func() error {
				peerEndpointCtrl.Stop()
				return nil
			})
		if c.Extra.PeerProxy != nil {
			s.GenericAPIServer.AddPostStartHookOrDie("unknown-version-proxy-filter", func(context genericapiserver.PostStartHookContext) error {
				err := c.Extra.PeerProxy.WaitForCacheSync(context.StopCh)
				return err
			})
		}
	}

	s.GenericAPIServer.AddPostStartHookOrDie("start-cluster-authentication-info-controller", func(hookContext genericapiserver.PostStartHookContext) error {
		kubeClient, err := kubernetes.NewForConfig(hookContext.LoopbackClientConfig)
		if err != nil {
			return err
		}
		controller := clusterauthenticationtrust.NewClusterAuthenticationTrustController(s.ClusterAuthenticationInfo, kubeClient)

		// generate a context  from stopCh. This is to avoid modifying files which are relying on apiserver
		// TODO: See if we can pass ctx to the current method
		ctx := wait.ContextForChannel(hookContext.StopCh)

		// prime values and start listeners
		if s.ClusterAuthenticationInfo.ClientCA != nil {
			s.ClusterAuthenticationInfo.ClientCA.AddListener(controller)
			if controller, ok := s.ClusterAuthenticationInfo.ClientCA.(dynamiccertificates.ControllerRunner); ok {
				// runonce to be sure that we have a value.
				if err := controller.RunOnce(ctx); err != nil {
					runtime.HandleError(err)
				}
				go controller.Run(ctx, 1)
			}
		}
		if s.ClusterAuthenticationInfo.RequestHeaderCA != nil {
			s.ClusterAuthenticationInfo.RequestHeaderCA.AddListener(controller)
			if controller, ok := s.ClusterAuthenticationInfo.RequestHeaderCA.(dynamiccertificates.ControllerRunner); ok {
				// runonce to be sure that we have a value.
				if err := controller.RunOnce(ctx); err != nil {
					runtime.HandleError(err)
				}
				go controller.Run(ctx, 1)
			}
		}

		go controller.Run(ctx, 1)
		return nil
	})

	if utilfeature.DefaultFeatureGate.Enabled(apiserverfeatures.APIServerIdentity) {
		s.GenericAPIServer.AddPostStartHookOrDie("start-kube-apiserver-identity-lease-controller", func(hookContext genericapiserver.PostStartHookContext) error {
			kubeClient, err := kubernetes.NewForConfig(hookContext.LoopbackClientConfig)
			if err != nil {
				return err
			}

			// generate a context  from stopCh. This is to avoid modifying files which are relying on apiserver
			// TODO: See if we can pass ctx to the current method
			ctx := wait.ContextForChannel(hookContext.StopCh)

			leaseName := s.GenericAPIServer.APIServerID
			holderIdentity := s.GenericAPIServer.APIServerID + "_" + string(uuid.NewUUID())

			controller := lease.NewController(
				clock.RealClock{},
				kubeClient,
				holderIdentity,
				int32(IdentityLeaseDurationSeconds),
				nil,
				IdentityLeaseRenewIntervalPeriod,
				leaseName,
				metav1.NamespaceSystem,
				// TODO: receive identity label value as a parameter when post start hook is moved to generic apiserver.
				labelAPIServerHeartbeatFunc(name))
			go controller.Run(ctx)
			return nil
		})
		// Labels for apiserver idenitiy leases switched from k8s.io/component=kube-apiserver to apiserver.kubernetes.io/identity=kube-apiserver.
		// For compatibility, garbage collect leases with both labels for at least 1 release
		// TODO: remove in Kubernetes 1.28
		s.GenericAPIServer.AddPostStartHookOrDie("start-deprecated-kube-apiserver-identity-lease-garbage-collector", func(hookContext genericapiserver.PostStartHookContext) error {
			kubeClient, err := kubernetes.NewForConfig(hookContext.LoopbackClientConfig)
			if err != nil {
				return err
			}
			go apiserverleasegc.NewAPIServerLeaseGC(
				kubeClient,
				IdentityLeaseGCPeriod,
				metav1.NamespaceSystem,
				DeprecatedKubeAPIServerIdentityLeaseLabelSelector,
			).Run(hookContext.StopCh)
			return nil
		})
		// TODO: move this into generic apiserver and make the lease identity value configurable
		s.GenericAPIServer.AddPostStartHookOrDie("start-kube-apiserver-identity-lease-garbage-collector", func(hookContext genericapiserver.PostStartHookContext) error {
			kubeClient, err := kubernetes.NewForConfig(hookContext.LoopbackClientConfig)
			if err != nil {
				return err
			}
			go apiserverleasegc.NewAPIServerLeaseGC(
				kubeClient,
				IdentityLeaseGCPeriod,
				metav1.NamespaceSystem,
				IdentityLeaseComponentLabelKey+"="+name,
			).Run(hookContext.StopCh)
			return nil
		})
	}

	s.GenericAPIServer.AddPostStartHookOrDie("start-legacy-token-tracking-controller", func(hookContext genericapiserver.PostStartHookContext) error {
		kubeClient, err := kubernetes.NewForConfig(hookContext.LoopbackClientConfig)
		if err != nil {
			return err
		}
		go legacytokentracking.NewController(kubeClient).Run(hookContext.StopCh)
		return nil
	})

	return s, nil
}

func labelAPIServerHeartbeatFunc(identity string) lease.ProcessLeaseFunc {
	return func(lease *coordinationapiv1.Lease) error {
		if lease.Labels == nil {
			lease.Labels = map[string]string{}
		}

		// This label indiciates the identity of the lease object.
		lease.Labels[IdentityLeaseComponentLabelKey] = identity

		hostname, err := os.Hostname()
		if err != nil {
			return err
		}

		// convenience label to easily map a lease object to a specific apiserver
		lease.Labels[apiv1.LabelHostname] = hostname
		return nil
	}
}

// utility function to get the apiserver address that is used by peer apiservers to proxy
// requests to this apiserver in case the peer is incapable of serving the request
func getPeerAddress(peerAdvertiseAddress peerreconcilers.PeerAdvertiseAddress, publicAddress net.IP, publicServicePort int) string {
	if peerAdvertiseAddress.PeerAdvertiseIP != "" && peerAdvertiseAddress.PeerAdvertisePort != "" {
		return net.JoinHostPort(peerAdvertiseAddress.PeerAdvertiseIP, peerAdvertiseAddress.PeerAdvertisePort)
	} else {
		return net.JoinHostPort(publicAddress.String(), strconv.Itoa(publicServicePort))
	}
}
