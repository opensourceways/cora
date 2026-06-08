package spec

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func newSpec(paths map[string]*openapi3.PathItem) *openapi3.T {
	p := openapi3.NewPaths()
	for path, item := range paths {
		p.Set(path, item)
	}
	return &openapi3.T{Info: &openapi3.Info{Title: "Test"}, Paths: p}
}

func newOp(tags []string) *openapi3.Operation {
	op := openapi3.NewOperation()
	op.Tags = tags
	return op
}

func TestNormalizer_TagRemap(t *testing.T) {
	spec := newSpec(map[string]*openapi3.PathItem{
		"/issues": {Get: newOp([]string{"Pull Requests"})},
	})
	n := NewNormalizer(NormalizeRule{
		TagRemap: map[string]string{"Pull Requests": "Pulls"},
	})
	n.Normalize(spec)

	op := spec.Paths.Value("/issues").Get
	if op.Tags[0] != "Pulls" {
		t.Errorf("TagRemap: got %q, want %q", op.Tags[0], "Pulls")
	}
}

func TestNormalizer_ResourceAlias(t *testing.T) {
	spec := newSpec(map[string]*openapi3.PathItem{
		"/repos": {Get: newOp([]string{"Repositories"})},
	})
	n := NewNormalizer(NormalizeRule{
		ResourceAlias: map[string]string{"repositories": "repos"},
	})
	n.Normalize(spec)

	op := spec.Paths.Value("/repos").Get
	if op.Tags[0] != "repos" {
		t.Errorf("ResourceAlias: got %q, want %q", op.Tags[0], "repos")
	}
}

func TestNormalizer_TagReassign(t *testing.T) {
	spec := newSpec(map[string]*openapi3.PathItem{
		"/api/v5/repos/{o}/{r}/issues": {Get: newOp([]string{"Users"})},
		"/api/v5/repos/{o}/{r}/pulls":  {Get: newOp([]string{"Users"})},
	})
	n := NewNormalizer(NormalizeRule{
		TagReassign: []TagReassignRule{
			{PathPrefix: "/api/v5/repos/{o}/{r}/pulls", Tag: "Pulls"},
			{PathPrefix: "/api/v5/repos/{o}/{r}/issues", Tag: "Issues"},
		},
	})
	n.Normalize(spec)

	isTag := spec.Paths.Value("/api/v5/repos/{o}/{r}/issues").Get.Tags[0]
	prTag := spec.Paths.Value("/api/v5/repos/{o}/{r}/pulls").Get.Tags[0]

	if isTag != "Issues" {
		t.Errorf("issues tag: got %q, want %q", isTag, "Issues")
	}
	if prTag != "Pulls" {
		t.Errorf("pulls tag: got %q, want %q", prTag, "Pulls")
	}
}

func TestNormalizer_AddMissingTags(t *testing.T) {
	op := openapi3.NewOperation()
	op.Tags = nil // no tags
	op.OperationID = "createPadUsingGET"

	spec := newSpec(map[string]*openapi3.PathItem{
		"/createPad": {Get: op},
	})
	n := NewNormalizer(NormalizeRule{
		AddMissingTags: []MissingTagRule{
			{PathPrefix: "/createPad", Tag: "pad"},
		},
	})
	n.Normalize(spec)

	result := spec.Paths.Value("/createPad").Get
	if len(result.Tags) == 0 || result.Tags[0] != "pad" {
		t.Errorf("AddMissingTags: got %v, want [pad]", result.Tags)
	}
}

func TestNormalizer_NoRules_Noop(t *testing.T) {
	spec := newSpec(map[string]*openapi3.PathItem{
		"/issues": {Get: newOp([]string{"Issues"})},
	})
	n := NewNormalizer(NormalizeRule{})
	n.Normalize(spec)

	op := spec.Paths.Value("/issues").Get
	if op.Tags[0] != "Issues" {
		t.Errorf("no-op normalizer changed tags: got %q", op.Tags[0])
	}
}

func TestNormalizer_NilSpec(t *testing.T) {
	n := NewNormalizer(NormalizeRule{TagRemap: map[string]string{"A": "B"}})
	// must not panic
	n.Normalize(nil)
	n.Normalize(&openapi3.T{})
}
