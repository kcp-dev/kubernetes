/*
Copyright 2022 The KCP Authors.

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

package logicalcluster

import (
	"encoding/json"
	"path"
	"strings"
)

// LogicalCluster is the name of a logical cluster. A logical cluster is
// 1. a (part of) etcd prefix to store objects in that cluster
// 2. a (part of) a http path which serves a Kubernetes-cluster-like API with
//    discovery, OpenAPI and the actual API groups.
// 3. a value in metadata.clusterName in objects from cross-workspace list/watches,
//    which is used to identify the logical cluster.
//
// A logical cluster is a colon separated list of words. In other words, it is
// like a path, but with colons instead of slashes.
type LogicalCluster struct {
	value string
}

const seperator = ":"

var (
	// Wildcard is the logical cluster indicating cross-workspace requests.
	Wildcard = New("*")
)

// New returns a logical cluster from a string.
func New(value string) LogicalCluster {
	return LogicalCluster{value}
}

// Empty returns true if the logical cluster is unset.
func (l LogicalCluster) Empty() bool {
	return l.value == ""
}

// Path returns a path segment for the logical cluster to access its API.
func (cn LogicalCluster) Path() string {
	return path.Join("/clusters", cn.value)
}

// String returns the string representation of the logical cluster name.
func (cn LogicalCluster) String() string {
	return cn.value
}

// Object is a local interface representation of the Kubernetes metav1.Object, to avoid dependencies on
// k8s.io/apimachinery.
type Object interface {
	GetClusterName() string
}

// From returns a logical cluster name from an Object's
// metadata.clusterName.
func From(obj Object) LogicalCluster {
	return LogicalCluster{obj.GetClusterName()}
}

// Parent returns the parent logical cluster name of the given logical cluster name.
func (cn LogicalCluster) Parent() (LogicalCluster, bool) {
	parent, _ := cn.Split()
	return parent, parent.value != ""
}

// Split splits logical cluster immediately following the final colon,
// separating it into a parent logical cluster and name component.
// If there is no colon in path, Split returns an empty logical cluster name
// and name set to path.
// The returned values have the property that lcn = dir+file.
func (cn LogicalCluster) Split() (parent LogicalCluster, name string) {
	i := strings.LastIndex(cn.value, seperator)
	if i < 0 {
		return LogicalCluster{}, cn.value
	}
	return LogicalCluster{cn.value[:i]}, cn.value[i+1:]
}

// Base returns the last component of the logical cluster name.
func (cn LogicalCluster) Base() string {
	_, name := cn.Split()
	return name
}

// Join joins a parent logical cluster name and a name component.
func (cn LogicalCluster) Join(name string) LogicalCluster {
	if cn.value == "" {
		return LogicalCluster{name}
	}
	return LogicalCluster{cn.value + seperator + name}
}

func (cn LogicalCluster) MarshalJSON() ([]byte, error) {
	return json.Marshal(&cn.value)
}

func (cn *LogicalCluster) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	cn.value = s
	return nil
}

func (cn LogicalCluster) HasPrefix(other LogicalCluster) bool {
	return strings.HasPrefix(cn.value, other.value)
}
