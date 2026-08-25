// Package acceptance exercises the bugsink client against a real, disposable
// Bugsink 2.5 instance started by docker-compose.yml in this directory (see
// up.sh / down.sh / README.md). It never runs by default: go test ./...
// stays fast and hermetic, and CI only sets BUGSINK_ACCEPTANCE after
// bringing the environment up.
package acceptance

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nmehlei/terraform-provider-bugsink/src/bugsink"
)

const defaultEndpoint = "http://localhost:8000"

func requireAcceptanceEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("BUGSINK_ACCEPTANCE") != "1" {
		t.Skip("skipping acceptance test: set BUGSINK_ACCEPTANCE=1 after running ./up.sh")
	}
}

func acceptanceToken(t *testing.T) string {
	t.Helper()
	if tok := os.Getenv("BUGSINK_TOKEN"); tok != "" {
		return tok
	}
	data, err := os.ReadFile(".token")
	if err != nil {
		t.Fatalf("no BUGSINK_TOKEN set and could not read .token (did you run ./up.sh?): %v", err)
	}
	return strings.TrimSpace(string(data))
}

func newAcceptanceClient(t *testing.T) *bugsink.Client {
	t.Helper()
	endpoint := os.Getenv("BUGSINK_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := bugsink.New(ctx, bugsink.Config{
		Endpoint:      endpoint,
		Token:         acceptanceToken(t),
		AllowInsecure: true, // local docker-compose instance only
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

// 6.2 + 6.3: create -> retrieve -> PATCH -> retrieve for a team and a
// project, against the real pinned instance. New() succeeding already
// proves the client's version check accepted the running instance (6.3).
func TestAcceptance_TeamAndProjectLifecycle(t *testing.T) {
	requireAcceptanceEnabled(t)
	c := newAcceptanceClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Bugsink has no delete endpoint (the entire premise of this repo), so
	// re-running this test against the same instance without a full
	// down.sh/up.sh cycle must not collide with names from a previous run.
	runID := time.Now().UnixNano()
	teamName := fmt.Sprintf("acceptance-spike-team-%d", runID)
	projectName := fmt.Sprintf("acceptance-spike-project-%d", runID)

	team, err := c.CreateTeam(ctx, bugsink.CreateTeamRequest{Name: teamName})
	if err != nil {
		t.Fatalf("CreateTeam() error = %v", err)
	}
	if team.ID == "" {
		t.Fatal("CreateTeam() returned empty ID")
	}

	got, err := c.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeam() error = %v", err)
	}
	if got.Name != teamName {
		t.Fatalf("GetTeam().Name = %q, want %q", got.Name, teamName)
	}

	renamed := teamName + "-renamed"
	updated, err := c.UpdateTeam(ctx, team.ID, bugsink.UpdateTeamRequest{Name: &renamed})
	if err != nil {
		t.Fatalf("UpdateTeam() error = %v", err)
	}
	if updated.Name != renamed {
		t.Fatalf("UpdateTeam().Name = %q, want %q", updated.Name, renamed)
	}

	gotAfterUpdate, err := c.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeam() after update error = %v", err)
	}
	if gotAfterUpdate.Name != renamed {
		t.Fatalf("GetTeam() after update Name = %q, want %q", gotAfterUpdate.Name, renamed)
	}

	project, err := c.CreateProject(ctx, bugsink.CreateProjectRequest{
		Team: team.ID,
		Name: projectName,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if project.ID == "" {
		t.Fatal("CreateProject() returned empty ID")
	}

	// The create response doesn't populate computed fields like DSN/Slug
	// (confirmed against a live instance) - only retrieve/list do, which is
	// exactly why docs/api-contract.md requires a read after every
	// create/update.
	gotProject, err := c.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if gotProject.Name != projectName {
		t.Fatalf("GetProject().Name = %q, want %q", gotProject.Name, projectName)
	}
	if gotProject.DSN == "" {
		t.Fatal("GetProject() returned empty DSN")
	}

	renamedProject := projectName + "-renamed"
	updatedProject, err := c.UpdateProject(ctx, project.ID, bugsink.UpdateProjectRequest{Name: &renamedProject})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if updatedProject.Name != renamedProject {
		t.Fatalf("UpdateProject().Name = %q, want %q", updatedProject.Name, renamedProject)
	}

	gotProjectAfterUpdate, err := c.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() after update error = %v", err)
	}
	if gotProjectAfterUpdate.Name != renamedProject {
		t.Fatalf("GetProject() after update Name = %q, want %q", gotProjectAfterUpdate.Name, renamedProject)
	}
	if gotProjectAfterUpdate.Team != team.ID {
		t.Fatalf("GetProject() after update Team = %q, want unchanged %q", gotProjectAfterUpdate.Team, team.ID)
	}
}
