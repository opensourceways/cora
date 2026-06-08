package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opensourceways/cora/internal/auth"
	"github.com/opensourceways/cora/internal/config"
	"github.com/opensourceways/cora/internal/log"
	"github.com/opensourceways/cora/internal/output"
	"github.com/opensourceways/cora/internal/view"
	"github.com/opensourceways/cora/pkg/errs"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 2 // up to 3 total attempts
	retryBaseDelay = 500 * time.Millisecond
)

// Request is the input to a single HTTP API call.
type Request struct {
	ServiceName  string
	PathTemplate string            // e.g. "/posts/{id}.json"
	Method       string            // "GET", "POST", …
	PathParams   map[string]string // {id} → "123"
	QueryParams  map[string]string
	Body         map[string]any
	Format       string // "table" | "json" | "yaml"
	DryRun       bool
	ViewConfig   *view.ViewConfig // nil → generic fallback rendering
}

// Executor executes API requests against configured backend services.
type Executor struct {
	cfg    *config.Config
	client *http.Client
}

// New creates an Executor backed by the given config.
func New(cfg *config.Config) *Executor {
	return &Executor{cfg: cfg, client: &http.Client{Timeout: defaultTimeout}}
}

// ExecuteRaw performs the HTTP request and returns the raw response body bytes.
// It does not print to stdout — useful for programmatic two-step flows
// (e.g. fetch build details to discover artifacts, then download one).
func (e *Executor) ExecuteRaw(ctx context.Context, req *Request) ([]byte, error) {
	_, respBytes, err := e.doRequest(ctx, req)
	return respBytes, err
}

// Execute performs the HTTP request described by req, formats the response,
// and writes it to stdout.  Errors are returned as CLIErrors.
func (e *Executor) Execute(ctx context.Context, req *Request) error {
	resp, respBytes, err := e.doRequest(ctx, req)
	if err != nil {
		return err
	}
	return e.writeResponse(resp, respBytes, req)
}

// doRequest builds the URL, injects auth, executes the HTTP request (with
// retries), and returns the response metadata and body bytes. It does NOT
// print anything.
func (e *Executor) doRequest(ctx context.Context, req *Request) (*http.Response, []byte, error) {
	svcCfg, ok := e.cfg.Services[req.ServiceName]
	if !ok {
		return nil, nil, errs.NewConfigError(fmt.Sprintf("service %q not found in config", req.ServiceName))
	}

	baseURL := strings.TrimRight(svcCfg.BaseURL, "/")
	if baseURL == "" {
		return nil, nil, errs.NewConfigError(fmt.Sprintf("service %q: base_url is not set", req.ServiceName))
	}

	// Substitute path parameters: /posts/{id}.json → /posts/123.json
	path := req.PathTemplate
	for k, v := range req.PathParams {
		path = strings.ReplaceAll(path, "{"+k+"}", escapePathParam(v))
	}

	// Jenkins jobs nested inside a view need /view/{view} prepended to
	// /job/ paths so the Ingress routes them correctly.
	if svcCfg.View != "" && strings.HasPrefix(path, "/job/") {
		path = "/view/" + svcCfg.View + path
	}

	// Build full URL with query string
	fullURL := baseURL + path
	if len(req.QueryParams) > 0 {
		q := url.Values{}
		for k, v := range req.QueryParams {
			q.Set(k, v)
		}
		fullURL += "?" + q.Encode()
	}

	// Serialise request body. Jenkins POST/PUT endpoints use Stapler and
	// require form-urlencoded with a json= wrapper; other services use JSON.
	var bodyReader io.Reader
	contentType := ""
	isJenkins := req.ServiceName == "jenkins"
	isMutating := req.Method != http.MethodGet && req.Method != http.MethodHead
	if len(req.Body) > 0 {
		b, err := json.Marshal(req.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal body: %w", err)
		}
		if isJenkins && isMutating {
			form := url.Values{}
			form.Set("json", string(b))
			bodyReader = bytes.NewReader([]byte(form.Encode()))
			contentType = "application/x-www-form-urlencoded"
		} else {
			bodyReader = bytes.NewReader(b)
			contentType = "application/json"
		}
	}

	// Build HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	if bodyReader != nil {
		httpReq.Header.Set("Content-Type", contentType)
	}
	httpReq.Header.Set("Accept", "application/json")

	// Inject auth credentials (Discourse: headers; Etherpad: ?apikey= query param).
	auth.InjectAuth(httpReq, svcCfg, req.ServiceName)

	// For mutating requests, attach a CSRF crumb if the service requires it.
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		if err = auth.AttachCrumb(ctx, httpReq, svcCfg, req.ServiceName); err != nil {
			return nil, nil, classifyError(err)
		}
	}

	// Log the outgoing request (after auth injection so the masked URL is accurate).
	bodySize := 0
	if len(req.Body) > 0 {
		if b, err2 := json.Marshal(req.Body); err2 == nil {
			bodySize = len(b)
		}
	}
	log.Debug("→ %s %s  [body: %d bytes]", req.Method, log.MaskURL(httpReq.URL.String()), bodySize)

	// --dry-run: print what would be sent and exit
	if req.DryRun {
		fmt.Printf("[dry-run] %s %s\n", req.Method, httpReq.URL.String())
		if len(req.Body) > 0 {
			pretty, _ := json.MarshalIndent(req.Body, "", "  ")
			fmt.Printf("Body:\n%s\n", pretty)
		}
		return nil, nil, nil
	}

	// Execute (with retry for transient network errors on idempotent methods)
	start := time.Now()
	resp, respBytes, err := e.doWithRetry(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		return nil, nil, classifyError(err)
	}

	log.Debug("← %s (%d bytes, %dms)", resp.Status, len(respBytes), elapsed.Milliseconds())
	log.Debug("response body: %s", log.FormatBody(respBytes, 3072))

	return resp, respBytes, nil
}

