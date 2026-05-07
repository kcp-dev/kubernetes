package generic

import (
	"k8s.io/apiserver/pkg/admission/plugin/cel"
	"k8s.io/apiserver/pkg/cel/environment"
	coreinformers "k8s.io/client-go/informers/core/v1"
)

func (a *Webhook) SetNamespaceInformer(namespaceInformer coreinformers.NamespaceInformer) {
	a.namespaceMatcher.NamespaceLister = namespaceInformer.Lister()
	a.namespaceInformer = namespaceInformer
}

func (a *Webhook) SetAPISource(apiSource Source) {
	a.apiSource = apiSource
}

var sharedFilterCompiler = cel.NewConditionCompiler(environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion()))
