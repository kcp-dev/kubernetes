/*
Copyright 2017 The Kubernetes Authors.

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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/printers"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"
	printerstorage "k8s.io/kubernetes/pkg/printers/storage"
)

type TableConverterFunc func(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error)

func (tcf TableConverterFunc) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return tcf(ctx, object, tableOptions)
}

// HACK: Currently CRDs only allow defining custom table column based on a single basic JsonPath expression.
// This is not sufficient to reproduce the various colum definitions of legacy scheme objects like
// deployments, etc ..., since those definitions are implemented in Go code.
// So for example in KCP, when deployments are brought back under the form of a CRD, the table columns
// shown from a `kubectl get deployments` command are not the ones typically expected.
//
// The `replaceTableConverterForLegacySchemaResources` function is a temporary hack to replace the table converter of
// CRDs that are related to legacy-schema resources, with the default table converter of the related legacy scheme resource.
//
// In the future this should probably be replaced by some new mechanism that would allow customizing some
// behaviors of resources defined by CRDs.
func replaceTableConverterForLegacySchemaResources(kind schema.GroupVersionKind, crd *apiextensionsv1.CustomResourceDefinition) rest.TableConvertor {
	legacySchemeTableConvertor := printerstorage.TableConvertor{TableGenerator: printers.NewTableGenerator().With(printersinternal.AddHandlers)}
	objectType, objectTypeExists := legacyscheme.Scheme.AllKnownTypes()[schema.GroupVersionKind{
		Group:   kind.Group,
		Kind:    crd.Spec.Names.Kind,
		Version: runtime.APIVersionInternal,
	}]
	listType, listTypeExists := legacyscheme.Scheme.AllKnownTypes()[schema.GroupVersionKind{
		Group:   kind.Group,
		Kind:    crd.Spec.Names.ListKind,
		Version: runtime.APIVersionInternal,
	}]
	if objectTypeExists && listTypeExists {
		return TableConverterFunc(func(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
			k := object.GetObjectKind().GroupVersionKind()
			var theType reflect.Type
			switch k.Kind {
			case crd.Spec.Names.Kind:
				theType = objectType
			default:
				theType = listType
			}
			out := reflect.New(theType).Interface().(runtime.Object)
			legacyscheme.Scheme.Convert(object, out, nil)
			return legacySchemeTableConvertor.ConvertToTable(ctx, out, tableOptions)
		})
	}
	return nil
}
