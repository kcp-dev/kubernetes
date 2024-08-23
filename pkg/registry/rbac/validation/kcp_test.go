package validation

import (
	"context"
	"testing"

	"github.com/kcp-dev/logicalcluster/v3"
	rbacv1 "k8s.io/api/rbac/v1"
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
