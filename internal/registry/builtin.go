package registry

import (
	"time"

	"github.com/opensourceways/cora/assets"
	"github.com/opensourceways/cora/internal/config"
	"github.com/opensourceways/cora/internal/spec"
)

var (
	// gitcodeNormalize fixes known quality issues in the GitCode OpenAPI spec.
	gitcodeNormalize = spec.NormalizeRule{
		ResourceAlias: map[string]string{
			"repositories":  "repos",
			"organizations": "orgs",
			"pull-requests": "pulls",
		},
		TagReassign: []spec.TagReassignRule{
			{PathPrefix: "/api/v5/repos/{owner}/{repo}/pulls", Tag: "Pulls"},
			{PathPrefix: "/api/v5/repos/{owner}/{repo}", Tag: "Repositories"},
			{PathPrefix: "/api/v5/repos/{owner}/{repo}/issues", Tag: "Issues"},
			{PathPrefix: "/api/v5/repos/{owner}/{repo}/labels", Tag: "Labels"},
			{PathPrefix: "/api/v5/repos/{owner}/{repo}/branches", Tag: "Branch"},
			{PathPrefix: "/api/v5/repos/{owner}/{repo}/tags", Tag: "Tag"},
			{PathPrefix: "/api/v5/repos/{owner}/{repo}/releases", Tag: "Release"},
			{PathPrefix: "/api/v5/repos/{owner}/{repo}/milestones", Tag: "Milestone"},
		},
		OpIDVerbExtract: true,
	}

	// etherpadNormalize assigns tags to untagged operations.
	etherpadNormalize = spec.NormalizeRule{
		AddMissingTags: []spec.MissingTagRule{
			{PathPrefix: "/appendText", Tag: "pad"},
			{PathPrefix: "/copyPad", Tag: "pad"},
			{PathPrefix: "/getAttributePool", Tag: "pad"},
			{PathPrefix: "/getPadID", Tag: "pad"},
			{PathPrefix: "/getRevisionChangeset", Tag: "pad"},
			{PathPrefix: "/getSavedRevisionsCount", Tag: "pad"},
			{PathPrefix: "/getStats", Tag: "pad"},
			{PathPrefix: "/listSavedRevisions", Tag: "pad"},
			{PathPrefix: "/movePad", Tag: "pad"},
			{PathPrefix: "/restoreRevision", Tag: "pad"},
			{PathPrefix: "/saveRevision", Tag: "pad"},
		},
	}
)

const (
	etherpadName = "etherpad"
	gitcodeName  = "gitcode"
	githubName   = "github"
	jenkinsName  = "jenkins"
	eurName      = "eur"
)

// registerBuiltins adds built-in service entries to the registry and ensures
// each built-in is present in cfg.Services (so the executor can look it up).
//
// The OpenAPI spec for each built-in is embedded in the binary; no remote
// spec_url is required. However, base_url MUST be explicitly set in the user's
// config file — there are no hardcoded default URLs. If a built-in service is
// missing its base_url the executor will return a clear config error at runtime.
func registerBuiltins(r *Registry, cfg *config.Config) {
	cacheDir := cfg.SpecCache.Dir
	ttl := cfg.SpecCache.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	addBuiltin(r, cfg, builtinDef{
		name:      etherpadName,
		specData:  assets.EtherpadSpec,
		cacheDir:  cacheDir,
		ttl:       ttl,
		normalize: etherpadNormalize,
	})

	addBuiltin(r, cfg, builtinDef{
		name:      gitcodeName,
		specData:  assets.GitcodeSpec,
		cacheDir:  cacheDir,
		ttl:       ttl,
		normalize: gitcodeNormalize,
	})

	addBuiltin(r, cfg, builtinDef{
		name:     githubName,
		specData: assets.GithubSpec,
		cacheDir: cacheDir,
		ttl:      ttl,
	})

	addBuiltin(r, cfg, builtinDef{
		name:     jenkinsName,
		specData: assets.JenkinsSpec,
		cacheDir: cacheDir,
		ttl:      ttl,
	})

	addBuiltin(r, cfg, builtinDef{
		name:     eurName,
		specData: assets.EURSpec,
		cacheDir: cacheDir,
		ttl:      ttl,
	})
}

type builtinDef struct {
	name      string
	specData  []byte
	cacheDir  string
	ttl       time.Duration
	normalize spec.NormalizeRule
}

func addBuiltin(r *Registry, cfg *config.Config, b builtinDef) {
	// base_url comes entirely from user config; empty string is allowed here —
	// the executor will surface a clear "base_url is not set" error at call time.
	baseURL := ""
	if svc, ok := cfg.Services[b.name]; ok {
		baseURL = svc.BaseURL
	}

	r.entries[b.name] = &Entry{
		Name:       b.name,
		BaseURL:    baseURL,
		SpecURL:    "", // embedded spec — no remote URL needed
		loader:     spec.NewEmbeddedLoader(b.name, b.specData, b.cacheDir, b.ttl),
		normalizer: spec.NewNormalizer(b.normalize),
	}

	// Ensure cfg.Services contains an entry for this built-in so the executor
	// can look it up. If the user omitted the service entirely from config, add
	// a stub; if they configured it (e.g. auth-only), preserve all their values.
	if _, ok := cfg.Services[b.name]; !ok {
		cfg.Services[b.name] = config.ServiceConfig{}
	}
}
