package radix

import (
	"slices"
	"strings"
)

// Param represents a matched path parameter.
type Param struct {
	Key   string
	Value string
}

// Params is a list of matched parameters from a route.
type Params []Param

// Get returns the value of the named parameter, or an empty string.
func (ps Params) Get(name string) string {
	if i := slices.IndexFunc(ps, func(p Param) bool { return p.Key == name }); i >= 0 {
		return ps[i].Value
	}
	return ""
}

type nodeType byte

const (
	static   nodeType = iota // /users
	param                    // :id
	catchAll                 // *filepath
)

type node[T any] struct {
	prefix   string
	children []*node[T]
	ntype    nodeType
	paramKey string
	handlers map[string]T // HTTP method → handler (strong generic type)
}

// Tree is a radix tree for route matching.
type Tree[T any] struct {
	root *node[T]
}

// New creates a new, empty [Tree].
func New[T any]() *Tree[T] {
	return &Tree[T]{
		root: &node[T]{},
	}
}

// Add registers a handler for the given method and path pattern.
func (t *Tree[T]) Add(method, path string, handler T) {
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	t.addRoute(t.root, method, path, handler)
}

func (t *Tree[T]) addRoute(n *node[T], method, path string, handler T) {
	if path == "" {
		if n.handlers == nil {
			n.handlers = make(map[string]T)
		}
		n.handlers[method] = handler
		return
	}

	if path[0] == ':' {
		t.addParamChild(n, method, path, handler)
		return
	}
	if path[0] == '*' {
		t.addCatchAllChild(n, method, path, handler)
		return
	}

	for _, child := range n.children {
		if child.ntype != static {
			continue
		}

		commonLen := longestCommonPrefix(path, child.prefix)
		if commonLen == 0 {
			continue
		}

		if commonLen < len(child.prefix) {
			splitChild := &node[T]{
				prefix:   child.prefix[commonLen:],
				children: child.children,
				ntype:    child.ntype,
				paramKey: child.paramKey,
				handlers: child.handlers,
			}

			child.prefix = child.prefix[:commonLen]
			child.children = []*node[T]{splitChild}
			child.handlers = nil
			child.paramKey = ""
		}

		t.addRoute(child, method, path[commonLen:], handler)
		return
	}

	newChild := &node[T]{
		prefix: path,
		ntype:  static,
	}

	if idx := indexOfParamOrWildcard(path); idx > 0 {
		newChild.prefix = path[:idx]
		n.children = append(n.children, newChild)
		t.addRoute(newChild, method, path[idx:], handler)
		return
	}

	newChild.handlers = map[string]T{method: handler}
	n.children = append(n.children, newChild)
}

func (t *Tree[T]) addParamChild(n *node[T], method, path string, handler T) {
	end := strings.IndexByte(path, '/')
	paramName := path[1:]
	rest := ""
	if end > 0 {
		paramName = path[1:end]
		rest = path[end:]
	}

	if i := slices.IndexFunc(n.children, func(child *node[T]) bool {
		return child.ntype == param && child.paramKey == paramName
	}); i >= 0 {
		t.addRoute(n.children[i], method, rest, handler)
		return
	}

	child := &node[T]{
		ntype:    param,
		paramKey: paramName,
	}
	n.children = append(n.children, child)
	t.addRoute(child, method, rest, handler)
}

func (t *Tree[T]) addCatchAllChild(n *node[T], method, path string, handler T) {
	paramName := path[1:]
	if paramName == "" {
		paramName = "*"
	}

	child := &node[T]{
		ntype:    catchAll,
		paramKey: paramName,
		handlers: map[string]T{method: handler},
	}
	n.children = append(n.children, child)
}

// Find looks up a handler for the given method and path.
func (t *Tree[T]) Find(method, path string) (T, Params, bool) {
	if path == "" {
		path = "/"
	}
	var params Params
	handler, ps, ok := t.find(t.root, method, path, params)
	return handler, ps, ok
}

func (t *Tree[T]) find(n *node[T], method, path string, params Params) (T, Params, bool) {
	if path == "" {
		if n.handlers != nil {
			if h, ok := n.handlers[method]; ok {
				return h, params, true
			}
		}
		var zero T
		return zero, params, false
	}

	for _, child := range n.children {
		if child.ntype != static {
			continue
		}
		if !strings.HasPrefix(path, child.prefix) {
			continue
		}
		remaining := path[len(child.prefix):]
		if h, ps, ok := t.find(child, method, remaining, params); ok {
			return h, ps, true
		}
	}

	for _, child := range n.children {
		if child.ntype != param {
			continue
		}
		end := strings.IndexByte(path, '/')
		val := path
		remaining := ""
		if end > 0 {
			val = path[:end]
			remaining = path[end:]
		}
		if val == "" {
			continue
		}
		ps := append(params, Param{Key: child.paramKey, Value: val})
		if h, ps2, ok := t.find(child, method, remaining, ps); ok {
			return h, ps2, true
		}
	}

	for _, child := range n.children {
		if child.ntype != catchAll {
			continue
		}
		if child.handlers != nil {
			if h, ok := child.handlers[method]; ok {
				ps := append(params, Param{Key: child.paramKey, Value: path})
				return h, ps, true
			}
		}
	}

	var zero T
	return zero, params, false
}

// Has reports whether any route exists for the given path string.
func (t *Tree[T]) Has(path string) bool {
	if path == "" {
		path = "/"
	}
	return t.has(t.root, path)
}

func (t *Tree[T]) has(n *node[T], path string) bool {
	if path == "" {
		return n.handlers != nil && len(n.handlers) > 0
	}
	for _, child := range n.children {
		switch child.ntype {
		case static:
			if strings.HasPrefix(path, child.prefix) {
				if t.has(child, path[len(child.prefix):]) {
					return true
				}
			}
		case param:
			end := strings.IndexByte(path, '/')
			if end < 0 {
				end = len(path)
			}
			if end > 0 {
				if t.has(child, path[end:]) {
					return true
				}
			}
		case catchAll:
			return child.handlers != nil && len(child.handlers) > 0
		}
	}
	return false
}

func longestCommonPrefix(a, b string) int {
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i := 0; i < max; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return max
}

func indexOfParamOrWildcard(path string) int {
	return strings.IndexAny(path, ":*")
}
