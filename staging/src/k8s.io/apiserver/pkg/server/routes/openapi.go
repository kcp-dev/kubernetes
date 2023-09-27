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

package routes

import (
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"

	"github.com/kcp-dev/logicalcluster/v3"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server/mux"
	builder2 "k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/builder3"
	"k8s.io/kube-openapi/pkg/cached"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/common/restfuladapter"
	"k8s.io/kube-openapi/pkg/handler"
	"k8s.io/kube-openapi/pkg/handler3"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// OpenAPI installs spec endpoints for each web service.
type OpenAPI struct {
	Config   *common.Config
	V3Config *common.OpenAPIV3Config
}

// OpenAPIServiceProvider is a hacky way to
// replace a single OpenAPIService by a provider which will
// provide an distinct openAPIService per logical cluster.
// This is required to implement CRD tenancy and have the openAPI
// models be conistent with the current logical cluster.
//
// However this is just a first step, since a better way
// would be to completly avoid the need of registering a OpenAPIService
// for each logical cluster. See the addition comments below.
type OpenAPIServiceProvider interface {
	ForCluster(clusterName logicalcluster.Name) *handler.OpenAPIService
	AddCuster(clusterName logicalcluster.Name)
	RemoveCuster(clusterName logicalcluster.Name)
	UpdateSpecLazy(swagger cached.Value[*spec.Swagger])
}

type clusterAwarePathHandler struct {
	clusterName          logicalcluster.Name
	addHandlerForCluster func(clusterName logicalcluster.Name, handler http.Handler)
}

func (c *clusterAwarePathHandler) Handle(path string, handler http.Handler) {
	c.addHandlerForCluster(c.clusterName, handler)
}

// HACK: This is the implementation of OpenAPIServiceProvider
// that allows supporting several logical clusters for CRD tenancy.
//
// However this should be conisdered a temporary step, to cope with the
// current design of OpenAPI publishing. But having to register every logical
// cluster creates more cost on creating logical clusters.
// Instead, we'd expect us to slowly refactor the openapi generation code so
// that it can be used dynamically, and time limited or size limited openapi caches
// would be used to serve the calculated version.
// Finally a development princple for the logical cluster prototype would be
// - don't do static registration of logical clusters
// - do lazy instantiation wherever possible so that starting a new logical cluster remains as cheap as possible
type openAPIServiceProvider struct {
	staticSpec                   *spec.Swagger
	defaultOpenAPIServiceHandler http.Handler
	defaultOpenAPIService        *handler.OpenAPIService
	openAPIServices              map[logicalcluster.Name]*handler.OpenAPIService
	handlers                     map[logicalcluster.Name]http.Handler
	path                         string
	mux                          *mux.PathRecorderMux
}

var _ OpenAPIServiceProvider = (*openAPIServiceProvider)(nil)

func (p *openAPIServiceProvider) ForCluster(clusterName logicalcluster.Name) *handler.OpenAPIService {
	return p.openAPIServices[clusterName]
}

func (p *openAPIServiceProvider) AddCuster(clusterName logicalcluster.Name) {
	if _, found := p.openAPIServices[clusterName]; !found {
		openAPIVersionedService := handler.NewOpenAPIService(p.staticSpec)

		openAPIVersionedService.RegisterOpenAPIVersionedService(p.path, &clusterAwarePathHandler{
			clusterName: clusterName,
			addHandlerForCluster: func(clusterName logicalcluster.Name, handler http.Handler) {
				p.handlers[clusterName] = handler
			},
		})

		p.openAPIServices[clusterName] = openAPIVersionedService
	}
}

func (p *openAPIServiceProvider) RemoveCuster(clusterName logicalcluster.Name) {
	delete(p.openAPIServices, clusterName)
	delete(p.handlers, clusterName)
}

func (p *openAPIServiceProvider) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	cluster := genericapirequest.ClusterFrom(req.Context())
	if cluster == nil {
		p.defaultOpenAPIServiceHandler.ServeHTTP(resp, req)
		return
	}
	handler, found := p.handlers[cluster.Name]
	if !found {
		resp.WriteHeader(404)
		return
	}
	handler.ServeHTTP(resp, req)
}

func (o *openAPIServiceProvider) UpdateSpecLazy(openapiSpec cached.Value[*spec.Swagger]) {
	o.defaultOpenAPIService.UpdateSpecLazy(openapiSpec)
}

func (p *openAPIServiceProvider) Register() {
	defaultOpenAPIService := handler.NewOpenAPIService(p.staticSpec)

	defaultOpenAPIService.RegisterOpenAPIVersionedService(p.path, &clusterAwarePathHandler{
		addHandlerForCluster: func(clusterName logicalcluster.Name, handler http.Handler) {
			p.defaultOpenAPIServiceHandler = handler
		},
	})

	p.defaultOpenAPIService = defaultOpenAPIService
	p.mux.Handle(p.path, p)
}

