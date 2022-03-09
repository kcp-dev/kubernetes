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

package etcd

import (
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	utilsnet "k8s.io/utils/net"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/features"
	"k8s.io/kubernetes/cmd/kubeadm/app/images"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	etcdutil "k8s.io/kubernetes/cmd/kubeadm/app/util/etcd"
	staticpodutil "k8s.io/kubernetes/cmd/kubeadm/app/util/staticpod"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/users"
)

// CreateLocalCrdbStaticPodManifestFile will write local CRDB static pod manifest file.
// This function is used by init - when the CRDB cluster is empty - or by kubeadm
// upgrade - when the CRDB cluster is already up and running (and the --initial-cluster flag have no impact)
func CreateLocalCrdbStaticPodManifestFile(manifestDir, patchesDir string, nodeName string, cfg *kubeadmapi.ClusterConfiguration, endpoint *kubeadmapi.APIEndpoint, isDryRun bool) error {
	if cfg.Etcd.External != nil {
		return errors.New("etcd static pod manifest cannot be generated for cluster using external etcd")
	}

	if err := prepareAndWriteCrdbStaticPod(manifestDir, patchesDir, cfg, endpoint, nodeName, []etcdutil.Member{}, isDryRun); err != nil {
		return err
	}

	klog.V(1).Infof("[etcd] wrote Static Pod manifest for a local etcd member to %q\n", kubeadmconstants.GetStaticPodFilepath(kubeadmconstants.Etcd, manifestDir))
	return nil
}

// CheckLocalCrdbClusterStatus verifies health state of local/stacked crdb cluster before installing a new crdb member
func CheckLocalCrdbClusterStatus(client clientset.Interface, certificatesDir string) error {
	return errors.New("CheckLocalCrdbClusterStatus: not yet implemented!")
}

// RemoveStackedCrdbMemberFromCluster will remove a local crdb member from crdb cluster,
// when reset the control plane node.
func RemoveStackedCrdbMemberFromCluster(client clientset.Interface, cfg *kubeadmapi.InitConfiguration) error {
	return errors.New("RemoveStackedCrdbMemberFromCluster: not yet implemented!")
}

// CreateStackedCrdbStaticPodManifestFile will write local crdb static pod manifest file
// for an additional crdb member that is joining an existing local/stacked crdb cluster.
// Other members of the crdb cluster will be notified of the joining node in beforehand as well.
func CreateStackedCrdbStaticPodManifestFile(client clientset.Interface, manifestDir, patchesDir string, nodeName string, cfg *kubeadmapi.ClusterConfiguration, endpoint *kubeadmapi.APIEndpoint, isDryRun bool, certificatesDir string) error {
	return errors.New("CreateStackedCrdbStaticPodManifestFile: not yet implemented!")
}

