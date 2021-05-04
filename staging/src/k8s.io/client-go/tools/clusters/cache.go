package clusters

import "strings"

// ToClusterAwareKey allows combining the object name and
// the object cluster in a single key that can be used by informers.
// This is KCP-related hack useful when watching across several
// logical clusters using a wildcard context cluster
//
// This is a temporary hack and should be replaced by thoughtful
// and real support of logical cluster in the client-go layer
func ToClusterAwareKey(clusterName, name string) string {
	if clusterName != "" {
		return clusterName + "#$#" + name
	}

	return name
}

// ToClusterAwareKey just allows extract the name and clusterName
// from a Key initially created with ToClusterAwareKey
func SplitClusterAwareKey(clusterAwareKey string) (clusterName, name string) {
	parts := strings.SplitN(clusterAwareKey, "#$#", 2)
	if len(parts) == 1 {
		// name only, no cluster
		return "", parts[0]
	}
	// clusterName and name
	return parts[0], parts[1]
}
