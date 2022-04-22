package mutating

import (
	"k8s.io/apiserver/pkg/admission/plugin/webhook/generic"
	webhookutil "k8s.io/apiserver/pkg/util/webhook"
)

func NewMutatingDispatcher(p *Plugin) func(cm *webhookutil.ClientManager) generic.Dispatcher {
	return newMutatingDispatcher(p)
}
