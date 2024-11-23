package validation

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kcp-dev/logicalcluster/v3"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	authserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authentication/user"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

const (
	// WarrantExtraKey is the key used in a user's "extra" to specify
	// JSON-encoded user infos for attached extra permissions for that user
	// evaluated by the authorizer.
	WarrantExtraKey = "authorization.kcp.io/warrants"

	// ScopeExtraKey is the key used in a user's "extra" to specify
	// that the user is restricted to a given scope. Valid values are:
	// - "cluster:<name>"
	// In the future, we might add:
	// - "interval:from:to".
	// Scopes are and'ed. Scoping to multiple clusters invalidates it for all.
	ScopeExtraKey = "authentication.kcp.io/scopes"
)

// Warrant is serialized into the user's "extra" field authorization.kcp.io/warrants
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

var appliesToUserWithWarrants = withWarrants(appliesToUser)

// withWarrants wraps the appliesToUser predicate to check for the base user and any warrants.
func withWarrants(appliesToUser appliesToUserFunc) appliesToUserFuncCtx {
	var recursive appliesToUserFuncCtx
	recursive = func(ctx context.Context, u user.Info, bindingSubject rbacv1.Subject, namespace string) bool {
		cluster := genericapirequest.ClusterFrom(ctx)
		if IsInScope(u, cluster.Name) && appliesToUser(u, bindingSubject, namespace) {
			return true
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
				continue
			}
			if recursive(ctx, wu, bindingSubject, namespace) {
				return true
			}
		}

		return false
	}
	return recursive
}

// IsServiceAccount returns true if the user is a service account.
func IsServiceAccount(attr user.Info) bool {
	return strings.HasPrefix(attr.GetName(), "system:serviceaccount:")
}

// IsForeign returns true if the service account is not from the given cluster.
func IsForeign(attr user.Info, cluster logicalcluster.Name) bool {
	clusters := attr.GetExtra()[authserviceaccount.ClusterNameKey]
	if clusters == nil {
		// an unqualified service account is considered local: think of some
		// local SubjectAccessReview specifying a service account without the
		// cluster scope.
		return false
	}
	return !sets.New(clusters...).Has(string(cluster))
}

// IsInScope checks if the user is valid for the given cluster.
func IsInScope(attr user.Info, cluster logicalcluster.Name) bool {
	if IsServiceAccount(attr) && IsForeign(attr, cluster) {
		return false
	}

	scopes := attr.GetExtra()[ScopeExtraKey]
	for _, scope := range scopes {
		switch {
		case strings.HasPrefix(scope, "cluster:"):
			if scope != "cluster:"+string(cluster) {
				return false
			}
		default:
			// Unknown scope, ignore.
		}
	}

	return true
}
