 #!/bin/bash

set -e

code-generator informer,lister \
    --go-header-file hack/boilerplate/boilerplate.generatego.txt \
    --clientset-api-path k8s.io/client-go/kubernetes \
    --listers-package k8s.io/client-go/listers \
    --informers-package k8s.io/client-go/informers \
    --informers-internal-interfaces-package k8s.io/client-go/informers/internalinterfaces \
    --input-dir staging/src/k8s.io/api \
    --output-dir staging/src/k8s.io/client-go/kcp \
    --group-versions storage:v1beta1 \
    --group-versions scheduling:v1alpha1 \
    --group-versions apiserverinternal:v1alpha1 \
    --group-versions core:v1 \
    --group-versions admission:v1 \
    --group-versions admission:v1beta1 \
    --group-versions admissionregistration:v1 \
    --group-versions admissionregistration:v1beta1 \
    --group-versions apps:v1 \
    --group-versions apps:v1beta1 \
    --group-versions apps:v1beta2 \
    --group-versions authentication:v1 \
    --group-versions authentication:v1beta1 \
    --group-versions authorization:v1 \
    --group-versions authorization:v1beta1 \
    --group-versions autoscaling:v1 \
    --group-versions autoscaling:v2 \
    --group-versions autoscaling:v2beta1 \
    --group-versions autoscaling:v2beta2 \
    --group-versions batch:v1 \
    --group-versions batch:v1beta1 \
    --group-versions certificates:v1 \
    --group-versions certificates:v1beta1 \
    --group-versions coordination:v1beta1 \
    --group-versions coordination:v1 \
    --group-versions discovery:v1 \
    --group-versions discovery:v1beta1 \
    --group-versions extensions:v1beta1 \
    --group-versions events:v1 \
    --group-versions events:v1beta1 \
    --group-versions imagepolicy:v1alpha1 \
    --group-versions networking:v1 \
    --group-versions networking:v1beta1 \
    --group-versions node:v1 \
    --group-versions node:v1alpha1 \
    --group-versions node:v1beta1 \
    --group-versions policy:v1 \
    --group-versions policy:v1beta1 \
    --group-versions rbac:v1 \
    --group-versions rbac:v1beta1 \
    --group-versions rbac:v1alpha1 \
    --group-versions scheduling:v1beta1 \
    --group-versions scheduling:v1 \
    --group-versions scheduling:v1alpha1 \
    --group-versions storage:v1 \
    --group-versions flowcontrol:v1alpha1 \
    --group-versions flowcontrol:v1beta1 \
    --group-versions flowcontrol:v1beta2

code-generator informer,lister \
    --go-header-file hack/boilerplate/boilerplate.generatego.txt \
    --clientset-api-path k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset \
    --listers-package k8s.io/apiextensions-apiserver/pkg/client/listers \
    --informers-package k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions \
    --informers-internal-interfaces-package k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions/internalinterfaces \
    --input-dir staging/src/k8s.io/apiextensions-apiserver/pkg/apis \
    --output-dir staging/src/k8s.io/apiextensions-apiserver/pkg/client/kcp \
    --group-versions "apiextensions:v1beta1,v1"
