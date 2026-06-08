package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/opensourceways/cora/internal/config"
	"github.com/opensourceways/cora/internal/log"
)

// InjectAuth adds authentication credentials to an outgoing request based on
// the service's configured auth provider. svcName is used only for log output.
//
// Discourse: injects Api-Key and Api-Username headers.
// Etherpad:  injects ?apikey= into the request URL's query string.
// GitCode:   injects ?access_token= into the request URL's query string.
// GitHub:    injects Authorization: Bearer <token> header.
// Jenkins:   injects Authorization: Basic <base64(user:token)> header.
// EUR:       injects Authorization: Basic <base64(user:token)> header.
//
// All providers inject credentials unconditionally when present; the server
// ignores them for public endpoints and enforces them for protected ones.
func InjectAuth(req *http.Request, svc config.ServiceConfig, svcName string) {
	if d := svc.Auth.Discourse; d != nil {
		if d.APIKey != "" {
			req.Header.Set("Api-Key", d.APIKey)
		}
		if d.APIUsername != "" {
			req.Header.Set("Api-Username", d.APIUsername)
		}
		log.Debug("auth: injecting discourse headers for service %q", svcName)
	}

	if e := svc.Auth.Etherpad; e != nil && e.APIKey != "" {
		q := req.URL.Query()
		q.Set("apikey", e.APIKey)
		req.URL.RawQuery = q.Encode()
		log.Debug("auth: injecting etherpad apikey for service %q", svcName)
	}

	if g := svc.Auth.Gitcode; g != nil && g.AccessToken != "" {
		q := req.URL.Query()
		q.Set("access_token", g.AccessToken)
		req.URL.RawQuery = q.Encode()
		log.Debug("auth: injecting gitcode access_token for service %q", svcName)
	}

	if h := svc.Auth.Github; h != nil && h.Token != "" {
		req.Header.Set("Authorization", "Bearer "+h.Token)
		// GitHub recommends sending an explicit API version header; this also
		// pins responses to a stable shape regardless of server-side rollouts.
		if req.Header.Get("X-GitHub-Api-Version") == "" {
			req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		}
		log.Debug("auth: injecting github bearer token for service %q", svcName)
	}

	if j := svc.Auth.Jenkins; j != nil && j.Username != "" && j.APIToken != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(j.Username + ":" + j.APIToken))
		req.Header.Set("Authorization", "Basic "+auth)
		log.Debug("auth: injecting jenkins basic auth for service %q", svcName)
	}

	if e := svc.Auth.EUR; e != nil && e.Username != "" && e.APIToken != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(e.Username + ":" + e.APIToken))
		req.Header.Set("Authorization", "Basic "+auth)
		log.Debug("auth: injecting eur basic auth for service %q", svcName)
	}
}

// IsDiscourseAuthParam reports whether an OpenAPI parameter is one of the
// Discourse auth headers that should be injected automatically (not exposed
// to the user as a CLI flag).
func IsDiscourseAuthParam(name string) bool {
	return name == "Api-Key" || name == "Api-Username"
}

// IsGitcodeAuthParam reports whether an OpenAPI parameter is a GitCode auth
// parameter that should be injected automatically (not exposed as a CLI flag).
// GitCode uses ?access_token= (PAT) and Authorization header (OAuth Bearer).
func IsGitcodeAuthParam(name string) bool {
	return name == "access_token" || name == "Authorization"
}

// AttachCrumb fetches a Jenkins CSRF crumb and attaches it as a header on the
// request. Only acts when the service has Jenkins auth configured and the HTTP
// method is not GET or HEAD (which don't require CSRF protection).
//
// The crumb is fetched from {baseURL}/crumbIssuer/api/json using Basic Auth.
func AttachCrumb(ctx context.Context, req *http.Request, svc config.ServiceConfig, svcName string) error {
	j := svc.Auth.Jenkins
	if j == nil || j.Username == "" || j.APIToken == "" {
		return nil
	}
	if req.Method == http.MethodGet || req.Method == http.MethodHead {
		return nil
	}

	baseURL := svc.BaseURL
	if baseURL == "" {
		return nil
	}
	crumbURL := baseURL + "/crumbIssuer/api/json"

	crumbReq, err := http.NewRequestWithContext(ctx, http.MethodGet, crumbURL, nil)
	if err != nil {
		return fmt.Errorf("jenkins crumb: build request: %w", err)
	}
	auth := base64.StdEncoding.EncodeToString([]byte(j.Username + ":" + j.APIToken))
	crumbReq.Header.Set("Authorization", "Basic "+auth)

	resp, err := http.DefaultClient.Do(crumbReq)
	if err != nil {
		return fmt.Errorf("jenkins crumb: fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("jenkins crumb: server returned %d: %s", resp.StatusCode, string(body))
	}

	var crumb struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&crumb); err != nil {
		return fmt.Errorf("jenkins crumb: parse response: %w", err)
	}
	if crumb.Crumb == "" {
		return fmt.Errorf("jenkins crumb: empty crumb in response")
	}

	field := crumb.CrumbRequestField
	if field == "" {
		field = "Jenkins-Crumb"
	}
	req.Header.Set(field, crumb.Crumb)
	log.Debug("auth: attached jenkins crumb for service %q", svcName)
	return nil
}
