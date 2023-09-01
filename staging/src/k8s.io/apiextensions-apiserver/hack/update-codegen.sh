#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

CLIENTSET_NAME_VERSIONED=clientset \
CLIENTSET_PKG_NAME=clientset \
bash "${CODEGEN_PKG}/generate-groups.sh" deepcopy,client,lister,informer \
  k8s.io/apiextensions-apiserver/pkg/client k8s.io/apiextensions-apiserver/pkg/apis \
  "apiextensions:v1beta1,v1" \
  --output-base "$(dirname "${BASH_SOURCE[0]}")/../../.." \
  --go-header-file "${SCRIPT_ROOT}/hack/boilerplate.go.txt"

CLIENTSET_NAME_VERSIONED=clientset \
CLIENTSET_PKG_NAME=clientset \
CLIENTSET_NAME_INTERNAL=internalclientset \
bash "${CODEGEN_PKG}/generate-internal-groups.sh" deepcopy,conversion \
  k8s.io/apiextensions-apiserver/pkg/client k8s.io/apiextensions-apiserver/pkg/apis k8s.io/apiextensions-apiserver/pkg/apis \
  "apiextensions:v1beta1,v1" \
  --output-base "$(dirname "${BASH_SOURCE[0]}")/../../.." \
  --go-header-file "${SCRIPT_ROOT}/hack/boilerplate.go.txt"

OUTPUT_DIR=$(cd "${SCRIPT_ROOT}/../../../../_output/bin" && pwd)
GO111MODULE=on GOBIN="${OUTPUT_DIR}" "${SCRIPT_ROOT}/hack/go-install.sh" github.com/kcp-dev/code-generator/v2 code-generator 7e515e775be8aa62166e65b70e54a704fc7a0630
pushd "${SCRIPT_ROOT}"
GO111MODULE=on "${OUTPUT_DIR}/code-generator" \
  "client:standalone=true,outputPackagePath=k8s.io/apiextensions-apiserver/pkg/client/kcp,name=clientset,apiPackagePath=k8s.io/apiextensions-apiserver/pkg/apis,singleClusterClientPackagePath=k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset,headerFile=${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
  "lister:apiPackagePath=k8s.io/apiextensions-apiserver/pkg/apis,singleClusterListerPackagePath=k8s.io/apiextensions-apiserver/pkg/client/listers,headerFile=${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
  "informer:standalone=true,outputPackagePath=k8s.io/apiextensions-apiserver/pkg/client/kcp,apiPackagePath=k8s.io/apiextensions-apiserver/pkg/apis,singleClusterClientPackagePath=k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset,singleClusterListerPackagePath=k8s.io/apiextensions-apiserver/pkg/client/listers,singleClusterInformerPackagePath=k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions,headerFile=${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
  "paths=./pkg/apis/..." \
  "output:dir=./pkg/client/kcp"
popd
