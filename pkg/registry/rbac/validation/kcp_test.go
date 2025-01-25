package validation

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kcp-dev/logicalcluster/v3"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	authserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func TestIsInScope(t *testing.T) {
	tests := []struct {
		name    string
		info    user.DefaultInfo
		cluster logicalcluster.Name
		want    bool
	}{
		{name: "empty", cluster: logicalcluster.Name("cluster"), want: true},
		{
			name:    "empty scope",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {""}}},
			cluster: logicalcluster.Name("cluster"),
			want:    false,
		},
		{
			name:    "scoped user",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:this"}}},
			cluster: logicalcluster.Name("this"),
			want:    true,
		},
		{
			name:    "scoped user to a different cluster",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:another"}}},
			cluster: logicalcluster.Name("this"),
			want:    false,
		},
		{
			name:    "contradicting scopes",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:this", "cluster:another"}}},
			cluster: logicalcluster.Name("this"),
			want:    false,
		},
		{
			name:    "empty contradicting value",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"", "cluster:this"}}},
			cluster: logicalcluster.Name("cluster"),
			want:    false,
		},
		{
			name:    "unknown scope",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"unknown:foo"}}},
			cluster: logicalcluster.Name("this"),
			want:    false,
		},
		{
			name:    "another or'ed scope",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:another,cluster:this"}}},
			cluster: logicalcluster.Name("this"),
			want:    true,
		},
		{
			name:    "multiple or'ed scopes",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:another,cluster:this", "cluster:this,cluster:other"}}},
			cluster: logicalcluster.Name("this"),
			want:    true,
		},
		{
			name:    "multiple wrong or'ed scopes",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:another,cluster:other"}}},
			cluster: logicalcluster.Name("this"),
			want:    false,
		},
		{
			name:    "multiple or'ed scopes that contradict eachother",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:this,cluster:other", "cluster:another,cluster:jungle"}}},
			cluster: logicalcluster.Name("this"),
			want:    false,
		},
		{
			name:    "or'ed empty scope",
			info:    user.DefaultInfo{Extra: map[string][]string{"authentication.kcp.io/scopes": {",,cluster:this"}}},
			cluster: logicalcluster.Name("this"),
			want:    true,
		},
		{
			name:    "serviceaccount from other cluster",
			info:    user.DefaultInfo{Name: "system:serviceaccount:default:foo", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"anotherws"}}},
			cluster: logicalcluster.Name("this"),
			want:    false,
		},
		{
			name:    "serviceaccount from same cluster",
			info:    user.DefaultInfo{Name: "system:serviceaccount:default:foo", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"this"}}},
			cluster: logicalcluster.Name("this"),
			want:    true,
		},
		{
			name:    "serviceaccount without a cluster",
			info:    user.DefaultInfo{Name: "system:serviceaccount:default:foo"},
			cluster: logicalcluster.Name("this"),
			// an unqualified service account is considered local: think of some
			// local SubjectAccessReview specifying a service account without the
			// cluster scope.
			want: true,
		},
		{
			name: "scoped service account",
			info: user.DefaultInfo{Name: "system:serviceaccount:default:foo", Extra: map[string][]string{
				"authentication.kubernetes.io/cluster-name": {"this"},
				"authentication.kcp.io/scopes":              {"cluster:this"},
			}},
			cluster: logicalcluster.Name("this"),
			want:    true,
		},
		{
			name: "scoped foreign service account",
			info: user.DefaultInfo{Name: "system:serviceaccount:default:foo", Extra: map[string][]string{
				"authentication.kubernetes.io/cluster-name": {"another"},
				"authentication.kcp.io/scopes":              {"cluster:this"},
			}},
			cluster: logicalcluster.Name("this"),
			want:    false,
		},
		{
			name: "scoped service account to another clusters",
			info: user.DefaultInfo{Name: "system:serviceaccount:default:foo", Extra: map[string][]string{
				"authentication.kubernetes.io/cluster-name": {"this"},
				"authentication.kcp.io/scopes":              {"cluster:another"},
			}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInScope(&tt.info, tt.cluster); got != tt.want {
				t.Errorf("IsInScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppliesToUserWithScopes(t *testing.T) {
	tests := []struct {
		name string
		user user.Info
		sub  rbacv1.Subject
		want bool
	}{
		{
			name: "simple matching user",
			user: &user.DefaultInfo{Name: "user-a"},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "simple non-matching user",
			user: &user.DefaultInfo{Name: "user-a"},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-b"},
			want: false,
		},
		{
			name: "foreign service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"other"}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: false,
		},
		{
			name: "local service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"this"}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: true,
		},
		{
			name: "non-cluster-aware service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa"},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: true,
		},
		{
			name: "in-scope scoped user",
			user: &user.DefaultInfo{Name: "user-a", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:this"}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "out-of-scope user",
			user: &user.DefaultInfo{Name: "user-a", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:other"}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: false,
		},
		{
			name: "out-of-scope anonymous user",
			user: &user.DefaultInfo{Name: "user-a", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:other"}}},
			sub:  rbacv1.Subject{Kind: "Group", Name: "system:authenticated"},
			want: false,
		},
		{
			name: "out-of-scope anonymous user",
			user: &user.DefaultInfo{Name: "user-a", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:other"}}},
			sub:  rbacv1.Subject{Kind: "Group", Name: "system:unauthenticated"},
			want: true,
		},
		{
			name: "out-of-scope authenticated user",
			user: &user.DefaultInfo{Name: "user-a", Groups: []string{user.AllAuthenticated}, Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:other"}}},
			sub:  rbacv1.Subject{Kind: "Group", Name: "system:authenticated"},
			want: true,
		},
		{
			name: "in-scope service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:this"}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: true,
		},
		{
			name: "out-of-scope service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:other"}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := request.WithCluster(context.Background(), request.Cluster{Name: "this"})
			if got := appliesToUserWithScopes(ctx, tt.user, tt.sub, "ns"); got != tt.want {
				t.Errorf("appliesToUserWithScopes(%#v, %#v) = %v, want %v", tt.user, tt.sub, got, tt.want)
			}
		})
	}
}

func TestAppliesToUserWithWarrantsAndScopes(t *testing.T) {
	tests := []struct {
		name string
		user user.Info
		sub  rbacv1.Subject
		want bool
	}{
		{
			name: "simple matching user without warrants",
			user: &user.DefaultInfo{Name: "user-a"},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "simple non-matching user without warrants",
			user: &user.DefaultInfo{Name: "user-a"},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-b"},
			want: false,
		},
		{
			name: "simple matching user with warrants",
			user: &user.DefaultInfo{Name: "user-a", Extra: map[string][]string{WarrantExtraKey: {`{"user":"user-b"}`}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "simple non-matching user with matching warrants",
			user: &user.DefaultInfo{Name: "user-b", Extra: map[string][]string{WarrantExtraKey: {`{"user":"user-a"}`}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "simple non-matching user with non-matching warrants",
			user: &user.DefaultInfo{Name: "user-b", Extra: map[string][]string{WarrantExtraKey: {`{"user":"user-b"}`}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: false,
		},
		{
			name: "simple non-matching user with multiple warrants",
			user: &user.DefaultInfo{Name: "user-b", Extra: map[string][]string{WarrantExtraKey: {`{"user":"user-b"}`, `{"user":"user-a"}`, `{"user":"user-c"}`}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "simple non-matching user with nested warrants",
			user: &user.DefaultInfo{Name: "user-b", Extra: map[string][]string{WarrantExtraKey: {`{"user":"user-b","extra":{"authorization.kcp.io/warrant":["{\"user\":\"user-a\"}"]}}`}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "foreign service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"other"}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: false,
		},
		{
			name: "local service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"this"}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: true,
		},
		{
			name: "foreign service account with local warrant",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"other"}, WarrantExtraKey: {`{"user":"system:serviceaccount:ns:sa","extra":{"authentication.kubernetes.io/cluster-name":["this"]}}`}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: true,
		},
		{
			name: "foreign service account with foreign warrant",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"other"}, WarrantExtraKey: {`{"user":"system:serviceaccount:ns:sa","extra":{"authentication.kubernetes.io/cluster-name":["other"]}}`}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: false,
		},
		{
			name: "non-cluster-aware service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa"},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: true,
		},
		{
			name: "non-cluster-aware service account as warrant",
			user: &user.DefaultInfo{Name: "user-b", Extra: map[string][]string{WarrantExtraKey: {`{"user":"system:serviceaccount:ns:sa"}`}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: false,
		},
		{
			name: "in-scope scoped user",
			user: &user.DefaultInfo{Name: "user-a", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:this"}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "out-of-scope user",
			user: &user.DefaultInfo{Name: "user-a", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:other"}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: false,
		},
		{
			name: "out-of-scope user with warrent",
			user: &user.DefaultInfo{Name: "user-a", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:other"}, WarrantExtraKey: {`{"user":"user-a"}`}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "out-of-scope warrant",
			user: &user.DefaultInfo{Name: "user-b", Extra: map[string][]string{WarrantExtraKey: {`{"user":"user-a","extra":{"authentication.kcp.io/scopes":["cluster:other"]}}`}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: false,
		},
		{
			name: "in-scope warrant",
			user: &user.DefaultInfo{Name: "user-b", Extra: map[string][]string{WarrantExtraKey: {`{"user":"user-a","extra":{"authentication.kcp.io/scopes":["cluster:this"]}}`}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "user-a"},
			want: true,
		},
		{
			name: "in-scope service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:this"}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: true,
		},
		{
			name: "out-of-scope service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:other"}}},
			sub:  rbacv1.Subject{Kind: "ServiceAccount", Namespace: "ns", Name: "sa"},
			want: false,
		},
		{
			name: "service account with cluster as global kcp service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kubernetes.io/cluster-name": {"this"}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "system:kcp:serviceaccount:this:ns:sa"},
			want: true,
		},
		{
			name: "service account with scope as global kcp service account",
			user: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Extra: map[string][]string{"authentication.kcp.io/scopes": {"cluster:this"}}},
			sub:  rbacv1.Subject{Kind: "User", Name: "system:kcp:serviceaccount:this:ns:sa"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := request.WithCluster(context.Background(), request.Cluster{Name: "this"})
			if got := appliesToUserWithScopedAndWarrants(ctx, tt.user, tt.sub, "ns"); got != tt.want {
				t.Errorf("withWarrants(withScopes(base)) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveGroups(t *testing.T) {
	tests := map[string]struct {
		u    user.Info
		want sets.Set[string]
	}{
		"empty user": {
			u:    &user.DefaultInfo{},
			want: sets.New[string](),
		},
		"authenticated user": {
			u:    &user.DefaultInfo{Name: user.Anonymous, Groups: []string{user.AllAuthenticated}},
			want: sets.New(user.AllAuthenticated),
		},
		"multiple groups": {
			u:    &user.DefaultInfo{Name: user.Anonymous, Groups: []string{"a", "b"}},
			want: sets.New("a", "b"),
		},
		"out of scope user": {
			u: &user.DefaultInfo{Name: user.Anonymous, Groups: []string{"a", "b"}, Extra: map[string][]string{
				ScopeExtraKey: {"cluster:other"},
			}},
			want: sets.New[string](),
		},
		"out of scope authenticated user": {
			u: &user.DefaultInfo{Name: user.Anonymous, Groups: []string{user.AllAuthenticated, "a", "b"}, Extra: map[string][]string{
				ScopeExtraKey: {"cluster:other"},
			}},
			want: sets.New(user.AllAuthenticated),
		},
		"user with warrant": {
			u: &user.DefaultInfo{Name: user.Anonymous, Groups: []string{"a", "b"}, Extra: map[string][]string{
				WarrantExtraKey: {`{"user":"warrant","groups":["c","d"]}`},
			}},
			want: sets.New("a", "b", "c", "d"),
		},
		"user with warrant out of scope": {
			u: &user.DefaultInfo{Name: user.Anonymous, Groups: []string{"a", "b"}, Extra: map[string][]string{
				WarrantExtraKey: {`{"user":"warrant","groups":["c","d"],"extra":{"authentication.kcp.io/scopes":["cluster:other"]}}`},
			}},
			want: sets.New("a", "b"),
		},
		"nested warrants": {
			u: &user.DefaultInfo{Name: user.Anonymous, Groups: []string{"a", "b"}, Extra: map[string][]string{
				WarrantExtraKey: {`{"user":"warrant","groups":["c","d"],"extra":{"authorization.kcp.io/warrant":["{\"user\":\"warrant2\",\"groups\":[\"e\",\"f\"]}"]}}`},
			}},
			want: sets.New("a", "b", "c", "d", "e", "f"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := request.WithCluster(context.Background(), request.Cluster{Name: "root:ws"})
			got := EffectiveGroups(ctx, tt.u)
			if diff := cmp.Diff(sets.List(tt.want), sets.List(got)); diff != "" {
				t.Errorf("effectiveGroups() +got -want\n%s", diff)
			}
		})
	}
}

func TestPrefixUser(t *testing.T) {
	tests := map[string]struct {
		u      user.Info
		prefix string
		want   user.Info
	}{
		"user with groups": {
			u:      &user.DefaultInfo{Name: "user", Groups: []string{"a", "b"}},
			prefix: "prefix:",
			want:   &user.DefaultInfo{Name: "prefix:user", Groups: []string{"prefix:a", "prefix:b"}},
		},
		"user with warrant": {
			u: &user.DefaultInfo{Name: "user", Extra: map[string][]string{
				WarrantExtraKey: {`{"user":"warrant","groups":["c","d"]}`},
			}},
			prefix: "prefix:",
			want: &user.DefaultInfo{Name: "prefix:user", Extra: map[string][]string{
				WarrantExtraKey: {`{"user":"prefix:warrant","groups":["prefix:c","prefix:d"]}`},
			}},
		},
		"service account without cluster": {
			u:      &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Groups: []string{"system:serviceaccounts"}},
			prefix: "prefix:",
			want:   &user.DefaultInfo{Name: "prefix:system:anonymous", Groups: []string{"prefix:system:unauthenticated"}},
		},
		"service account without cluster but authenticated": {
			u:      &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Groups: []string{"system:serviceaccounts", user.AllAuthenticated}},
			prefix: "prefix:",
			want:   &user.DefaultInfo{Name: "prefix:system:anonymous", Groups: []string{"prefix:system:authenticated"}},
		},
		"service account with cluster": {
			u: &user.DefaultInfo{Name: "system:serviceaccount:ns:sa", Groups: []string{"system:serviceaccounts"}, Extra: map[string][]string{
				authserviceaccount.ClusterNameKey: {"cluster"},
			}},
			prefix: "prefix:",
			want: &user.DefaultInfo{Name: "prefix:system:kcp:serviceaccount:cluster:ns:sa", Groups: []string{"prefix:system:serviceaccounts"}, Extra: map[string][]string{
				authserviceaccount.ClusterNameKey: {"cluster"},
			}},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := PrefixUser(tt.u, tt.prefix)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("prefixUser() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
