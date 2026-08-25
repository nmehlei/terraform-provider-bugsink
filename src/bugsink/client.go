// Package bugsink is a minimal, hand-written HTTP client for a Bugsink
// instance's canonical API (/api/canonical/0/). It intentionally has no
// dependency on the Terraform Plugin Framework so it can be used and tested
// standalone.
package bugsink

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// supportedAPIMajorMinor is the only Bugsink API version this client
	// validates against. See docs/api-contract.md: "Start with Bugsink 2.5
	// as the only supported version."
	supportedAPIMajorMinor = "2.5"

	defaultTimeout = 30 * time.Second

	// canonicalPrefix is relative to the endpoint's own path, so a
	// reverse-proxied install under a path prefix (e.g.
	// https://host/bugsink/) is resolved correctly instead of having its
	// prefix silently discarded.
	canonicalPrefix = "api/canonical/0/"
	schemaPath      = "schema/"

	// maxResponseBody bounds how much of a response this client will ever
	// read into memory or embed in an error, so a misconfigured proxy
	// returning a large HTML page can't produce a multi-MB diagnostic.
	maxResponseBody = 1 << 20 // 1 MiB
)

// Config configures a Client.
type Config struct {
	// Endpoint is the base URL of the Bugsink instance, e.g.
	// "https://bugsink.example.com". Must be HTTPS unless AllowInsecure is set.
	Endpoint string

	// Token authenticates every request as a bearer token. The client does
	// not read this from the environment; callers source it from
	// BUGSINK_TOKEN or a sensitive provider attribute themselves.
	Token string

	// Timeout bounds every HTTP request. Defaults to 30s when zero. Applied
	// per-request via context, independent of HTTPClient below, so it holds
	// even when HTTPClient is overridden (e.g. in tests).
	Timeout time.Duration

	// AllowInsecure permits a plain-HTTP endpoint. Intended only for local
	// development against a disposable instance.
	AllowInsecure bool

	// HTTPClient overrides the underlying HTTP client. Primarily for tests.
	HTTPClient *http.Client
}

// Client is a Bugsink canonical API client for teams and projects. It never
// exposes a delete operation: Bugsink 2.5 does not document one.
type Client struct {
	// baseURL is the resolved canonical API root (endpoint + canonicalPrefix),
	// with any userinfo stripped so it can never leak into a request URL or
	// error message.
	baseURL    *url.URL
	token      string
	timeout    time.Duration
	httpClient *http.Client
}

// New validates cfg, fetches and validates the remote API version, and
// returns a ready Client. It fails closed if the remote version is not the
// supported Bugsink 2.5 API, per docs/api-contract.md's compatibility rule.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, &ConfigError{Msg: "endpoint is required"}
	}
	if cfg.Token == "" {
		return nil, &ConfigError{Msg: "token is required"}
	}

	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, &ConfigError{Msg: "invalid endpoint", Err: err}
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !cfg.AllowInsecure {
			return nil, &ConfigError{Msg: "endpoint uses plain HTTP; set AllowInsecure for local development only"}
		}
	default:
		return nil, &ConfigError{Msg: fmt.Sprintf("unsupported endpoint scheme %q", u.Scheme)}
	}

	base := *u
	base.User = nil // never let a credential embedded in the endpoint reach a request URL or error message
	base.RawQuery = ""
	base.Fragment = ""
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	base.Path += canonicalPrefix

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	c := &Client{
		baseURL:    &base,
		token:      cfg.Token,
		timeout:    timeout,
		httpClient: httpClient,
	}

	if err := c.checkVersion(ctx); err != nil {
		return nil, err
	}

	return c, nil
}

// openAPISchema is the minimal subset of an OpenAPI document this client
// reads. Every Bugsink installation serves its own schema at schemaPath.
type openAPISchema struct {
	Info struct {
		Version string `json:"version"`
	} `json:"info"`
}

func isSupportedVersion(v string) bool {
	return v == supportedAPIMajorMinor || strings.HasPrefix(v, supportedAPIMajorMinor+".")
}

func (c *Client) checkVersion(ctx context.Context) error {
	var schema openAPISchema
	if err := c.do(ctx, http.MethodGet, schemaPath, nil, &schema); err != nil {
		return &ConfigError{Msg: "fetching API schema", Err: err}
	}

	if !isSupportedVersion(schema.Info.Version) {
		return &ConfigError{Msg: fmt.Sprintf(
			"unsupported Bugsink API version %q: this client only supports %s.x",
			schema.Info.Version, supportedAPIMajorMinor,
		)}
	}

	return nil
}

// do issues a request against a path relative to the canonical API root (or
// an absolute pagination URL previously returned by the server, restricted
// to the same scheme+host as the configured endpoint) and decodes a JSON
// response body into out when out is non-nil. The configured timeout is
// applied to every call via ctx, regardless of any HTTPClient override.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	reqURL, err := c.resolve(path)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "terraform-provider-bugsink-client")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// req/reqURL never carry credentials (see resolve and New), so this
		// error chain never contains the token.
		return fmt.Errorf("bugsink: %s %s: %w", method, reqURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return fmt.Errorf("bugsink: %s %s: reading response: %w", method, reqURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			Method:     method,
			URL:        reqURL,
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("bugsink: %s %s: decoding response (status %d, content-type %q): %w",
			method, reqURL, resp.StatusCode, resp.Header.Get("Content-Type"), err)
	}
	return nil
}

// resolve turns path into a request URL. A relative path is resolved
// against the canonical API root. An absolute URL (as returned by the
// server for pagination) is only followed when it targets the same
// scheme+host as the configured endpoint and carries no credentials of its
// own - otherwise a malicious or misconfigured server could redirect the
// client's bearer token to an arbitrary host via a "next" link.
func (c *Client) resolve(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		u, err := url.Parse(path)
		if err != nil {
			return "", fmt.Errorf("bugsink: invalid pagination URL %q: %w", path, err)
		}
		if !strings.EqualFold(u.Scheme, c.baseURL.Scheme) || !strings.EqualFold(u.Host, c.baseURL.Host) {
			return "", &ConfigError{Msg: fmt.Sprintf(
				"refusing to follow pagination URL to a different host (%s://%s), configured endpoint is %s://%s",
				u.Scheme, u.Host, c.baseURL.Scheme, c.baseURL.Host,
			)}
		}
		if u.User != nil {
			return "", &ConfigError{Msg: "refusing to follow pagination URL that embeds credentials"}
		}
		return u.String(), nil
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("bugsink: invalid path %q: %w", path, err)
	}
	return c.baseURL.ResolveReference(ref).String(), nil
}