// Install adds the SwaggerUI webservice to the given mux.
func (oa OpenAPI) InstallV2(c *restful.Container, mux *mux.PathRecorderMux) (OpenAPIServiceProvider, *spec.Swagger) {
	spec, err := builder2.BuildOpenAPISpec(c.RegisteredWebServices(), oa.Config)
	if err != nil {
		klog.Fatalf("Failed to build open api spec for root: %v", err)
	}
	spec.Definitions = handler.PruneDefaults(spec.Definitions)

	provider := &openAPIServiceProvider{
		mux:             mux,
		staticSpec:      spec,
		openAPIServices: map[logicalcluster.Name]*handler.OpenAPIService{},
		handlers:        map[logicalcluster.Name]http.Handler{},
		path:            "/openapi/v2",
	}

	provider.Register()

	return provider, spec
}

// InstallV3 adds the static group/versions defined in the RegisteredWebServices to the OpenAPI v3 spec.
// This only covers built-in resources served via go-restful; CRDs and aggregated APIs publish
// their OpenAPI v3 specs through separate code paths.
func (oa OpenAPI) InstallV3(c *restful.Container, mux *mux.PathRecorderMux) *handler3.OpenAPIService {
	openAPIVersionedService := handler3.NewOpenAPIService()
	err := openAPIVersionedService.RegisterOpenAPIV3VersionedService("/openapi/v3", mux)
	if err != nil {
		klog.Fatalf("Failed to register versioned open api spec for root: %v", err)
	}

	grouped := make(map[string][]*restful.WebService)

	for _, t := range c.RegisteredWebServices() {
		// Strip the "/" prefix from the name
		gvName := t.RootPath()[1:]
		grouped[gvName] = []*restful.WebService{t}
	}

	for gv, ws := range grouped {
		spec, err := builder3.BuildOpenAPISpecFromRoutes(restfuladapter.AdaptWebServices(ws), oa.V3Config)
		if err != nil {
			klog.Errorf("Failed to build OpenAPI v3 for group %s, %q", gv, err)
			continue
		}
		if group, version, ok := groupVersionFromPath(gv); ok {
			filterScopedGVKs(spec, group, version)
		}
		openAPIVersionedService.UpdateGroupVersion(gv, spec)
	}
	return openAPIVersionedService
}

// groupVersionFromPath extracts the API group and version from a root path like "apis/apps/v1" or "api/v1".
func groupVersionFromPath(path string) (group, version string, ok bool) {
	// "api/v1" → ("", "v1", true)
	// "apis/apps/v1" → ("apps", "v1", true)
	// "apis/networking.k8s.io/v1" → ("networking.k8s.io", "v1", true)
	parts := strings.SplitN(path, "/", 4)
	switch {
	case len(parts) < 2:
		return "", "", false
	case parts[0] == "api" && len(parts) == 2:
		return "", parts[1], true
	case parts[0] == "apis" && len(parts) == 3:
		return parts[1], parts[2], true
	default:
		return "", "", false
	}
}

// crossRegisteredKinds lists the kinds that AddToGroupVersion (in
// k8s.io/apimachinery/pkg/apis/meta/v1) registers into every API group.
// Only these types get their x-kubernetes-group-version-kind list filtered
// in per-GV v3 specs.
var crossRegisteredKinds = map[string]bool{
	"WatchEvent":    true,
	"DeleteOptions": true,
}

// filterScopedGVKs narrows x-kubernetes-group-version-kind on the meta types
// that AddToGroupVersion cross-registers into every API group (WatchEvent and
// DeleteOptions). Without filtering, each per-GV v3 spec carries ~60 GVKs for
// these types. This filter keeps only the entry matching this spec's group/version
// plus the canonical core/v1 entry.
//
// Only the meta types listed in crossRegisteredKinds are filtered. Other types
// that are intentionally registered into specific groups (like autoscaling/v1
// Scale into apps/v1 and core/v1) are left unchanged.
//
// This only affects built-in resource specs generated by InstallV3 above.
// CRDs and aggregated APIs are not affected as they publish specs through separate paths.
func filterScopedGVKs(s *spec3.OpenAPI, group, version string) {
	if s == nil || s.Components == nil {
		return
	}
	for _, schema := range s.Components.Schemas {
		if schema == nil {
			continue
		}
		ext, ok := schema.Extensions["x-kubernetes-group-version-kind"]
		if !ok {
			continue
		}
		gvks, ok := ext.([]interface{})
		if !ok || len(gvks) <= 1 {
			continue
		}
		if !isCrossRegisteredKind(gvks) {
			continue
		}
		var filtered []interface{}
		for _, item := range gvks {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			g, ok := m["group"].(string)
			if !ok {
				continue
			}
			v, ok := m["version"].(string)
			if !ok {
				continue
			}
			if (g == group && v == version) || (g == "" && v == "v1") {
				filtered = append(filtered, item)
			}
		}
		if len(filtered) > 0 {
			schema.Extensions["x-kubernetes-group-version-kind"] = filtered
		} else {
			klog.Warningf("Unexpected: filtering x-kubernetes-group-version-kind for %s/%s produced no matches", group, version)
			delete(schema.Extensions, "x-kubernetes-group-version-kind")
		}
	}
}

// isCrossRegisteredKind checks whether a GVK list represents one of the
// meta types from crossRegisteredKinds by inspecting the first entry's kind.
func isCrossRegisteredKind(gvks []interface{}) bool {
	if len(gvks) == 0 {
		return false
	}
	m, ok := gvks[0].(map[string]interface{})
	if !ok {
		return false
	}
	kind, _ := m["kind"].(string)
	return crossRegisteredKinds[kind]
}
