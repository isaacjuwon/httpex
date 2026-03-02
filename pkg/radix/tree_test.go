package radix

import (
	"testing"
)

func TestRadixTreeStatic(t *testing.T) {
	tree := New[string]()
	tree.Add("GET", "/users", "get_users")
	tree.Add("POST", "/users", "post_users")

	h, params, ok := tree.Find("GET", "/users")
	if !ok || h != "get_users" || len(params) != 0 {
		t.Errorf("expected get_users, got %v with params %v", h, params)
	}

	h, _, ok = tree.Find("POST", "/users")
	if !ok || h != "post_users" {
		t.Errorf("expected post_users, got %v", h)
	}

	_, _, ok = tree.Find("PUT", "/users")
	if ok {
		t.Errorf("expected false for missing method")
	}

	if !tree.Has("/users") {
		t.Errorf("expected Has(/users) to be true")
	}
}

func TestRadixTreeParams(t *testing.T) {
	tree := New[string]()
	tree.Add("GET", "/users/:id", "get_user")
	tree.Add("GET", "/users/:id/posts/:post_id", "get_user_post")

	h, params, ok := tree.Find("GET", "/users/123")
	if !ok || h != "get_user" {
		t.Errorf("expected get_user, got %v", h)
	}
	if params.Get("id") != "123" {
		t.Errorf("expected id=123, got %s", params.Get("id"))
	}

	h, params, ok = tree.Find("GET", "/users/123/posts/456")
	if !ok || h != "get_user_post" {
		t.Errorf("expected get_user_post, got %v", h)
	}
	if params.Get("id") != "123" || params.Get("post_id") != "456" {
		t.Errorf("expected id=123 and post_id=456, got %v", params)
	}
}

func TestRadixTreeCatchAll(t *testing.T) {
	tree := New[string]()
	tree.Add("GET", "/static/*filepath", "static_files")

	h, params, ok := tree.Find("GET", "/static/css/main.css")
	if !ok || h != "static_files" {
		t.Errorf("expected static_files, got %v", h)
	}
	if params.Get("filepath") != "css/main.css" {
		t.Errorf("expected filepath=css/main.css, got %s", params.Get("filepath"))
	}
}
