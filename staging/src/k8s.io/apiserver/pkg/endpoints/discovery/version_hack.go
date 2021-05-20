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

package discovery

import (
	"net/http"
	"regexp"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

// HACK: support the case when we can add core or other legacy scheme resources through CRDs (KCP scenario)
// Possibly we wouldn't need a global variable for ContributedResources that is cluster scoped,
// In the long run we could see the context carrying an injected interface that would let us perform
// some of these cluster scoped behaviors (where we would call methods on it instead of having lots of little caches everywhere).
// Finally with more refactoring we might also just want to skip having the cache and do it dynamically.
var ContributedResources map[ClusterGroupVersion]APIResourceLister = map[ClusterGroupVersion]APIResourceLister{}

type ClusterGroupVersion struct {
	ClusterName string
	Group       string
	Version     string
}

// Empty returns true if group and version are empty
func (cgv ClusterGroupVersion) Empty() bool {
	return len(cgv.Group) == 0 && len(cgv.Version) == 0
}

// String puts "group" and "version" into a single "group/version" string. For the legacy v1
// it returns "v1".
func (cgv ClusterGroupVersion) String() string {
	// special case the internal apiVersion for the legacy kube types
	if cgv.Empty() {
		return ""
	}

	gv := cgv.Group + "/" + cgv.Version
	// special case of "v1" for backward compatibility
	if len(cgv.Group) == 0 && cgv.Version == "v1" {
		gv = cgv.Version
	}
	result := gv
	if cgv.ClusterName != "" {
		result = cgv.ClusterName + "/" + gv
	}
	return result
}

func (cgv ClusterGroupVersion) GroupVersion() schema.GroupVersion {
	return schema.GroupVersion{
		Group:   cgv.Group,
		Version: cgv.Version,
	}
}

func withContributedResources(groupVersion schema.GroupVersion, apiResourceLister APIResourceLister) func(*http.Request) APIResourceLister {
	return func(req *http.Request) APIResourceLister {
		cluster := genericapirequest.ClusterFrom(req.Context())
		return APIResourceListerFunc(func() []metav1.APIResource {
			result := []metav1.APIResource{}
			result = append(result, apiResourceLister.ListAPIResources()...)
			if cluster != nil {
				if additionalResources := ContributedResources[ClusterGroupVersion{
					ClusterName: cluster.Name,
					Group:       groupVersion.Group,
					Version:     groupVersion.Version}]; additionalResources != nil {
					result = append(result, additionalResources.ListAPIResources()...)
				}
				sort.Slice(result, func(i, j int) bool {
					return result[i].Name < result[j].Name
				})
			}
			return result
		})
	}
}

// IsAPIContributed returns `true` is the path corresponds to a resource that
// has been contribued to a legacy scheme group from a CRD.
func IsAPIContributed(path string) bool {
	for gv, resourceLister := range ContributedResources {
		prefix := gv.Group
		if prefix != "" {
			prefix = "/apis/" + prefix + "/" + gv.Version + "/"
		} else {
			prefix = "/api/" + gv.Version + "/"
		}
		if !strings.HasPrefix(path, prefix) {
			continue
		}

		for _, resource := range resourceLister.ListAPIResources() {
			if strings.HasPrefix(path, prefix+resource.Name) {
				return true
			}
			if resource.Namespaced {
				if matched, _ := regexp.MatchString(prefix+"namespaces/[^/][^/]*/"+resource.Name+"(/[^/].*)?", path); matched {
					return true
				}
			}
		}
	}
	return false
}
