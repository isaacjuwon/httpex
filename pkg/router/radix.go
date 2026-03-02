package router

import (
	"github.com/isaacjuwon/httpex/pkg/core"
	"github.com/isaacjuwon/httpex/pkg/radix"
)

// RadixAdapter implements the core.Router interface using a radix tree.
type RadixAdapter struct {
	tree *radix.Tree[core.Handler]
}

// NewRadixAdapter creates a new RadixAdapter.
func NewRadixAdapter() core.Router {
	return &RadixAdapter{tree: radix.New[core.Handler]()}
}

// Add registers a handler for the given method and path pattern.
func (r *RadixAdapter) Add(method, path string, handler core.Handler) {
	r.tree.Add(method, path, handler)
}

// Find looks up a handler by method and path.
func (r *RadixAdapter) Find(method, path string) (core.Handler, core.Params, bool) {
	h, ps, ok := r.tree.Find(method, path)
	if !ok {
		return nil, nil, false
	}
	// Convert radix.Params to core.Params
	coreParams := make(core.Params, len(ps))
	for i, p := range ps {
		coreParams[i] = core.Param{Key: p.Key, Value: p.Value}
	}
	return h, coreParams, true
}

// Has reports whether any route exists for the path (any method).
func (r *RadixAdapter) Has(path string) bool {
	return r.tree.Has(path)
}
