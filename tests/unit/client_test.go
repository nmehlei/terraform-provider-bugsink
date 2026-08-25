package bugsink_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nmehlei/terraform-provider-bugsink/src/bugsink"
)

const testToken = "super-secret-test-token"

func newClient(t *testing.T, srv *httptest.Server, allowInsecure bool) *bugsink.Client {
	t.Helper()
	c, err := bugsink.New(context.Background(), bugsink.Config{
		Endpoint:      srv.URL,
		Token:         testToken,
		AllowInsecure: allowInsecure,
		HTTPClient:    srv.Client(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

func schemaHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONResponse(w, map[string]interface{}{"info": map[string]string{"version": version}})
	}
}

// 4.1 Test bearer header is set on every request.
func TestBearerHeaderSetOnEveryRequest(t *testing.T) {
	var mu sync.Mutex
	var gotAuth []string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		mu.Unlock()
		schemaHandler("2.5.0")(w, r)
	})
	mux.HandleFunc("/api/canonical/0/teams/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		mu.Unlock()
		writeJSONResponse(w, map[string]interface{}{"next": "", "results": []bugsink.Team{}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newClient(t, srv, true)
	if _, err := c.ListTeams(context.Background(), ""); err != nil {
		t.Fatalf("ListTeams() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(gotAuth) != 2 {
		t.Fatalf("got %d requests, want 2 (schema + list)", len(gotAuth))
	}
	for i, auth := range gotAuth {
		if auth != "Bearer "+testToken {
			t.Fatalf("request %d Authorization = %q, want Bearer token", i, auth)
		}
	}
}

// 4.2 Test the token and any endpoint credentials never appear in an error.
func TestErrorsNeverContainCredentials(t *testing.T) {
	t.Run("non-2xx response", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/canonical/0/schema/", schemaHandler("2.5.0"))
		mux.HandleFunc("/api/canonical/0/teams/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("boom"))
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		c := newClient(t, srv, true)
		_, err := c.ListTeams(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if strings.Contains(err.Error(), testToken) {
			t.Fatalf("error message leaked token: %v", err)
		}
	})

	t.Run("network error", func(t *testing.T) {
		// A server that closes without responding forces the *http.Client
		// Do() error path (client.go's "network error" branch), distinct
		// from the non-2xx APIError path above.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("test server does not support hijacking")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			conn.Close()
		}))
		defer srv.Close()

		_, err := bugsink.New(context.Background(), bugsink.Config{
			Endpoint:      srv.URL,
			Token:         testToken,
			AllowInsecure: true,
			HTTPClient:    srv.Client(),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if strings.Contains(err.Error(), testToken) {
			t.Fatalf("error message leaked token: %v", err)
		}
	})

	t.Run("credentials embedded in endpoint", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/canonical/0/schema/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("boom"))
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		endpoint := strings.Replace(srv.URL, "http://", "http://admin:hunter2@", 1)
		_, err := bugsink.New(context.Background(), bugsink.Config{
			Endpoint:      endpoint,
			Token:         testToken,
			AllowInsecure: true,
			HTTPClient:    srv.Client(),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if strings.Contains(err.Error(), "hunter2") {
			t.Fatalf("error message leaked endpoint password: %v", err)
		}
	})
}

// 4.3 Test plain HTTP rejected by default and accepted with AllowInsecure.
func TestPlainHTTPRejectedByDefault(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", schemaHandler("2.5.0"))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := bugsink.New(context.Background(), bugsink.Config{
		Endpoint:   srv.URL,
		Token:      testToken,
		HTTPClient: srv.Client(),
	})
	var cfgErr *bugsink.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("New() error = %v, want *ConfigError", err)
	}

	if _, err := bugsink.New(context.Background(), bugsink.Config{
		Endpoint:      srv.URL,
		Token:         testToken,
		AllowInsecure: true,
		HTTPClient:    srv.Client(),
	}); err != nil {
		t.Fatalf("New() with AllowInsecure error = %v", err)
	}
}

// 4.4 Test timeout aborts a hung request with an actual deadline error, not
// just "any error" (which would also pass for e.g. connection refused).
func TestRequestTimeoutAborts(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
		schemaHandler("2.5.0")(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := bugsink.New(context.Background(), bugsink.Config{
		Endpoint:      srv.URL,
		Token:         testToken,
		AllowInsecure: true,
		Timeout:       10 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want it to wrap context.DeadlineExceeded", err)
	}
}

// The timeout must hold even when the caller supplies their own HTTPClient
// (e.g. every other test in this file) - it must not rely solely on
// http.Client.Timeout, which a caller-supplied client might leave unset.
func TestRequestTimeoutAppliesWithCustomHTTPClient(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
		schemaHandler("2.5.0")(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	start := time.Now()
	_, err := bugsink.New(context.Background(), bugsink.Config{
		Endpoint:      srv.URL,
		Token:         testToken,
		AllowInsecure: true,
		Timeout:       50 * time.Millisecond,
		HTTPClient:    srv.Client(), // no Timeout set on this client
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > time.Second {
		t.Fatalf("New() took %v with a 50ms configured timeout and a client with no timeout of its own", elapsed)
	}
}

// 4.5 Test pagination follows a server-provided next-page URL verbatim.
func TestPaginationFollowsServerURL(t *testing.T) {
	var mu sync.Mutex
	var requestedPaths []string
	var nextURL string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", schemaHandler("2.5.0"))
	mux.HandleFunc("/api/canonical/0/teams/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestedPaths = append(requestedPaths, r.URL.String())
		mu.Unlock()
		writeJSONResponse(w, map[string]interface{}{
			"next":    nextURL,
			"results": []bugsink.Team{{ID: "1", Name: "first"}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	nextURL = srv.URL + "/api/canonical/0/teams/?cursor=opaque-server-cursor"

	c := newClient(t, srv, true)

	page1, err := c.ListTeams(context.Background(), "")
	if err != nil {
		t.Fatalf("ListTeams() page1 error = %v", err)
	}
	if page1.Next == "" {
		t.Fatal("expected non-empty Next cursor URL")
	}

	if _, err := c.ListTeams(context.Background(), page1.Next); err != nil {
		t.Fatalf("ListTeams() page2 error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requestedPaths) != 2 {
		t.Fatalf("requestedPaths = %v, want 2 requests", requestedPaths)
	}
	if !strings.Contains(requestedPaths[1], "cursor=opaque-server-cursor") {
		t.Fatalf("second request path = %q, want it to contain the server's cursor verbatim", requestedPaths[1])
	}
}

// A "next" link pointing at a different host must be refused: the client
// attaches the bearer token to every request, so blindly following a
// server-controlled URL would let a compromised or misconfigured Bugsink
// instance redirect the token to an arbitrary host.
func TestPaginationRefusesDifferentHost(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", schemaHandler("2.5.0"))
	mux.HandleFunc("/api/canonical/0/teams/", func(w http.ResponseWriter, r *http.Request) {
		writeJSONResponse(w, map[string]interface{}{
			"next":    "",
			"results": []bugsink.Team{},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newClient(t, srv, true)

	_, err := c.ListTeams(context.Background(), "http://evil.example.invalid/api/canonical/0/teams/?cursor=x")
	if err == nil {
		t.Fatal("expected error following a cross-host pagination URL, got nil")
	}
	var cfgErr *bugsink.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("err = %v, want *ConfigError", err)
	}
}

// A "next" link embedding credentials must also be refused, for the same
// reason: nothing about a pagination cursor should ever need or receive
// its own auth.
func TestPaginationRefusesEmbeddedCredentials(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", schemaHandler("2.5.0"))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newClient(t, srv, true)

	maliciousNext := strings.Replace(srv.URL, "http://", "http://user:pass@", 1) + "/api/canonical/0/teams/"
	_, err := c.ListTeams(context.Background(), maliciousNext)
	if err == nil {
		t.Fatal("expected error following a pagination URL with embedded credentials, got nil")
	}
}

// 4.6 Test version validation accepts Bugsink 2.5 and rejects other
// versions, including ones that share the "2.5" string prefix without
// being a 2.5.x patch release (e.g. 2.50.0, 2.51.2).
func TestVersionValidation(t *testing.T) {
	cases := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"exact minor, no patch", "2.5", false},
		{"supported patch", "2.5.3", false},
		{"unsupported major", "3.0.0", true},
		{"unsupported minor", "2.6.0", true},
		{"prefix collision, minor 50", "2.50.0", true},
		{"prefix collision, minor 51", "2.51.2", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/canonical/0/schema/", schemaHandler(tc.version))
			srv := httptest.NewServer(mux)
			defer srv.Close()

			_, err := bugsink.New(context.Background(), bugsink.Config{
				Endpoint:      srv.URL,
				Token:         testToken,
				AllowInsecure: true,
				HTTPClient:    srv.Client(),
			})
			if tc.wantErr && err == nil {
				t.Fatalf("New() with version %q: expected error, got nil", tc.version)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("New() with version %q: unexpected error: %v", tc.version, err)
			}
		})
	}
}

// 4.7 Test PATCH only changes specified fields.
func TestPatchOnlyChangesSpecifiedFields(t *testing.T) {
	var mu sync.Mutex
	stored := bugsink.Team{ID: "1", Name: "original", Visibility: "internal"}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", schemaHandler("2.5.0"))
	mux.HandleFunc("/api/canonical/0/teams/1/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "want PATCH", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if _, hasVisibility := body["visibility"]; hasVisibility {
			http.Error(w, "unexpected visibility in PATCH body", http.StatusBadRequest)
			return
		}
		name, ok := body["name"].(string)
		if !ok || name == "" {
			http.Error(w, "missing name in PATCH body", http.StatusBadRequest)
			return
		}

		mu.Lock()
		stored.Name = name
		snapshot := stored
		mu.Unlock()

		writeJSONResponse(w, snapshot)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newClient(t, srv, true)
	newName := "renamed"
	updated, err := c.UpdateTeam(context.Background(), "1", bugsink.UpdateTeamRequest{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateTeam() error = %v", err)
	}
	if updated.Name != "renamed" {
		t.Fatalf("updated.Name = %q, want %q", updated.Name, "renamed")
	}
	if updated.Visibility != "internal" {
		t.Fatalf("updated.Visibility = %q, want unchanged %q", updated.Visibility, "internal")
	}
}

// CreateProjectRequest must be able to send an explicit false/0, not just
// omit the field and rely on the server's default - see the comment on the
// type itself for why plain bool/int with `omitempty` can't do this.
func TestCreateProjectRequestSendsExplicitFalseAndZero(t *testing.T) {
	var mu sync.Mutex
	var gotBody map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", schemaHandler("2.5.0"))
	mux.HandleFunc("/api/canonical/0/projects/", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		mu.Lock()
		gotBody = body
		mu.Unlock()
		writeJSONResponse(w, bugsink.Project{ID: "1", Team: "t1", Name: "p"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newClient(t, srv, true)
	no := false
	zero := 0
	_, err := c.CreateProject(context.Background(), bugsink.CreateProjectRequest{
		Team:                   "t1",
		Name:                   "p",
		AlertOnNewIssue:        &no,
		RetentionMaxEventCount: &zero,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if v, ok := gotBody["alert_on_new_issue"]; !ok || v != false {
		t.Fatalf(`request body["alert_on_new_issue"] = %v (present=%v), want explicit false`, v, ok)
	}
	if v, ok := gotBody["retention_max_event_count"]; !ok || v != float64(0) {
		t.Fatalf(`request body["retention_max_event_count"] = %v (present=%v), want explicit 0`, v, ok)
	}
	if _, ok := gotBody["alert_on_regression"]; ok {
		t.Fatalf(`request body unexpectedly included unset field "alert_on_regression": %v`, gotBody)
	}
}

// 3.3 Confirm the client's exported API is exactly what's expected - in
// particular that it never grows a delete method under any name, since
// Bugsink 2.5 has no delete endpoint to back one.
func TestClientExportedMethodSet(t *testing.T) {
	want := []string{
		"CreateProject",
		"CreateTeam",
		"GetProject",
		"GetTeam",
		"ListProjects",
		"ListTeams",
		"UpdateProject",
		"UpdateTeam",
	}

	clientType := reflect.TypeOf(&bugsink.Client{})
	var got []string
	for i := 0; i < clientType.NumMethod(); i++ {
		got = append(got, clientType.Method(i).Name)
	}
	sort.Strings(got)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Client exported methods = %v, want exactly %v", got, want)
	}
}

func writeJSONResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
