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

	restful "github.com/emicklei/go-restful"
	"github.com/go-openapi/spec"
	"k8s.io/klog"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/handler"
)

// OpenAPI installs spec endpoints for each web service.
type OpenAPI struct {
	Config *common.Config
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
	ForCluster(clusterName string) *handler.OpenAPIService
	AddCuster(clusterName string)
	RemoveCuster(clusterName string)
	UpdateSpec(openapiSpec *spec.Swagger) error
}

type clusterAwarePathHandler struct {
	clusterName          string
	addHandlerForCluster func(clusterName string, handler http.Handler)
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
	openAPIServices              map[string]*handler.OpenAPIService
	handlers                     map[string]http.Handler
	path                         string
	mux                          *mux.PathRecorderMux
}

var _ OpenAPIServiceProvider = (*openAPIServiceProvider)(nil)

func (p *openAPIServiceProvider) ForCluster(clusterName string) *handler.OpenAPIService {
	return p.openAPIServices[clusterName]
}

func (p *openAPIServiceProvider) AddCuster(clusterName string) {
	if _, found := p.openAPIServices[clusterName]; !found {
		openAPIVersionedService, err := handler.RegisterOpenAPIVersionedService(p.staticSpec, p.path, &clusterAwarePathHandler{
			clusterName: clusterName,
			addHandlerForCluster: func(clusterName string, handler http.Handler) {
				p.handlers[clusterName] = handler
			},
		})
		if err != nil {
			klog.Fatalf("Failed to register versioned open api spec for root: %v", err)
		}
		p.openAPIServices[clusterName] = openAPIVersionedService
	}
}

func (p *openAPIServiceProvider) RemoveCuster(clusterName string) {
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

func (o *openAPIServiceProvider) UpdateSpec(openapiSpec *spec.Swagger) (err error) {
	return o.defaultOpenAPIService.UpdateSpec(openapiSpec)
}

func (p *openAPIServiceProvider) Register() {
	defaultOpenAPIService, err := handler.RegisterOpenAPIVersionedService(p.staticSpec, p.path, &clusterAwarePathHandler{
		clusterName: "",
		addHandlerForCluster: func(clusterName string, handler http.Handler) {
			p.defaultOpenAPIServiceHandler = handler
		},
	})
	if err != nil {
		klog.Fatalf("Failed to register versioned open api spec for root: %v", err)
	}
	p.defaultOpenAPIService = defaultOpenAPIService
	p.mux.Handle(p.path, p)
}

// Install adds the SwaggerUI webservice to the given mux.
func (oa OpenAPI) Install(c *restful.Container, mux *mux.PathRecorderMux) (OpenAPIServiceProvider, *spec.Swagger) {
	spec, err := builder.BuildOpenAPISpec(c.RegisteredWebServices(), oa.Config)
	if err != nil {
		klog.Fatalf("Failed to build open api spec for root: %v", err)
	}

	provider := &openAPIServiceProvider{
		mux:             mux,
		staticSpec:      spec,
		openAPIServices: map[string]*handler.OpenAPIService{},
		handlers:        map[string]http.Handler{},
		path:            "/openapi/v2",
	}

	provider.Register()

	return provider, spec
}
