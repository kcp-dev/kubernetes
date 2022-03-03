package integration

const (
	// ReasonCRDHandler describes tests where we will hack the CRD handler to hard-code "admin" ...
	ReasonCRDHandler = "the custom resource handler in k8s does not have a lister that knows how to be cluster-aware, so it cannot fetch CRDs in logical clusters"

	// ReasonWatchCache describes tests that will not affect us, we will likely be caching at a different layer anyway.
	ReasonWatchCache = "no watch cache is supported with CRDB"

	// ReasonRawClient describes tests that *can* be refactored, but are low-value tests for now.
	ReasonRawClient = "a refactor is required to allow raw CRDB client usage in this test"

	// ReasonEtcdSpecific describes tests that are specific to etcd
	ReasonEtcdSpecific = "specific to an etcd backend"

	// ReasonOpenAPI describes tests that are specific to OpenAPI features we disable
	ReasonOpenAPI = "OpenAPI is disabled" // TODO: is this true?
)