// GetCrdbPodSpec returns the crdb static Pod actualized to the context of the current configuration
// NB. GetCrdbPodSpec methods holds the information about how kubeadm creates etcd static pod manifests.
func GetCrdbPodSpec(cfg *kubeadmapi.ClusterConfiguration, endpoint *kubeadmapi.APIEndpoint, nodeName string, initialCluster []etcdutil.Member) v1.Pod {
	pathType := v1.HostPathDirectoryOrCreate
	etcdMounts := map[string]v1.Volume{
		etcdVolumeName:  staticpodutil.NewVolume(etcdVolumeName, cfg.Etcd.Local.DataDir, &pathType),
		certsVolumeName: staticpodutil.NewVolume(certsVolumeName, cfg.CertificatesDir+"/etcd", &pathType),
	}
	// probeHostname returns the correct localhost IP address family based on the endpoint AdvertiseAddress
	probeHostname, probePort, probeScheme := staticpodutil.GetEtcdProbeEndpoint(&cfg.Etcd, utilsnet.IsIPv6String(endpoint.AdvertiseAddress))
	return staticpodutil.ComponentPod(
		v1.Container{
			Name:            kubeadmconstants.Etcd,
			Command:         getCrdbCommand(cfg, endpoint, nodeName, initialCluster),
			Image:           images.GetEtcdImage(cfg),
			ImagePullPolicy: v1.PullIfNotPresent,
			// Mount the etcd datadir path read-write so etcd can store data in a more persistent manner
			VolumeMounts: []v1.VolumeMount{
				staticpodutil.NewVolumeMount(etcdVolumeName, cfg.Etcd.Local.DataDir, false),
				staticpodutil.NewVolumeMount(certsVolumeName, cfg.CertificatesDir+"/etcd", false),
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("100m"),
					v1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
			LivenessProbe: staticpodutil.LivenessProbe(probeHostname, "/health", probePort, probeScheme),
			StartupProbe:  staticpodutil.StartupProbe(probeHostname, "/health", probePort, probeScheme, cfg.APIServer.TimeoutForControlPlane),
			Env: []v1.EnvVar{
				{
					Name:  "COCKROACH_CHANNEL",
					Value: "kubernetes-secure",
				},
				{
					Name: "GOMAXPROCS",
					ValueFrom: &v1.EnvVarSource{
						ResourceFieldRef: &v1.ResourceFieldSelector{
							Resource: "limits.cpu",
							Divisor:  resource.MustParse("1"),
						},
					},
				},
				{
					Name: "MEMORY_LIMIT_MIB",
					ValueFrom: &v1.EnvVarSource{
						ResourceFieldRef: &v1.ResourceFieldSelector{
							Resource: "limits.memory",
							Divisor:  resource.MustParse("1Mi"),
						},
					},
				},
			},
		},
		etcdMounts,
		// etcd will listen on the advertise address of the API server, in a different port (2379)
		map[string]string{kubeadmconstants.EtcdAdvertiseClientUrlsAnnotationKey: etcdutil.GetClientURL(endpoint)},
	)
}

// getCrdbCommand builds the right etcd command from the given config object
func getCrdbCommand(cfg *kubeadmapi.ClusterConfiguration, endpoint *kubeadmapi.APIEndpoint, nodeName string, initialCluster []etcdutil.Member) []string {
	// localhost IP family should be the same that the AdvertiseAddress
	etcdLocalhostAddress := "127.0.0.1"
	if utilsnet.IsIPv6String(endpoint.AdvertiseAddress) {
		etcdLocalhostAddress = "::1"
	}
	clientHostPort := net.JoinHostPort(endpoint.AdvertiseAddress, strconv.Itoa(constants.EtcdListenClientPort))
	peerHostPort := net.JoinHostPort(endpoint.AdvertiseAddress, strconv.Itoa(constants.EtcdListenPeerPort))
	defaultArguments := map[string]string{
		"cluster-name": nodeName, // name
		"logtostderr":  "true",

		"certs-dir": path.Join(cfg.CertificatesDir, "etcd"),    // {cert,key,trusted-ca}-file (had to rename to node.crt etc)
		"store":     cfg.Etcd.Local.DataDir, // data-dir

		"listen-addr":    peerHostPort, // listen-peer-urls
		"advertise-addr": peerHostPort, // initial-advertise-peer-urls

		"sql-addr":           clientHostPort, // listen-client-urls (etcd has both localhost and external here ... ?)
		"advertise-sql-addr": clientHostPort, // advertise-client-urls

		"http-addr": net.JoinHostPort(etcdLocalhostAddress, strconv.Itoa(kubeadmconstants.EtcdMetricsPort)), // listen-metrics-urls

		"cache":          "$(expr $MEMORY_LIMIT_MIB / 4)MiB",
		"max-sql-memory": "$(expr $MEMORY_LIMIT_MIB / 4)MiB",
	}

	if len(initialCluster) != 0 {
		var peers []string
		for _, peer := range initialCluster {
			peers = append(peers, peer.PeerURL)
		}
		defaultArguments["join"] = strings.Join(peers, ",")
	}

	command := []string{
		"/bin/bash", "-ecx",
		strings.Join(append([]string{
			"exec", "/cockroach/cockroach", "start-single-node", // TODO: use start, but then figure out the `init` stuff?
		}, kubeadmutil.BuildArgumentListFromMap(defaultArguments, cfg.Etcd.Local.ExtraArgs)...), " "),
	}
	return command
}

func prepareAndWriteCrdbStaticPod(manifestDir string, patchesDir string, cfg *kubeadmapi.ClusterConfiguration, endpoint *kubeadmapi.APIEndpoint, nodeName string, initialCluster []etcdutil.Member, isDryRun bool) error {
	// gets etcd StaticPodSpec, actualized for the current ClusterConfiguration and the new list of etcd members
	spec := GetCrdbPodSpec(cfg, endpoint, nodeName, initialCluster)

	var usersAndGroups *users.UsersAndGroups
	var err error
	if features.Enabled(cfg.FeatureGates, features.RootlessControlPlane) {
		if isDryRun {
			fmt.Printf("[dryrun] Would create users and groups for %q to run as non-root\n", kubeadmconstants.Etcd)
			fmt.Printf("[dryrun] Would update static pod manifest for %q to run run as non-root\n", kubeadmconstants.Etcd)
		} else {
			usersAndGroups, err = staticpodutil.GetUsersAndGroups()
			if err != nil {
				return errors.Wrap(err, "failed to create users and groups")
			}
			// usersAndGroups is nil on non-linux.
			if usersAndGroups != nil {
				if err := staticpodutil.RunComponentAsNonRoot(kubeadmconstants.Etcd, &spec, usersAndGroups, cfg); err != nil {
					return errors.Wrapf(err, "failed to run component %q as non-root", kubeadmconstants.Etcd)
				}
			}
		}
	}

	// if patchesDir is defined, patch the static Pod manifest
	if patchesDir != "" {
		patchedSpec, err := staticpodutil.PatchStaticPod(&spec, patchesDir, os.Stdout)
		if err != nil {
			return errors.Wrapf(err, "failed to patch static Pod manifest file for %q", kubeadmconstants.Etcd)
		}
		spec = *patchedSpec
	}

	// writes etcd StaticPod to disk
	if err := staticpodutil.WriteStaticPodToDisk(kubeadmconstants.Etcd, manifestDir, spec); err != nil {
		return err
	}

	return nil
}
