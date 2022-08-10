#!/usr/bin/env bash

# set -o errexit
# set -o nounset
# set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${KUBE_ROOT}/hack/lib/init.sh"

# kube::golang::setup_env

# go install github.com/kcp-dev/code-generator
kcpcodegen=$(kube::util::find-binary "code-generator")

IFS=" " read -r -a GROUP_VERSIONS <<< "${KUBE_AVAILABLE_GROUP_VERSIONS}"
GV_DIRS=()
for gv in "${GROUP_VERSIONS[@]}"; do
  # add items, but strip off any leading apis/ you find to match command expectations
  api_dir=$(kube::util::group-version-to-pkg-path "${gv}")
  nopkg_dir=${api_dir#pkg/}
  nopkg_dir=${nopkg_dir#vendor/k8s.io/api/}
  pkg_dir=${nopkg_dir#apis/}


  # skip groups that aren't being served, clients for these don't matter
  if [[ " ${KUBE_NONSERVER_GROUP_VERSIONS} " == *" ${gv} "* ]]; then
    continue
  fi

  GV_DIRS+=("${pkg_dir}")
done
# delimit by commas for the command
GVS=$(echo "${GV_DIRS[*]// /,}" | sed 's|/|:|g' )

set +e
${kcpcodegen} informer,lister \
  --output-dir staging/src/k8s.io/client-go/kcp \
  --clientset-api-path k8s.io/client-go/kubernetes \
  --listers-package k8s.io/client-go/listers \
  --informers-package k8s.io/client-go/informers \
  --informers-internal-interfaces-package k8s.io/client-go/informers/internalinterfaces \
  --input-dir staging/src/k8s.io/api \
  --go-header-file "${KUBE_ROOT}/hack/boilerplate/boilerplate.generatego.txt" \
  --group-versions $(echo ${GVS} | sed 's/ / --group-versions /g')

${kcpcodegen} informer,lister \
  --output-dir staging/src/k8s.io/apiextensions-apiserver/pkg/client/kcp \
  --clientset-api-path k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset \
  --listers-package k8s.io/apiextensions-apiserver/pkg/client/listers \
  --informers-package k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions \
  --input-dir staging/src/k8s.io/apiextensions-apiserver/pkg/apis \
  --go-header-file "${KUBE_ROOT}/hack/boilerplate/boilerplate.generatego.txt" \
  --group-versions "apiextensions:v1beta1,v1"
