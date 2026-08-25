package bugsink

import "fmt"

// ConfigError indicates the Client was misconfigured (bad endpoint, missing
// token, unsupported remote API version) or a request failed during
// configuration (e.g. fetching the schema for the version check). It never
// includes the token or endpoint credentials. Err, when set, can be
// inspected with errors.As/errors.Is (e.g. to distinguish a network/timeout
// failure from a genuine configuration mistake).
type ConfigError struct {
	Msg string
	Err error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("bugsink: %s: %v", e.Msg, e.Err)
	}
	return "bugsink: " + e.Msg
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// APIError represents a non-2xx response from the Bugsink API. It is built
// only from the method, request URL, status code, and response body -
// never from request headers - so it never includes the bearer token. The
// request URL itself never carries credentials either: Client strips any
// endpoint userinfo in New and refuses to follow a pagination URL that
// embeds credentials or targets a different host (see resolve in client.go).
type APIError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	body := e.Body
	const maxErrBody = 2048
	if len(body) > maxErrBody {
		body = body[:maxErrBody] + "... (truncated)"
	}
	return fmt.Sprintf("bugsink: %s %s: unexpected status %d: %s", e.Method, e.URL, e.StatusCode, body)
}

// NotFound reports whether the error is a 404, the condition under which
// callers should treat a resource as removed from the remote and drop it
// from Terraform state instead of surfacing a hard error.
func (e *APIError) NotFound() bool {
	return e.StatusCode == 404
}
