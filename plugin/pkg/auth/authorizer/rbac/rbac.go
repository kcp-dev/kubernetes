/*
Copyright 2016 The Kubernetes Authors.

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

// Package rbac implements the authorizer.Authorizer interface using roles base access control.
package rbac

import (
	"bytes"
	"context"
	"fmt"

	"k8s.io/klog/v2"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/clusters"
	rbacv1helpers "k8s.io/kubernetes/pkg/apis/rbac/v1"
	rbacregistryvalidation "k8s.io/kubernetes/pkg/registry/rbac/validation"
)

type RequestToRuleMapper interface {
	// RulesFor returns all known PolicyRules and any errors that happened while locating those rules.
	// Any rule returned is still valid, since rules are deny by default.  If you can pass with the rules
	// supplied, you do not have to fail the request.  If you cannot, you should indicate the error along
	// with your denial.
	RulesFor(subject user.Info, namespace string) ([]rbacv1.PolicyRule, error)

	// VisitRulesFor invokes visitor() with each rule that applies to a given user in a given namespace,
	// and each error encountered resolving those rules. Rule may be nil if err is non-nil.
	// If visitor() returns false, visiting is short-circuited.
	VisitRulesFor(user user.Info, namespace string, visitor func(source fmt.Stringer, rule *rbacv1.PolicyRule, err error) bool)
}

type RBACAuthorizer struct {
	authorizationRuleResolver RequestToRuleMapper
}

// authorizingVisitor short-circuits once allowed, and collects any resolution errors encountered
type authorizingVisitor struct {
	requestAttributes authorizer.Attributes

	allowed bool
	reason  string
	errors  []error
}

type UserInfoWithCluster struct {
	Username string
	UID      string
	Groups   []string
	Extra    map[string][]string
}

func UserInfo(clusterName string, requestAttributes authorizer.Attributes) *UserInfoWithCluster {
	info := &UserInfoWithCluster{
		Username: requestAttributes.GetUser().GetName(),
		UID:      requestAttributes.GetUser().GetUID(),
		Groups:   requestAttributes.GetUser().GetGroups(),
		Extra:    map[string][]string{},
	}

	info.Extra["logical-cluster"] = []string{clusterName}

	return info
}

func (u *UserInfoWithCluster) GetName() string {
	return u.Username
}

func (u *UserInfoWithCluster) GetUID() string {
	return u.UID
}

func (u *UserInfoWithCluster) GetGroups() []string {
	return u.Groups
}

func (u *UserInfoWithCluster) GetExtra() map[string][]string {
	return u.Extra
}

func (v *authorizingVisitor) visit(source fmt.Stringer, rule *rbacv1.PolicyRule, err error) bool {
	if rule != nil && RuleAllows(v.requestAttributes, rule) {
		v.allowed = true
		v.reason = fmt.Sprintf("RBAC: allowed by %s", source.String())
		return false
	}
	if err != nil {
		v.errors = append(v.errors, err)
	}
	return true
}

func (r *RBACAuthorizer) Authorize(ctx context.Context, requestAttributes authorizer.Attributes) (authorizer.Decision, string, error) {
	clusterName, err := genericapirequest.ClusterNameFrom(ctx)
	if err != nil {
		klog.Errorf("No cluster defined in authorize action : %s", err.Error())
	}
	klog.Infof("cluster is %s", clusterName)

	ruleCheckingVisitor := &authorizingVisitor{requestAttributes: requestAttributes}

	user := UserInfo(clusterName, requestAttributes)

	r.authorizationRuleResolver.VisitRulesFor(user, requestAttributes.GetNamespace(), ruleCheckingVisitor.visit)
	if ruleCheckingVisitor.allowed {
		return authorizer.DecisionAllow, ruleCheckingVisitor.reason, nil
	}

	// Build a detailed log of the denial.
	// Make the whole block conditional so we don't do a lot of string-building we won't use.
	if klog.V(5).Enabled() {
		var operation string
		if requestAttributes.IsResourceRequest() {
			b := &bytes.Buffer{}
			b.WriteString(`"`)
			b.WriteString(requestAttributes.GetVerb())
			b.WriteString(`" resource "`)
			b.WriteString(requestAttributes.GetResource())
			if len(requestAttributes.GetAPIGroup()) > 0 {
				b.WriteString(`.`)
				b.WriteString(requestAttributes.GetAPIGroup())
			}
			if len(requestAttributes.GetSubresource()) > 0 {
				b.WriteString(`/`)
				b.WriteString(requestAttributes.GetSubresource())
			}
			b.WriteString(`"`)
			if len(requestAttributes.GetName()) > 0 {
				b.WriteString(` named "`)
				b.WriteString(requestAttributes.GetName())
				b.WriteString(`"`)
			}
			operation = b.String()
		} else {
			operation = fmt.Sprintf("%q nonResourceURL %q", requestAttributes.GetVerb(), requestAttributes.GetPath())
		}

		var scope string
		if ns := requestAttributes.GetNamespace(); len(ns) > 0 {
			scope = fmt.Sprintf("in namespace %q", ns)
		} else {
			scope = "cluster-wide"
		}

		klog.Infof("RBAC: no rules authorize user %q with groups %q to %s %s", requestAttributes.GetUser().GetName(), requestAttributes.GetUser().GetGroups(), operation, scope)
	}

	reason := ""
	if len(ruleCheckingVisitor.errors) > 0 {
		reason = fmt.Sprintf("RBAC: %v", utilerrors.NewAggregate(ruleCheckingVisitor.errors))
	}
	return authorizer.DecisionNoOpinion, reason, nil
}

func (r *RBACAuthorizer) RulesFor(user user.Info, namespace string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	var (
		resourceRules    []authorizer.ResourceRuleInfo
		nonResourceRules []authorizer.NonResourceRuleInfo
	)

	policyRules, err := r.authorizationRuleResolver.RulesFor(user, namespace)
	for _, policyRule := range policyRules {
		if len(policyRule.Resources) > 0 {
			r := authorizer.DefaultResourceRuleInfo{
				Verbs:         policyRule.Verbs,
				APIGroups:     policyRule.APIGroups,
				Resources:     policyRule.Resources,
				ResourceNames: policyRule.ResourceNames,
			}
			var resourceRule authorizer.ResourceRuleInfo = &r
			resourceRules = append(resourceRules, resourceRule)
		}
		if len(policyRule.NonResourceURLs) > 0 {
			r := authorizer.DefaultNonResourceRuleInfo{
				Verbs:           policyRule.Verbs,
				NonResourceURLs: policyRule.NonResourceURLs,
			}
			var nonResourceRule authorizer.NonResourceRuleInfo = &r
			nonResourceRules = append(nonResourceRules, nonResourceRule)
		}
	}
	return resourceRules, nonResourceRules, false, err
}

func New(roles rbacregistryvalidation.RoleGetter, roleBindings rbacregistryvalidation.RoleBindingLister, clusterRoles rbacregistryvalidation.ClusterRoleGetter, clusterRoleBindings rbacregistryvalidation.ClusterRoleBindingLister) *RBACAuthorizer {
	authorizer := &RBACAuthorizer{
		authorizationRuleResolver: rbacregistryvalidation.NewDefaultRuleResolver(
			roles, roleBindings, clusterRoles, clusterRoleBindings,
		),
	}
	return authorizer
}

func RulesAllow(requestAttributes authorizer.Attributes, rules ...rbacv1.PolicyRule) bool {
	for i := range rules {
		if RuleAllows(requestAttributes, &rules[i]) {
			return true
		}
	}

	return false
}

func RuleAllows(requestAttributes authorizer.Attributes, rule *rbacv1.PolicyRule) bool {
	if requestAttributes.IsResourceRequest() {
		combinedResource := requestAttributes.GetResource()
		if len(requestAttributes.GetSubresource()) > 0 {
			combinedResource = requestAttributes.GetResource() + "/" + requestAttributes.GetSubresource()
		}

		return rbacv1helpers.VerbMatches(rule, requestAttributes.GetVerb()) &&
			rbacv1helpers.APIGroupMatches(rule, requestAttributes.GetAPIGroup()) &&
			rbacv1helpers.ResourceMatches(rule, combinedResource, requestAttributes.GetSubresource()) &&
			rbacv1helpers.ResourceNameMatches(rule, requestAttributes.GetName())
	}

	return rbacv1helpers.VerbMatches(rule, requestAttributes.GetVerb()) &&
		rbacv1helpers.NonResourceURLMatches(rule, requestAttributes.GetPath())
}

type RoleGetter struct {
	Lister rbaclisters.RoleLister
}

func (g *RoleGetter) GetRole(cluster, namespace, name string) (*rbacv1.Role, error) {
	return g.Lister.Roles(namespace).Get(name)
}

type RoleBindingLister struct {
	Lister rbaclisters.RoleBindingLister
}

func (l *RoleBindingLister) ListRoleBindings(cluster, namespace string) ([]*rbacv1.RoleBinding, error) {
	return l.Lister.RoleBindings(namespace).List(labels.Everything())
}

type ClusterRoleGetter struct {
	Lister rbaclisters.ClusterRoleLister
}

func (g *ClusterRoleGetter) GetClusterRole(cluster, name string) (*rbacv1.ClusterRole, error) {
	klog.Infof("cluster is %s", cluster)
	key := clusters.ToClusterAwareKey(cluster, name)
	a, _ := g.Lister.Get(key)
	klog.Infof("cluster role are %v", a)
	return g.Lister.Get(key)
}

type ClusterRoleBindingLister struct {
	Lister rbaclisters.ClusterRoleBindingLister
}

func (l *ClusterRoleBindingLister) ListClusterRoleBindings(cluster string) ([]*rbacv1.ClusterRoleBinding, error) {
	clbs, err := l.Lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	filtered := []*rbacv1.ClusterRoleBinding{}
	for _, clb := range clbs {
		if clb.ClusterName == cluster {
			filtered = append(filtered, clb)
		}
	}
	return filtered, nil
}