// writeResponse formats and prints the HTTP response to stdout.
func (e *Executor) writeResponse(resp *http.Response, respBytes []byte, req *Request) error {
	if resp == nil {
		return nil // dry-run
	}

	// Treat 4xx/5xx as API errors
	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("API error %d", resp.StatusCode)
		if len(respBytes) > 0 {
			msg += ": " + truncate(string(respBytes), 300)
		}
		return errs.NewAPIError(msg, nil)
	}

	// Empty body (e.g. 204 No Content, 201 Created)
	if len(respBytes) == 0 {
		if loc := resp.Header.Get("Location"); loc != "" {
			fmt.Println("Location:", loc)
		} else {
			fmt.Println("OK")
		}
		return nil
	}

	// Non-JSON responses (e.g. artifact downloads) — print raw content.
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "json") && !strings.Contains(ct, "html") {
		fmt.Print(string(respBytes))
		return nil
	}

	format := req.Format
	if format == "" {
		format = "table"
	}
	return output.Print(respBytes, format, req.ViewConfig)
}

// doWithRetry executes the HTTP request and reads the full response body.
// For idempotent methods (GET, HEAD), transient network errors trigger up to
// maxRetries additional attempts with exponential backoff.
func (e *Executor) doWithRetry(req *http.Request) (*http.Response, []byte, error) {
	idempotent := req.Method == http.MethodGet || req.Method == http.MethodHead

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay << (attempt - 1)
			log.Debug("retry %d/%d after %v (last error: %v)", attempt, maxRetries, delay, lastErr)
			select {
			case <-req.Context().Done():
				return nil, nil, req.Context().Err()
			case <-time.After(delay):
			}
		}

		resp, err := e.client.Do(req)
		if err != nil {
			if idempotent && isTransientNetworkError(err) {
				lastErr = err
				continue
			}
			return nil, nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("read response body: %w", err)
		}

		return resp, body, nil
	}

	return nil, nil, lastErr
}

// isTransientNetworkError returns true for errors that are safe to retry:
// connection resets, unexpected EOF from the server, and temporary errors.
// Context cancellation and deadline exceeded are NOT retried.
func isTransientNetworkError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Temporary() {
			return true
		}
		msg := urlErr.Err.Error()
		return strings.Contains(msg, "EOF") ||
			strings.Contains(msg, "connection reset") ||
			strings.Contains(msg, "broken pipe")
	}
	return false
}

// classifyError converts raw http.Client errors into user-friendly CLIErrors,
// distinguishing timeouts, context cancellation, and generic network failures.
func classifyError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errs.NewAPIError(fmt.Sprintf("request timed out after %v", defaultTimeout), nil)
	}
	if errors.Is(err, context.Canceled) {
		return errs.NewAPIError("request cancelled", nil)
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return errs.NewAPIError(fmt.Sprintf("request timed out after %v", defaultTimeout), nil)
		}
		masked := fmt.Sprintf("%s %q: %v", urlErr.Op, log.MaskURL(urlErr.URL), urlErr.Err)
		return errs.NewAPIError("request failed: "+masked, nil)
	}
	return errs.NewAPIError("request failed", err)
}

// escapePathParam is like url.PathEscape but preserves '/' so that
// nested resource paths (e.g. Jenkins folder jobs like "folder/job/name")
// work correctly in path parameter substitution.
func escapePathParam(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), "%2F", "/")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
