package apiserver

import (
	"fmt"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
)

type apiBindingAwareCRDRESTOptionsGetter struct {
	delegate generic.RESTOptionsGetter
	crd      *apiextensionsv1.CustomResourceDefinition
}

func (t apiBindingAwareCRDRESTOptionsGetter) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	ret, err := t.delegate.GetRESTOptions(resource)
	if err != nil {
		return ret, err
	}

	if _, bound := t.crd.Annotations["apis.kcp.dev/bound-crd"]; !bound {
		return ret, nil
	}

	apiIdentity := t.crd.Annotations["apis.kcp.dev/identity"]
	if apiIdentity == "" {
		return generic.RESTOptions{}, fmt.Errorf("missing 'apis.kcp.dev/identity' annotation")
	}

	// Use a : to separate as it is a character that is not valid in CRD.spec.names.plural
	ret.ResourcePrefix += ":" + apiIdentity

	return ret, err
}
