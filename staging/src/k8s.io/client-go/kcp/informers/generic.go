//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by kcp code-generator. DO NOT EDIT.

package informers

import (
	"fmt"

	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"

	kcpcache "github.com/kcp-dev/apimachinery/pkg/cache"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apiserverinternalv1alpha1 "k8s.io/api/apiserverinternal/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	certificatesv1 "k8s.io/api/certificates/v1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	coordinationv1 "k8s.io/api/coordination/v1"
	coordinationv1beta1 "k8s.io/api/coordination/v1beta1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	eventsv1 "k8s.io/api/events/v1"
	eventsv1beta1 "k8s.io/api/events/v1beta1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	flowcontrolv1alpha1 "k8s.io/api/flowcontrol/v1alpha1"
	flowcontrolv1beta1 "k8s.io/api/flowcontrol/v1beta1"
	flowcontrolv1beta2 "k8s.io/api/flowcontrol/v1beta2"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	nodev1 "k8s.io/api/node/v1"
	nodev1alpha1 "k8s.io/api/node/v1alpha1"
	nodev1beta1 "k8s.io/api/node/v1beta1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1alpha1 "k8s.io/api/rbac/v1alpha1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	schedulingv1alpha1 "k8s.io/api/scheduling/v1alpha1"
	schedulingv1beta1 "k8s.io/api/scheduling/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1alpha1 "k8s.io/api/storage/v1alpha1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	upstreaminformers "k8s.io/client-go/informers"
)

type GenericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *GenericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericClusterLister.
func (f *GenericInformer) Lister() cache.GenericLister {
	return kcpcache.NewGenericClusterLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *SharedInformerFactory) ForResource(resource schema.GroupVersionResource) (upstreaminformers.GenericInformer, error) {
	switch resource {
	// Group=admissionregistration.k8s.io, Version=v1
	case admissionregistrationv1.SchemeGroupVersion.WithResource("mutatingwebhookconfigurations"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Admissionregistration().V1().MutatingWebhookConfigurations().Informer()}, nil
	case admissionregistrationv1.SchemeGroupVersion.WithResource("validatingwebhookconfigurations"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Admissionregistration().V1().ValidatingWebhookConfigurations().Informer()}, nil

		// Group=admissionregistration.k8s.io, Version=v1beta1
	case admissionregistrationv1beta1.SchemeGroupVersion.WithResource("mutatingwebhookconfigurations"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Admissionregistration().V1beta1().MutatingWebhookConfigurations().Informer()}, nil
	case admissionregistrationv1beta1.SchemeGroupVersion.WithResource("validatingwebhookconfigurations"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Admissionregistration().V1beta1().ValidatingWebhookConfigurations().Informer()}, nil

		// Group=internal.apiserver.k8s.io, Version=v1alpha1
	case apiserverinternalv1alpha1.SchemeGroupVersion.WithResource("storageversions"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Internal().V1alpha1().StorageVersions().Informer()}, nil

		// Group=apps, Version=v1
	case appsv1.SchemeGroupVersion.WithResource("controllerrevisions"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1().ControllerRevisions().Informer()}, nil
	case appsv1.SchemeGroupVersion.WithResource("daemonsets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1().DaemonSets().Informer()}, nil
	case appsv1.SchemeGroupVersion.WithResource("deployments"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1().Deployments().Informer()}, nil
	case appsv1.SchemeGroupVersion.WithResource("replicasets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1().ReplicaSets().Informer()}, nil
	case appsv1.SchemeGroupVersion.WithResource("statefulsets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1().StatefulSets().Informer()}, nil

		// Group=apps, Version=v1beta1
	case appsv1beta1.SchemeGroupVersion.WithResource("controllerrevisions"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1beta1().ControllerRevisions().Informer()}, nil
	case appsv1beta1.SchemeGroupVersion.WithResource("deployments"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1beta1().Deployments().Informer()}, nil
	case appsv1beta1.SchemeGroupVersion.WithResource("statefulsets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1beta1().StatefulSets().Informer()}, nil

		// Group=apps, Version=v1beta2
	case appsv1beta2.SchemeGroupVersion.WithResource("controllerrevisions"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1beta2().ControllerRevisions().Informer()}, nil
	case appsv1beta2.SchemeGroupVersion.WithResource("daemonsets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1beta2().DaemonSets().Informer()}, nil
	case appsv1beta2.SchemeGroupVersion.WithResource("deployments"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1beta2().Deployments().Informer()}, nil
	case appsv1beta2.SchemeGroupVersion.WithResource("replicasets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1beta2().ReplicaSets().Informer()}, nil
	case appsv1beta2.SchemeGroupVersion.WithResource("statefulsets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Apps().V1beta2().StatefulSets().Informer()}, nil

		// Group=autoscaling, Version=v1
	case autoscalingv1.SchemeGroupVersion.WithResource("horizontalpodautoscalers"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Autoscaling().V1().HorizontalPodAutoscalers().Informer()}, nil

		// Group=autoscaling, Version=v2
	case autoscalingv2.SchemeGroupVersion.WithResource("horizontalpodautoscalers"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Autoscaling().V2().HorizontalPodAutoscalers().Informer()}, nil

		// Group=autoscaling, Version=v2beta1
	case autoscalingv2beta1.SchemeGroupVersion.WithResource("horizontalpodautoscalers"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Autoscaling().V2beta1().HorizontalPodAutoscalers().Informer()}, nil

		// Group=autoscaling, Version=v2beta2
	case autoscalingv2beta2.SchemeGroupVersion.WithResource("horizontalpodautoscalers"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Autoscaling().V2beta2().HorizontalPodAutoscalers().Informer()}, nil

		// Group=batch, Version=v1
	case batchv1.SchemeGroupVersion.WithResource("cronjobs"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Batch().V1().CronJobs().Informer()}, nil
	case batchv1.SchemeGroupVersion.WithResource("jobs"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Batch().V1().Jobs().Informer()}, nil

		// Group=batch, Version=v1beta1
	case batchv1beta1.SchemeGroupVersion.WithResource("cronjobs"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Batch().V1beta1().CronJobs().Informer()}, nil

		// Group=certificates.k8s.io, Version=v1
	case certificatesv1.SchemeGroupVersion.WithResource("certificatesigningrequests"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Certificates().V1().CertificateSigningRequests().Informer()}, nil

		// Group=certificates.k8s.io, Version=v1beta1
	case certificatesv1beta1.SchemeGroupVersion.WithResource("certificatesigningrequests"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Certificates().V1beta1().CertificateSigningRequests().Informer()}, nil

		// Group=coordination.k8s.io, Version=v1
	case coordinationv1.SchemeGroupVersion.WithResource("leases"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Coordination().V1().Leases().Informer()}, nil

		// Group=coordination.k8s.io, Version=v1beta1
	case coordinationv1beta1.SchemeGroupVersion.WithResource("leases"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Coordination().V1beta1().Leases().Informer()}, nil

		// Group=core, Version=v1
	case corev1.SchemeGroupVersion.WithResource("componentstatuses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().ComponentStatuses().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("configmaps"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().ConfigMaps().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("endpoints"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().Endpoints().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("events"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().Events().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("limitranges"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().LimitRanges().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("namespaces"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().Namespaces().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("nodes"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().Nodes().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("persistentvolumes"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().PersistentVolumes().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("persistentvolumeclaims"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().PersistentVolumeClaims().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("pods"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().Pods().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("podtemplates"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().PodTemplates().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("replicationcontrollers"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().ReplicationControllers().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("resourcequotas"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().ResourceQuotas().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("secrets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().Secrets().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("services"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().Services().Informer()}, nil
	case corev1.SchemeGroupVersion.WithResource("serviceaccounts"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Core().V1().ServiceAccounts().Informer()}, nil

		// Group=discovery.k8s.io, Version=v1
	case discoveryv1.SchemeGroupVersion.WithResource("endpointslices"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Discovery().V1().EndpointSlices().Informer()}, nil

		// Group=discovery.k8s.io, Version=v1beta1
	case discoveryv1beta1.SchemeGroupVersion.WithResource("endpointslices"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Discovery().V1beta1().EndpointSlices().Informer()}, nil

		// Group=events.k8s.io, Version=v1
	case eventsv1.SchemeGroupVersion.WithResource("events"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Events().V1().Events().Informer()}, nil

		// Group=events.k8s.io, Version=v1beta1
	case eventsv1beta1.SchemeGroupVersion.WithResource("events"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Events().V1beta1().Events().Informer()}, nil

		// Group=extensions, Version=v1beta1
	case extensionsv1beta1.SchemeGroupVersion.WithResource("daemonsets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Extensions().V1beta1().DaemonSets().Informer()}, nil
	case extensionsv1beta1.SchemeGroupVersion.WithResource("deployments"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Extensions().V1beta1().Deployments().Informer()}, nil
	case extensionsv1beta1.SchemeGroupVersion.WithResource("ingresses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Extensions().V1beta1().Ingresses().Informer()}, nil
	case extensionsv1beta1.SchemeGroupVersion.WithResource("networkpolicies"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Extensions().V1beta1().NetworkPolicies().Informer()}, nil
	case extensionsv1beta1.SchemeGroupVersion.WithResource("podsecuritypolicies"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Extensions().V1beta1().PodSecurityPolicies().Informer()}, nil
	case extensionsv1beta1.SchemeGroupVersion.WithResource("replicasets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Extensions().V1beta1().ReplicaSets().Informer()}, nil

		// Group=flowcontrol.apiserver.k8s.io, Version=v1alpha1
	case flowcontrolv1alpha1.SchemeGroupVersion.WithResource("flowschemas"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Flowcontrol().V1alpha1().FlowSchemas().Informer()}, nil
	case flowcontrolv1alpha1.SchemeGroupVersion.WithResource("prioritylevelconfigurations"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Flowcontrol().V1alpha1().PriorityLevelConfigurations().Informer()}, nil

		// Group=flowcontrol.apiserver.k8s.io, Version=v1beta1
	case flowcontrolv1beta1.SchemeGroupVersion.WithResource("flowschemas"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Flowcontrol().V1beta1().FlowSchemas().Informer()}, nil
	case flowcontrolv1beta1.SchemeGroupVersion.WithResource("prioritylevelconfigurations"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Flowcontrol().V1beta1().PriorityLevelConfigurations().Informer()}, nil

		// Group=flowcontrol.apiserver.k8s.io, Version=v1beta2
	case flowcontrolv1beta2.SchemeGroupVersion.WithResource("flowschemas"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Flowcontrol().V1beta2().FlowSchemas().Informer()}, nil
	case flowcontrolv1beta2.SchemeGroupVersion.WithResource("prioritylevelconfigurations"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Flowcontrol().V1beta2().PriorityLevelConfigurations().Informer()}, nil

		// Group=networking.k8s.io, Version=v1
	case networkingv1.SchemeGroupVersion.WithResource("ingresses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Networking().V1().Ingresses().Informer()}, nil
	case networkingv1.SchemeGroupVersion.WithResource("ingressclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Networking().V1().IngressClasses().Informer()}, nil
	case networkingv1.SchemeGroupVersion.WithResource("networkpolicies"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Networking().V1().NetworkPolicies().Informer()}, nil

		// Group=networking.k8s.io, Version=v1beta1
	case networkingv1beta1.SchemeGroupVersion.WithResource("ingresses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Networking().V1beta1().Ingresses().Informer()}, nil
	case networkingv1beta1.SchemeGroupVersion.WithResource("ingressclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Networking().V1beta1().IngressClasses().Informer()}, nil

		// Group=node.k8s.io, Version=v1
	case nodev1.SchemeGroupVersion.WithResource("runtimeclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Node().V1().RuntimeClasses().Informer()}, nil

		// Group=node.k8s.io, Version=v1alpha1
	case nodev1alpha1.SchemeGroupVersion.WithResource("runtimeclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Node().V1alpha1().RuntimeClasses().Informer()}, nil

		// Group=node.k8s.io, Version=v1beta1
	case nodev1beta1.SchemeGroupVersion.WithResource("runtimeclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Node().V1beta1().RuntimeClasses().Informer()}, nil

		// Group=policy, Version=v1
	case policyv1.SchemeGroupVersion.WithResource("poddisruptionbudgets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Policy().V1().PodDisruptionBudgets().Informer()}, nil

		// Group=policy, Version=v1beta1
	case policyv1beta1.SchemeGroupVersion.WithResource("poddisruptionbudgets"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Policy().V1beta1().PodDisruptionBudgets().Informer()}, nil
	case policyv1beta1.SchemeGroupVersion.WithResource("podsecuritypolicies"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Policy().V1beta1().PodSecurityPolicies().Informer()}, nil

		// Group=rbac.authorization.k8s.io, Version=v1
	case rbacv1.SchemeGroupVersion.WithResource("clusterroles"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1().ClusterRoles().Informer()}, nil
	case rbacv1.SchemeGroupVersion.WithResource("clusterrolebindings"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1().ClusterRoleBindings().Informer()}, nil
	case rbacv1.SchemeGroupVersion.WithResource("roles"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1().Roles().Informer()}, nil
	case rbacv1.SchemeGroupVersion.WithResource("rolebindings"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1().RoleBindings().Informer()}, nil

		// Group=rbac.authorization.k8s.io, Version=v1alpha1
	case rbacv1alpha1.SchemeGroupVersion.WithResource("clusterroles"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1alpha1().ClusterRoles().Informer()}, nil
	case rbacv1alpha1.SchemeGroupVersion.WithResource("clusterrolebindings"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1alpha1().ClusterRoleBindings().Informer()}, nil
	case rbacv1alpha1.SchemeGroupVersion.WithResource("roles"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1alpha1().Roles().Informer()}, nil
	case rbacv1alpha1.SchemeGroupVersion.WithResource("rolebindings"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1alpha1().RoleBindings().Informer()}, nil

		// Group=rbac.authorization.k8s.io, Version=v1beta1
	case rbacv1beta1.SchemeGroupVersion.WithResource("clusterroles"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1beta1().ClusterRoles().Informer()}, nil
	case rbacv1beta1.SchemeGroupVersion.WithResource("clusterrolebindings"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1beta1().ClusterRoleBindings().Informer()}, nil
	case rbacv1beta1.SchemeGroupVersion.WithResource("roles"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1beta1().Roles().Informer()}, nil
	case rbacv1beta1.SchemeGroupVersion.WithResource("rolebindings"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Rbac().V1beta1().RoleBindings().Informer()}, nil

		// Group=scheduling.k8s.io, Version=v1
	case schedulingv1.SchemeGroupVersion.WithResource("priorityclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Scheduling().V1().PriorityClasses().Informer()}, nil

		// Group=scheduling.k8s.io, Version=v1alpha1
	case schedulingv1alpha1.SchemeGroupVersion.WithResource("priorityclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Scheduling().V1alpha1().PriorityClasses().Informer()}, nil

		// Group=scheduling.k8s.io, Version=v1beta1
	case schedulingv1beta1.SchemeGroupVersion.WithResource("priorityclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Scheduling().V1beta1().PriorityClasses().Informer()}, nil

		// Group=storage.k8s.io, Version=v1
	case storagev1.SchemeGroupVersion.WithResource("csidrivers"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1().CSIDrivers().Informer()}, nil
	case storagev1.SchemeGroupVersion.WithResource("csinodes"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1().CSINodes().Informer()}, nil
	case storagev1.SchemeGroupVersion.WithResource("csistoragecapacities"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1().CSIStorageCapacities().Informer()}, nil
	case storagev1.SchemeGroupVersion.WithResource("storageclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1().StorageClasses().Informer()}, nil
	case storagev1.SchemeGroupVersion.WithResource("volumeattachments"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1().VolumeAttachments().Informer()}, nil

		// Group=storage.k8s.io, Version=v1alpha1
	case storagev1alpha1.SchemeGroupVersion.WithResource("csistoragecapacities"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1alpha1().CSIStorageCapacities().Informer()}, nil
	case storagev1alpha1.SchemeGroupVersion.WithResource("volumeattachments"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1alpha1().VolumeAttachments().Informer()}, nil

		// Group=storage.k8s.io, Version=v1beta1
	case storagev1beta1.SchemeGroupVersion.WithResource("csidrivers"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1beta1().CSIDrivers().Informer()}, nil
	case storagev1beta1.SchemeGroupVersion.WithResource("csinodes"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1beta1().CSINodes().Informer()}, nil
	case storagev1beta1.SchemeGroupVersion.WithResource("csistoragecapacities"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1beta1().CSIStorageCapacities().Informer()}, nil
	case storagev1beta1.SchemeGroupVersion.WithResource("storageclasses"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1beta1().StorageClasses().Informer()}, nil
	case storagev1beta1.SchemeGroupVersion.WithResource("volumeattachments"):
		return &GenericInformer{resource: resource.GroupResource(), informer: f.Storage().V1beta1().VolumeAttachments().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
