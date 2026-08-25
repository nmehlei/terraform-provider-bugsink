package acceptance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"

	"github.com/nmehlei/terraform-provider-bugsink/src/bugsink"
)

const providerRequiredProvidersBlock = `
terraform {
  required_providers {
    bugsink = {
      source = "nmehlei/bugsink"
    }
  }
}
`

func providerBlock(token string) string {
	return fmt.Sprintf(`
provider "bugsink" {
  endpoint = %q
  token    = %q
  insecure = true
}
`, defaultEndpoint, token)
}

func writeConfig(t *testing.T, dir, config string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(config), 0o600); err != nil {
		t.Fatalf("writing main.tf: %v", err)
	}
}

// 6.1 + 6.2: bugsink_team create, in-place update, import, drift detection,
// and the hard-fail delete / state-rm flow, against the real pinned
// instance via the real terraform CLI.
func TestAcceptance_TeamResourceLifecycle(t *testing.T) {
	requireAcceptanceEnabled(t)
	ctx := context.Background()
	token := acceptanceToken(t)
	runID := time.Now().UnixNano()
	teamName := fmt.Sprintf("acceptance-tf-team-%d", runID)
	renamedName := teamName + "-renamed"

	workingDir := t.TempDir()
	tf := newTerraformCLI(t, workingDir)

	writeConfig(t, workingDir, providerRequiredProvidersBlock+providerBlock(token)+fmt.Sprintf(`
resource "bugsink_team" "test" {
  name = %q
}
`, teamName))

	// terraform init is deliberately skipped: it actively warns against
	// itself when dev_overrides are in effect (see tf_helpers.go), and
	// isn't needed for a single-provider config with the default local
	// backend.
	if err := tf.Apply(ctx); err != nil {
		t.Fatalf("terraform apply (create): %v", err)
	}

	state, err := tf.Show(ctx)
	if err != nil {
		t.Fatalf("terraform show: %v", err)
	}
	teamID := resourceAttribute(t, state, "bugsink_team.test", "id")
	if resourceAttribute(t, state, "bugsink_team.test", "name") != teamName {
		t.Fatalf("created team name = %q, want %q", resourceAttribute(t, state, "bugsink_team.test", "name"), teamName)
	}

	// In-place update: rename.
	writeConfig(t, workingDir, providerRequiredProvidersBlock+providerBlock(token)+fmt.Sprintf(`
resource "bugsink_team" "test" {
  name = %q
}
`, renamedName))
	if err := tf.Apply(ctx); err != nil {
		t.Fatalf("terraform apply (rename): %v", err)
	}

	// Applying again with the same config should be a no-op (no spurious diff).
	hasChanges, err := tf.Plan(ctx)
	if err != nil {
		t.Fatalf("terraform plan (no-op check): %v", err)
	}
	if hasChanges {
		t.Fatal("expected no changes on repeat plan after in-place update, got a diff")
	}

	// Import into a separate working directory/state.
	importDir := t.TempDir()
	importTF := newTerraformCLI(t, importDir)
	writeConfig(t, importDir, providerRequiredProvidersBlock+providerBlock(token)+`
resource "bugsink_team" "imported" {
  name = "placeholder-overwritten-by-import"
}
`)
	if err := importTF.Import(ctx, "bugsink_team.imported", teamID); err != nil {
		t.Fatalf("terraform import: %v", err)
	}
	importedState, err := importTF.Show(ctx)
	if err != nil {
		t.Fatalf("terraform show (import dir): %v", err)
	}
	if got := resourceAttribute(t, importedState, "bugsink_team.imported", "name"); got != renamedName {
		t.Fatalf("imported team name = %q, want %q", got, renamedName)
	}

	// Drift detection: rename out-of-band via the raw client, expect the
	// next plan (against the unchanged declared config) to show a diff.
	client, err := bugsink.New(ctx, bugsink.Config{Endpoint: defaultEndpoint, Token: token, AllowInsecure: true})
	if err != nil {
		t.Fatalf("bugsink.New() error = %v", err)
	}
	driftedName := renamedName + "-drifted"
	if _, err := client.UpdateTeam(ctx, teamID, bugsink.UpdateTeamRequest{Name: &driftedName}); err != nil {
		t.Fatalf("out-of-band UpdateTeam() error = %v", err)
	}

	hasChanges, err = tf.Plan(ctx)
	if err != nil {
		t.Fatalf("terraform plan (drift check): %v", err)
	}
	if !hasChanges {
		t.Fatal("expected a plan diff after out-of-band rename, got none (drift not detected)")
	}

	// Restore what Terraform's config expects, so destroy below operates on
	// a resource whose state matches the declared name (not load-bearing
	// for the destroy-fails assertion, just avoids an unrelated diff
	// muddying the error output).
	if _, err := client.UpdateTeam(ctx, teamID, bugsink.UpdateTeamRequest{Name: &renamedName}); err != nil {
		t.Fatalf("restoring team name out-of-band: %v", err)
	}

	// Destroy must fail loudly, not silently succeed.
	destroyErr := tf.Destroy(ctx)
	if destroyErr == nil {
		t.Fatal("expected terraform destroy to fail for bugsink_team, it succeeded")
	}
	if !strings.Contains(destroyErr.Error(), "terraform state rm") {
		t.Fatalf("destroy error doesn't mention `terraform state rm`: %v", destroyErr)
	}
	if _, err := client.GetTeam(ctx, teamID); err != nil {
		t.Fatalf("team should still exist remotely after failed destroy, GetTeam() error = %v", err)
	}

	// state rm must succeed and must not touch the remote object.
	if err := tf.StateRm(ctx, "bugsink_team.test"); err != nil {
		t.Fatalf("terraform state rm: %v", err)
	}
	if _, err := client.GetTeam(ctx, teamID); err != nil {
		t.Fatalf("team should still exist remotely after state rm, GetTeam() error = %v", err)
	}
}

