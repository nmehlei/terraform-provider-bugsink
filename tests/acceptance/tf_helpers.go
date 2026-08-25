package acceptance

// Provider-level acceptance tests drive a real `terraform` CLI via
// terraform-exec instead of terraform-plugin-testing's resource.Test
// harness. resource.Test always attempts an automatic final `destroy` and
// fails the test if that destroy errors - which is exactly what
// bugsink_team/bugsink_project's Delete is supposed to do by design (see
// AGENTS.md's "Delete-endpoint policy"). tfexec gives full manual control
// over apply/plan/destroy/state-rm, so the "destroy fails, state rm
// detaches" flow can be exercised and asserted on directly instead of
// fighting the harness's own cleanup assumptions.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// buildProviderBinary compiles the provider under test into a fresh temp
// directory and returns that directory, for use with Terraform's
// dev_overrides CLI config (see writeDevOverridesConfig).
func buildProviderBinary(t *testing.T) string {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "terraform-provider-bugsink")

	cmd := exec.Command("go", "build", "-o", binPath, "./src")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building provider binary: %v\n%s", err, out)
	}

	return binDir
}

// writeDevOverridesConfig writes a Terraform CLI config file pointing the
// "nmehlei/bugsink" provider source at binDir, bypassing the Registry and
// any version/checksum requirements entirely - the standard mechanism for
// testing a locally built provider binary against the real terraform CLI.
func writeDevOverridesConfig(t *testing.T, binDir string) string {
	t.Helper()

	cfg := fmt.Sprintf(`
provider_installation {
  dev_overrides {
    "registry.terraform.io/nmehlei/bugsink" = %q
  }
  direct {}
}
`, binDir)

	path := filepath.Join(t.TempDir(), "cli-config.tfrc")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("writing CLI config: %v", err)
	}
	return path
}

// newTerraformCLI returns a terraform-exec handle, rooted at workingDir,
// wired to use the locally built provider binary via dev_overrides.
func newTerraformCLI(t *testing.T, workingDir string) *tfexec.Terraform {
	t.Helper()

	tfBin, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform CLI not found in PATH, skipping provider acceptance test")
	}

	binDir := buildProviderBinary(t)
	cliConfig := writeDevOverridesConfig(t, binDir)

	tf, err := tfexec.NewTerraform(workingDir, tfBin)
	if err != nil {
		t.Fatalf("initializing terraform-exec: %v", err)
	}
	if err := tf.SetEnv(map[string]string{
		"TF_CLI_CONFIG_FILE": cliConfig,
		"PATH":               os.Getenv("PATH"),
	}); err != nil {
		t.Fatalf("setting terraform-exec environment: %v", err)
	}
	return tf
}

// planShowsReplace reports whether the plan replaces (destroys + creates)
// the named resource address.
func planShowsReplace(plan *tfjson.Plan, address string) bool {
	for _, rc := range plan.ResourceChanges {
		if rc.Address == address {
			return rc.Change.Actions.Replace()
		}
	}
	return false
}

// resourceAttribute returns a top-level string attribute of the named
// resource from a terraform show/state result, failing the test if either
// is missing.
func resourceAttribute(t *testing.T, state *tfjson.State, address, attr string) string {
	t.Helper()

	if state.Values == nil || state.Values.RootModule == nil {
		t.Fatalf("state has no root module (looking for %s.%s)", address, attr)
	}
	for _, r := range state.Values.RootModule.Resources {
		if r.Address != address {
			continue
		}
		v, ok := r.AttributeValues[attr]
		if !ok {
			t.Fatalf("resource %s has no %q attribute", address, attr)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("resource %s attribute %q is %T, want string", address, attr, v)
		}
		return s
	}
	t.Fatalf("resource %s not found in state", address)
	return ""
}
