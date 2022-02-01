package etcd3

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
)

const (
	contextKeyInitialObject = contextKey("initial")
	contextKeyFinalObject   = contextKey("final")
)

// StoredObject co-locates encoded and decoded values of an object along with metadata
// regarding our storage of the object in etcd.
type StoredObject struct {
	// Object is a pointer to the decoded object in storage
	Object runtime.Object
	// Metadata holds user-accessible data about the object
	Metadata *storage.ResponseMeta
	// Data is the raw encoded value in the storage
	Data []byte
	// Stale is true when data has been read from storage and deemed
	// stale by virtue of the encryption-at-rest settings changing between
	// that read and last write of the object. For example, if encryption
	// has been enabled or if the key has been rotated since the last write.
	Stale bool
}

// WithInitialObject stores the initial object in the provided context
func WithInitialObject(ctx context.Context, object *StoredObject) context.Context {
	return context.WithValue(ctx, contextKeyInitialObject, object)
}

// WithFinalObject stores the final object in the provided context
func WithFinalObject(ctx context.Context, object *StoredObject) context.Context {
	return context.WithValue(ctx, contextKeyFinalObject, object)
}

// InitialObjectFrom extracts the initial stored object from the context
func InitialObjectFrom(ctx context.Context) (*StoredObject, bool) {
	o, ok := ctx.Value(contextKeyInitialObject).(*StoredObject)
	return o, ok
}

// FinalObjectFrom extracts the final stored object from the context
func FinalObjectFrom(ctx context.Context) (*StoredObject, bool) {
	o, ok := ctx.Value(contextKeyFinalObject).(*StoredObject)
	return o, ok
}