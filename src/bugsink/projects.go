package bugsink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const projectsPath = "projects/"

// ProjectID holds a project id. Confirmed empirically against a live
// Bugsink 2.5 instance: the id round-trips inconsistently across
// endpoints - a JSON string ("2") in the create response, a bare JSON
// number (2) on retrieve and list. A plain Go string field fails to
// unmarshal the latter, so ProjectID accepts either wire form and always
// behaves as a string from the caller's side.
type ProjectID string

func (id ProjectID) String() string { return string(id) }

func (id ProjectID) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(id))
}

func (id *ProjectID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*id = ProjectID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("bugsink: project id %s is neither a JSON string nor number", data)
	}
	*id = ProjectID(n.String())
	return nil
}

// Project is a Bugsink project, per docs/api-contract.md's canonical
// fields. DSN is treated as sensitive by every caller of this client: it is
// never logged or included in an error (see errors.go). This is a
// read/response type, so its fields intentionally have no `omitempty`: that
// tag only affects marshaling, and a false/zero value here is meaningful
// server state, not an absent one (see CreateProjectRequest for the
// create-time version of this distinction).
//
// The create response only populates ID/Team/Name/Visibility/the alert
// flags/RetentionMaxEventCount - Slug, DSN, and GroupingMechanism are only
// present on retrieve/list (confirmed empirically), matching
// docs/api-contract.md's "read after every create/update" rule. Callers
// that need the DSN must GetProject after CreateProject.
type Project struct {
	ID                     ProjectID `json:"id"`
	Team                   string    `json:"team"`
	Name                   string    `json:"name"`
	Visibility             string    `json:"visibility"`
	AlertOnNewIssue        bool      `json:"alert_on_new_issue"`
	AlertOnRegression      bool      `json:"alert_on_regression"`
	AlertOnUnmute          bool      `json:"alert_on_unmute"`
	RetentionMaxEventCount int       `json:"retention_max_event_count"`

	// Computed / read-only fields. Only populated by retrieve/list, not create.
	Slug              string `json:"slug"`
	DSN               string `json:"dsn"`
	GroupingMechanism string `json:"grouping_mechanism"`
}

// ProjectPage is one page of a project listing, following the server's
// cursor pagination.
type ProjectPage struct {
	Next    string    `json:"next"`
	Results []Project `json:"results"`
}

// ListProjects fetches one page of projects. Pass an empty pageURL for the
// first page and ProjectPage.Next for subsequent pages; the client never
// constructs cursor values itself.
func (c *Client) ListProjects(ctx context.Context, pageURL string) (*ProjectPage, error) {
	path := pageURL
	if path == "" {
		path = projectsPath
	}
	var page ProjectPage
	if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CreateProjectRequest is the writable subset of Project accepted on
// create. AlertOnNewIssue, AlertOnRegression, and RetentionMaxEventCount
// are pointers, not plain bool/int: a plain `bool` with `omitempty` can
// never send `false`, and a plain `int` with `omitempty` can never send
// `0` - both would silently fall back to the server's default instead of
// the caller's explicit choice. Leave a field nil to accept the server
// default; set it to send an explicit value, including false/0.
type CreateProjectRequest struct {
	Team                   string `json:"team"`
	Name                   string `json:"name"`
	Visibility             string `json:"visibility,omitempty"`
	AlertOnNewIssue        *bool  `json:"alert_on_new_issue,omitempty"`
	AlertOnRegression      *bool  `json:"alert_on_regression,omitempty"`
	AlertOnUnmute          *bool  `json:"alert_on_unmute,omitempty"`
	RetentionMaxEventCount *int   `json:"retention_max_event_count,omitempty"`
}

// CreateProject creates a project and returns the server's representation
// of it, including computed fields such as Slug and DSN.
func (c *Client) CreateProject(ctx context.Context, in CreateProjectRequest) (*Project, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out Project
	if err := c.do(ctx, http.MethodPost, projectsPath, bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetProject retrieves a project by ID.
func (c *Client) GetProject(ctx context.Context, id ProjectID) (*Project, error) {
	var out Project
	if err := c.do(ctx, http.MethodGet, projectsPath+id.String()+"/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateProjectRequest is a partial update: only non-nil fields are sent,
// so unset fields are left unchanged on the server (PATCH semantics).
type UpdateProjectRequest struct {
	Name                   *string `json:"name,omitempty"`
	Visibility             *string `json:"visibility,omitempty"`
	AlertOnNewIssue        *bool   `json:"alert_on_new_issue,omitempty"`
	AlertOnRegression      *bool   `json:"alert_on_regression,omitempty"`
	AlertOnUnmute          *bool   `json:"alert_on_unmute,omitempty"`
	RetentionMaxEventCount *int    `json:"retention_max_event_count,omitempty"`
}

// UpdateProject partially updates a project and returns the resulting representation.
func (c *Client) UpdateProject(ctx context.Context, id ProjectID, in UpdateProjectRequest) (*Project, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out Project
	if err := c.do(ctx, http.MethodPatch, projectsPath+id.String()+"/", bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Deliberately no DeleteProject: Bugsink 2.5 does not document a delete
// endpoint for projects. See
// openspec/changes/acceptance-spike/specs/bugsink-api-client/spec.md,
// "No delete operation exposed."
