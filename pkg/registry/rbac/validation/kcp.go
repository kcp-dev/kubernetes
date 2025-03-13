package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/kcp-dev/logicalcluster/v3"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	authserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authentication/user"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/features"
)

const (
	// WarrantExtraKey is the key used in a user's "extra" to specify
	// JSON-encoded user infos for attached extra permissions for that user
	// evaluated by the authorizer.
	WarrantExtraKey = "authorization.kcp.io/warrant"

	// ScopeExtraKey is the key used in a user's "extra" to specify
	// that the user is restricted to a given scope. Valid values for
	// one extra value are:
	// - "cluster:<name>"
	// - "cluster:<name1>,cluster:<name2>"
	// - etc.
	// The clusters in one extra value are or'ed, multiple extra values
	// are and'ed.
	ScopeExtraKey = "authentication.kcp.io/scopes"

	// ClusterPrefix is the prefix for cluster scopes.
	clusterPrefix = "cluster:"
)

// Warrant is serialized into the user's "extra" field authorization.kcp.io/warrant
// to hold user information for extra permissions.
type Warrant struct {
	// User is the user you're testing for.
	// If you specify "User" but not "Groups", then is it interpreted as "What if User were not a member of any groups
	// +optional
	User string `json:"user,omitempty"`
	// Groups is the groups you're testing for.
	// +optional
	// +listType=atomic
	Groups []string `json:"groups,omitempty"`
	// Extra corresponds to the user.Info.GetExtra() method from the authenticator.  Since that is input to the authorizer
	// it needs a reflection here.
	// +optional
	Extra map[string][]string `json:"extra,omitempty"`
	// UID information about the requesting user.
	// +optional
	UID string `json:"uid,omitempty"`
}

type appliesToUserFunc func(user user.Info, subject rbacv1.Subject, namespace string) bool
type appliesToUserFuncCtx func(ctx context.Context, user user.Info, subject rbacv1.Subject, namespace string) bool

var appliesToUserWithScopedAndWarrants = withScopesAndWarrants(appliesToUser)

// withScopesAndWarrants flattens the warrants, applies scopes and then applies the users to the subjects.
func withScopesAndWarrants(appliesToUser appliesToUserFunc) appliesToUserFuncCtx {
	return func(ctx context.Context, u user.Info, bindingSubject rbacv1.Subject, namespace string) bool {
		var clusterName logicalcluster.Name
		if cluster := genericapirequest.ClusterFrom(ctx); cluster != nil {
			clusterName = cluster.Name
		}

		for _, eu := range EffectiveUsers(clusterName, u) {
			if appliesToUser(eu, bindingSubject, namespace) {
				return true
			}
		}

		return false
	}
}

var (
	authenticated   = &user.DefaultInfo{Name: user.Anonymous, Groups: []string{user.AllAuthenticated}}
	unauthenticated = &user.DefaultInfo{Name: user.Anonymous, Groups: []string{user.AllUnauthenticated}}
)

// EffectiveUsers flattens the warrants and scopes each user to the given cluster.
func EffectiveUsers(clusterName logicalcluster.Name, u user.Info) []user.Info {
	ret := make([]user.Info, 0, 2)

	var wantAuthenticated bool
	var wantUnauthenticated bool
	globalsa := utilfeature.DefaultFeatureGate.Enabled(features.GlobalServiceAccount)

	var recursive func(u user.Info)
	recursive = func(u user.Info) {
		if IsInScope(u, clusterName) {
			ret = append(ret, u)
		} else {
			found := false
			for _, g := range u.GetGroups() {
				if g == user.AllAuthenticated {
					found = true
					break
				}
			}
			wantAuthenticated = wantAuthenticated || found
			wantUnauthenticated = wantUnauthenticated || !found
		}

		if IsServiceAccount(u) && globalsa {
			if clusters := u.GetExtra()[authserviceaccount.ClusterNameKey]; len(clusters) == 1 {
				nsNameSuffix := strings.TrimPrefix(u.GetName(), "system:serviceaccount:")
				rewritten := &user.DefaultInfo{
					Name:  fmt.Sprintf("system:kcp:serviceaccount:%s:%s", clusters[0], nsNameSuffix),
					Extra: u.GetExtra(),
				}
				for _, g := range u.GetGroups() {
					if g == user.AllAuthenticated {
						rewritten.Groups = []string{user.AllAuthenticated}
						break
					}
				}
				ret = append(ret, rewritten)
			}
		}

		for _, v := range u.GetExtra()[WarrantExtraKey] {
			var w Warrant
			if err := json.Unmarshal([]byte(v), &w); err != nil {
				continue
			}

			wu := &user.DefaultInfo{
				Name:   w.User,
				UID:    w.UID,
				Groups: w.Groups,
				Extra:  w.Extra,
			}
			if IsServiceAccount(wu) && len(w.Extra[authserviceaccount.ClusterNameKey]) == 0 {
				// warrants must be scoped to a cluster
				continue
			}
			recursive(wu)
		}
	}
	recursive(u)

	if wantAuthenticated {
		ret = append(ret, authenticated)
	}
	if wantUnauthenticated {
		ret = append(ret, unauthenticated)
	}

	return ret
}

