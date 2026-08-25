package bugsink

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

const teamsPath = "teams/"

// Team is a Bugsink team, per docs/api-contract.md's canonical fields.
type Team struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

// TeamPage is one page of a team listing, following the server's cursor
// pagination. Next is empty when there is no further page.
type TeamPage struct {
	Next    string `json:"next"`
	Results []Team `json:"results"`
}

// ListTeams fetches one page of teams. Pass an empty pageURL for the first
// page and TeamPage.Next for subsequent pages; the client never constructs
// cursor values itself.
func (c *Client) ListTeams(ctx context.Context, pageURL string) (*TeamPage, error) {
	path := pageURL
	if path == "" {
		path = teamsPath
	}
	var page TeamPage
	if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CreateTeamRequest is the writable subset of Team accepted on create.
type CreateTeamRequest struct {
	Name       string `json:"name"`
	Visibility string `json:"visibility,omitempty"`
}

// CreateTeam creates a team and returns the server's representation of it.
func (c *Client) CreateTeam(ctx context.Context, in CreateTeamRequest) (*Team, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out Team
	if err := c.do(ctx, http.MethodPost, teamsPath, bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTeam retrieves a team by ID.
func (c *Client) GetTeam(ctx context.Context, id string) (*Team, error) {
	var out Team
	if err := c.do(ctx, http.MethodGet, teamsPath+id+"/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateTeamRequest is a partial update: only non-nil fields are sent, so
// unset fields are left unchanged on the server (PATCH semantics).
type UpdateTeamRequest struct {
	Name       *string `json:"name,omitempty"`
	Visibility *string `json:"visibility,omitempty"`
}

// UpdateTeam partially updates a team and returns the resulting representation.
func (c *Client) UpdateTeam(ctx context.Context, id string, in UpdateTeamRequest) (*Team, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out Team
	if err := c.do(ctx, http.MethodPatch, teamsPath+id+"/", bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Deliberately no DeleteTeam: Bugsink 2.5 does not document a delete
// endpoint for teams. See
// openspec/changes/acceptance-spike/specs/bugsink-api-client/spec.md,
// "No delete operation exposed."
