#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

kind=/home/stevekuznetsov/code/kubernetes-sigs/kind/bin/kind
kinder=/home/stevekuznetsov/code/kubernetes/kubeadm/bin/kinder

"${kind}" --loglevel trace delete cluster

if [[ -n "${DELETE:-}" ]]; then
	exit 0
fi

if [[ -n "${BASE:-}" ]]; then
	pushd /home/stevekuznetsov/code/kubernetes-sigs/kind/src/sigs.k8s.io/kind
	IMAGE=localhost/kindest/base:latest make -C ./images/base/ quick
	popd
fi

if [[ -n "${NODE:-}" ]]; then
	pushd /home/stevekuznetsov/code/kcp-dev/kubernetes/src/github.com/kcp-dev/kubernetes
	"${kind}" build node-image --image skuznets/node:latest --kube-root /home/stevekuznetsov/code/kcp-dev/kubernetes/src/github.com/kcp-dev/kubernetes --base-image localhost/kindest/base:latest
	popd
fi

cleanup() {
	for job in $( jobs -p ); do
		kill $job
	done
	wait
}
trap cleanup EXIT

rm -rf /tmp/kind
mkdir /tmp/kind
unit_logs() {
	unit=$1
	while true; do
		podman exec kind-control-plane journalctl --all --lines all -xefu "${unit}" >>/tmp/kind/${unit}.log 2>&1
	done
}
unit_logs kubelet &
unit_logs containerd &

cri_ps() {
	while true; do
		podman exec kind-control-plane crictl ps -a >>/tmp/kind/crictl.log 2>&1
		sleep 1
	done
}
cri_ps &

container_log() {
	while true; do
		label=$1
		container=$( podman exec kind-control-plane crictl ps --label=io.kubernetes.container.name="${label}" -o json | jq '.containers[0].id' --raw-output )
		if [[ -n "${container}" && "${container}" != "null" ]]; then
			podman exec kind-control-plane crictl logs -f "${container}" >>/tmp/kind/${label}.log 2>&1
		fi
		sleep 1
	done
}
container_log etcd &
container_log kube-apiserver &

kube_objects() {
	item=$1
	while true; do
		if podman exec kind-control-plane kubectl get ns --kubeconfig=/etc/kubernetes/admin.conf >/dev/null 2>&1 ; then
			podman exec kind-control-plane kubectl get "${item}" --kubeconfig=/etc/kubernetes/admin.conf --all-namespaces -o yaml -w >> /tmp/kind/"${item}".log 2>&1
		fi
		sleep 1
	done
}
kube_objects pods &
kube_objects events &

cat<<EOF >/tmp/kind/cluster.conf
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    etcd:
      crdb: true
EOF

"${kind}" --loglevel trace create cluster --image localhost/skuznets/node:latest --config /tmp/kind/cluster.conf --kubeconfig /tmp/kind/kube.conf

# "${kinder}" --loglevel trace do kubeadm-init

sleep 3600