// IsServiceAccount returns true if the user is a service account.
func IsServiceAccount(attr user.Info) bool {
	return strings.HasPrefix(attr.GetName(), "system:serviceaccount:")
}

// IsForeign returns true if the service account is not from the given cluster.
func IsForeign(attr user.Info, cluster logicalcluster.Name) bool {
	clusters := attr.GetExtra()[authserviceaccount.ClusterNameKey]
	switch {
	case len(clusters) == 0:
		// an unqualified service account is considered local: think of some
		// local SubjectAccessReview specifying a service account without the
		// cluster scope.
		return false
	case len(clusters) != 1:
		return true
	default:
		return !sets.New(clusters...).Has(string(cluster))
	}
}

// IsInScope checks if the user is valid for the given cluster.
func IsInScope(attr user.Info, cluster logicalcluster.Name) bool {
	if IsServiceAccount(attr) && IsForeign(attr, cluster) {
		return false
	}

	values := attr.GetExtra()[ScopeExtraKey]
	for _, scopes := range values {
		found := false
		for _, scope := range strings.Split(scopes, ",") {
			if strings.HasPrefix(scope, clusterPrefix) && scope[len(clusterPrefix):] == string(cluster) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// EffectiveGroups returns the effective groups of the user in the given context
// taking scopes and warrants into account.
func EffectiveGroups(ctx context.Context, u user.Info) sets.Set[string] {
	var clusterName logicalcluster.Name
	if cluster := genericapirequest.ClusterFrom(ctx); cluster != nil {
		clusterName = cluster.Name
	}

	eus := EffectiveUsers(clusterName, u)
	groups := sets.New[string]()
	for _, eu := range eus {
		groups.Insert(eu.GetGroups()...)
	}

	return groups
}

// PrefixUser returns a new user with the name and groups prefixed with the
// given prefix, and all warrants recursively prefixed.
//
// If the user is a service account, the prefix is added to the global service
// account name.
//
// Invalid warrants are skipped.
func PrefixUser(u user.Info, prefix string) user.Info {
	pu := &user.DefaultInfo{
		Name: prefix + u.GetName(),
		UID:  u.GetUID(),
	}
	if IsServiceAccount(u) {
		if clusters := u.GetExtra()[authserviceaccount.ClusterNameKey]; len(clusters) != 1 {
			// this should not happen. But if it does, we are defensive.
			for _, g := range u.GetGroups() {
				if g == user.AllAuthenticated {
					return &user.DefaultInfo{Name: prefix + user.Anonymous, Groups: []string{prefix + user.AllAuthenticated}}
				}
			}
			return &user.DefaultInfo{Name: prefix + user.Anonymous, Groups: []string{prefix + user.AllUnauthenticated}}
		} else {
			pu.Name = fmt.Sprintf("%ssystem:kcp:serviceaccount:%s:%s", prefix, clusters[0], strings.TrimPrefix(u.GetName(), "system:serviceaccount:"))
		}
	}

	for _, g := range u.GetGroups() {
		pu.Groups = append(pu.Groups, prefix+g)
	}

	for k, v := range u.GetExtra() {
		if k == WarrantExtraKey {
			continue
		}
		if pu.Extra == nil {
			pu.Extra = map[string][]string{}
		}
		pu.Extra[k] = v
	}

	for _, w := range u.GetExtra()[WarrantExtraKey] {
		var warrant Warrant
		if err := json.Unmarshal([]byte(w), &warrant); err != nil {
			continue // skip invalid warrant
		}

		wpu := PrefixUser(&user.DefaultInfo{Name: warrant.User, UID: warrant.UID, Groups: warrant.Groups, Extra: warrant.Extra}, prefix)
		warrant = Warrant{User: wpu.GetName(), UID: wpu.GetUID(), Groups: wpu.GetGroups(), Extra: wpu.GetExtra()}

		bs, err := json.Marshal(warrant)
		if err != nil {
			continue // skip invalid warrant
		}

		if pu.Extra == nil {
			pu.Extra = map[string][]string{}
		}
		pu.Extra[WarrantExtraKey] = append(pu.Extra[WarrantExtraKey], string(bs))
	}

	return pu
}
