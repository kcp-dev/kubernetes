// This is a generated file. Do not edit directly.

module k8s.io/sample-controller

go 1.16

require (
	github.com/kcp-dev/logicalcluster/v2 v2.0.0-alpha.1
	k8s.io/api v0.24.3
	k8s.io/apimachinery v0.24.3
	k8s.io/client-go v0.24.3
	k8s.io/code-generator v0.0.0
	k8s.io/klog/v2 v2.60.1
)

replace (
	github.com/stretchr/testify => github.com/stretchr/testify v1.7.0
	k8s.io/api => ../api
	k8s.io/apimachinery => ../apimachinery
	k8s.io/client-go => ../client-go
	k8s.io/code-generator => ../code-generator
	k8s.io/sample-controller => ../sample-controller
)
