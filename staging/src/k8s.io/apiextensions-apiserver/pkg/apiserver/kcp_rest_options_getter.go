package apiserver

import (
	"fmt"
	"strings"

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

	// Priority 1: wildcard partial metadata requests. These have been assigned a fake UID that ends with
	// .wildcard.partial-metadata. If this is present, we don't want to modify the ResourcePrefix, which means that
	// a wildcard partial metadata list/watch request will return every CR from every CRD for that group-resource, which
	// could include instances from normal CRDs as well as those coming from CRDs with different identities. This would
	// return e.g. everything under
	//
	//   - /registry/mygroup.io/widgets/customresources/...
	//   - /registry/mygroup.io/widgets/identity1234/...
	//   - /registry/mygroup.io/widgets/identity4567/...
	if strings.HasSuffix(string(t.crd.UID), ".wildcard.partial-metadata") {
		return ret, nil
	}

	// Normal CRDs (not coming from an APIBinding) are stored in e.g. /registry/mygroup.io/widgets/customresources/...
	if _, bound := t.crd.Annotations["apis.kcp.dev/bound-crd"]; !bound {
		ret.ResourcePrefix += "/customresources"
		return ret, nil
	}

	// Bound CRDs must have the associated identity annotation
	apiIdentity := t.crd.Annotations["apis.kcp.dev/identity"]
	if apiIdentity == "" {
		return generic.RESTOptions{}, fmt.Errorf("missing 'apis.kcp.dev/identity' annotation")
	}

	// Modify the ResourcePrefix so it results in e.g. /registry/mygroup.io/widgets/identity4567/...
	ret.ResourcePrefix += "/" + apiIdentity

	return ret, err
}
