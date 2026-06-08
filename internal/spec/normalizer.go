package spec

import (
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// NormalizeRule declares transformations to apply to an OpenAPI spec
// before command generation. All fields are optional; zero value = no-op.
type NormalizeRule struct {
	// TagRemap renames a tag. Keys are old names, values are new names.
	// e.g. {"Pull Requests" → "Pulls"}.
	TagRemap map[string]string

	// ResourceAlias maps a kebab-cased tag to a canonical resource name.
	// Applied after TagRemap.
	// e.g. {"pull-requests" → "pulls", "repositories" → "repos"}.
	ResourceAlias map[string]string

	// TagReassign moves operations matching a path prefix to a target tag.
	// Used when spec authors tag an operation with the wrong parent resource.
	// Only the first tag (used as resource name) is reassigned.
	// e.g. {PathPrefix: "/repos/{o}/{r}/pulls", Tag: "Pulls"}.
	TagReassign []TagReassignRule

	// OpIDVerbExtract enables HTTP-method-prefix extraction for path-encoded
	// operationIds (e.g. GitCode's get_api_v5_… → "get"/"list").
	OpIDVerbExtract bool

	// AddMissingTags assigns a tag to operations that have no tags,
	// using the path to infer the appropriate tag.
	// e.g. {PathPattern: "/createPad*", Tag: "pad", Method: "GET"}.
	AddMissingTags []MissingTagRule
}

// TagReassignRule maps a path prefix to a target tag.
type TagReassignRule struct {
	PathPrefix string // e.g. "/api/v5/repos/{owner}/{repo}/pulls"
	Tag        string // target tag, e.g. "Pulls"
}

// MissingTagRule assigns a tag when an operation has none.
type MissingTagRule struct {
	PathPrefix string // empty = match any path
	Method     string // empty = match any method
	Tag        string // tag to assign
}

// Normalizer applies NormalizeRule to an OpenAPI spec.
type Normalizer struct {
	rules NormalizeRule
}

// NewNormalizer creates a Normalizer for the given rules.
func NewNormalizer(rules NormalizeRule) *Normalizer {
	return &Normalizer{rules: rules}
}

// Normalize applies all registered transformations to the spec, returning
// a mutated copy. The builder should always use the normalized spec.
func (n *Normalizer) Normalize(spec *openapi3.T) *openapi3.T {
	if spec == nil || spec.Paths == nil {
		return spec
	}

	n.applyTagRemap(spec)
	n.applyResourceAlias(spec)
	n.applyTagReassign(spec)
	n.applyAddMissingTags(spec)

	return spec
}

// ── tag remap ───────────────────────────────────────────────────────────

func (n *Normalizer) applyTagRemap(spec *openapi3.T) {
	if len(n.rules.TagRemap) == 0 {
		return
	}
	for _, pathItem := range spec.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for _, op := range pathItem.Operations() {
			if op == nil {
				continue
			}
			for i, tag := range op.Tags {
				if newTag, ok := n.rules.TagRemap[tag]; ok {
					op.Tags[i] = newTag
				}
			}
		}
	}
}

// ── resource alias ──────────────────────────────────────────────────────

func (n *Normalizer) applyResourceAlias(spec *openapi3.T) {
	if len(n.rules.ResourceAlias) == 0 {
		return
	}
	for _, pathItem := range spec.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for _, op := range pathItem.Operations() {
			if op == nil || len(op.Tags) == 0 {
				continue
			}
			kebab := strings.ToLower(strings.ReplaceAll(op.Tags[0], " ", "-"))
			if canonical, ok := n.rules.ResourceAlias[kebab]; ok {
				op.Tags[0] = canonical
			}
		}
	}
}

// ── tag reassign ────────────────────────────────────────────────────────

func (n *Normalizer) applyTagReassign(spec *openapi3.T) {
	if len(n.rules.TagReassign) == 0 {
		return
	}
	// Sort by path length descending so longer (more specific) prefixes
	// are matched before shorter ones.
	sorted := make([]TagReassignRule, len(n.rules.TagReassign))
	copy(sorted, n.rules.TagReassign)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].PathPrefix) > len(sorted[j].PathPrefix)
	})

	for path, pathItem := range spec.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for _, rule := range sorted {
			if strings.HasPrefix(path, rule.PathPrefix) {
				for _, op := range pathItem.Operations() {
					if op != nil && len(op.Tags) > 0 {
						op.Tags[0] = rule.Tag
					}
				}
				break // first (longest) match wins
			}
		}
	}
}

// ── add missing tags ────────────────────────────────────────────────────

func (n *Normalizer) applyAddMissingTags(spec *openapi3.T) {
	if len(n.rules.AddMissingTags) == 0 {
		return
	}
	for path, pathItem := range spec.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for method, op := range pathItem.Operations() {
			if op == nil || len(op.Tags) > 0 {
				continue
			}
			for _, rule := range n.rules.AddMissingTags {
				if rule.Method != "" && !strings.EqualFold(rule.Method, method) {
					continue
				}
				if rule.PathPrefix != "" && !strings.HasPrefix(path, rule.PathPrefix) {
					continue
				}
				op.Tags = []string{rule.Tag}
				break
			}
		}
	}
}