// 6.3, 6.4, 6.5, 6.6: bugsink_project create (dsn populated post-apply),
// in-place update, team-change forcing replacement, drift detection, and
// the hard-fail delete / state-rm flow.
func TestAcceptance_ProjectResourceLifecycle(t *testing.T) {
	requireAcceptanceEnabled(t)
	ctx := context.Background()
	token := acceptanceToken(t)
	runID := time.Now().UnixNano()

	client, err := bugsink.New(ctx, bugsink.Config{Endpoint: defaultEndpoint, Token: token, AllowInsecure: true})
	if err != nil {
		t.Fatalf("bugsink.New() error = %v", err)
	}

	// Two teams: the project starts under teamA and moves to teamB to
	// exercise RequiresReplace. Created via the raw client - not what's
	// under test here.
	teamA, err := client.CreateTeam(ctx, bugsink.CreateTeamRequest{Name: fmt.Sprintf("acceptance-tf-project-team-a-%d", runID)})
	if err != nil {
		t.Fatalf("setup CreateTeam(teamA) error = %v", err)
	}
	teamB, err := client.CreateTeam(ctx, bugsink.CreateTeamRequest{Name: fmt.Sprintf("acceptance-tf-project-team-b-%d", runID)})
	if err != nil {
		t.Fatalf("setup CreateTeam(teamB) error = %v", err)
	}

	projectName := fmt.Sprintf("acceptance-tf-project-%d", runID)

	workingDir := t.TempDir()
	tf := newTerraformCLI(t, workingDir)

	configFor := func(team bugsink.Team, retention int) string {
		return providerRequiredProvidersBlock + providerBlock(token) + fmt.Sprintf(`
resource "bugsink_project" "test" {
  team                      = %q
  name                      = %q
  retention_max_event_count = %d
}
`, team.ID, projectName, retention)
	}

	writeConfig(t, workingDir, configFor(*teamA, 5000))
	if err := tf.Apply(ctx); err != nil {
		t.Fatalf("terraform apply (create): %v", err)
	}

	state, err := tf.Show(ctx)
	if err != nil {
		t.Fatalf("terraform show: %v", err)
	}
	if dsn := resourceAttribute(t, state, "bugsink_project.test", "dsn"); dsn == "" {
		t.Fatal("dsn is empty after apply, want it populated via the read-after-write")
	}
	projectID := resourceAttribute(t, state, "bugsink_project.test", "id")

	// In-place update of a non-team attribute must not force replacement.
	writeConfig(t, workingDir, configFor(*teamA, 9000))
	if err := tf.Apply(ctx); err != nil {
		t.Fatalf("terraform apply (update retention): %v", err)
	}
	stateAfterUpdate, err := tf.Show(ctx)
	if err != nil {
		t.Fatalf("terraform show (after update): %v", err)
	}
	if got := resourceAttribute(t, stateAfterUpdate, "bugsink_project.test", "id"); got != projectID {
		t.Fatalf("project id changed after an in-place attribute update (%s -> %s); it should not have been replaced", projectID, got)
	}

	// Changing team must force replacement.
	writeConfig(t, workingDir, configFor(*teamB, 9000))
	planFile := filepath.Join(workingDir, "team-change.tfplan")
	hasChanges, err := tf.Plan(ctx, tfexec.Out(planFile))
	if err != nil {
		t.Fatalf("terraform plan (team change): %v", err)
	}
	if !hasChanges {
		t.Fatal("expected a plan diff after changing team, got none")
	}
	plan, err := tf.ShowPlanFile(ctx, planFile)
	if err != nil {
		t.Fatalf("terraform show plan file: %v", err)
	}
	if !planShowsReplace(plan, "bugsink_project.test") {
		t.Fatal("expected changing team to force replacement (delete+create), plan did not show that")
	}
	// Don't actually apply the replacement: the interesting assertion is
	// the plan shape, and replacing here would just create an orphaned
	// project this test would then need to separately track and clean up.

	// Drift detection: change retention out-of-band, expect the next plan
	// (against the unchanged declared config, still teamA/9000) to show a diff.
	writeConfig(t, workingDir, configFor(*teamA, 9000))
	driftedRetention := 1234
	if _, err := client.UpdateProject(ctx, bugsink.ProjectID(projectID), bugsink.UpdateProjectRequest{RetentionMaxEventCount: &driftedRetention}); err != nil {
		t.Fatalf("out-of-band UpdateProject() error = %v", err)
	}
	hasChanges, err = tf.Plan(ctx)
	if err != nil {
		t.Fatalf("terraform plan (drift check): %v", err)
	}
	if !hasChanges {
		t.Fatal("expected a plan diff after out-of-band retention change, got none (drift not detected)")
	}
	restoredRetention := 9000
	if _, err := client.UpdateProject(ctx, bugsink.ProjectID(projectID), bugsink.UpdateProjectRequest{RetentionMaxEventCount: &restoredRetention}); err != nil {
		t.Fatalf("restoring retention out-of-band: %v", err)
	}

	// Destroy must fail loudly, not silently succeed.
	destroyErr := tf.Destroy(ctx)
	if destroyErr == nil {
		t.Fatal("expected terraform destroy to fail for bugsink_project, it succeeded")
	}
	if !strings.Contains(destroyErr.Error(), "terraform state rm") {
		t.Fatalf("destroy error doesn't mention `terraform state rm`: %v", destroyErr)
	}
	if _, err := client.GetProject(ctx, bugsink.ProjectID(projectID)); err != nil {
		t.Fatalf("project should still exist remotely after failed destroy, GetProject() error = %v", err)
	}

	if err := tf.StateRm(ctx, "bugsink_project.test"); err != nil {
		t.Fatalf("terraform state rm: %v", err)
	}
	if _, err := client.GetProject(ctx, bugsink.ProjectID(projectID)); err != nil {
		t.Fatalf("project should still exist remotely after state rm, GetProject() error = %v", err)
	}
}
