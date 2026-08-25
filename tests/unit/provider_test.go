package bugsink_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	bugsinkprovider "github.com/nmehlei/terraform-provider-bugsink/src/provider"
)

func providerSchema(t *testing.T) fwprovider.SchemaResponse {
	t.Helper()
	p := bugsinkprovider.New("test")()
	var resp fwprovider.SchemaResponse
	p.Schema(context.Background(), fwprovider.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp
}

func providerConfig(schemaResp fwprovider.SchemaResponse, endpoint string, token *string) tfsdk.Config {
	tfType := schemaResp.Schema.Type().TerraformType(context.Background())

	tokenVal := tftypes.NewValue(tftypes.String, nil)
	if token != nil {
		tokenVal = tftypes.NewValue(tftypes.String, *token)
	}

	raw := tftypes.NewValue(tfType, map[string]tftypes.Value{
		"endpoint": tftypes.NewValue(tftypes.String, endpoint),
		"token":    tokenVal,
		"insecure": tftypes.NewValue(tftypes.Bool, true),
	})

	return tfsdk.Config{Raw: raw, Schema: schemaResp.Schema}
}

// 5.1 Configure fails with an actionable diagnostic when token and
// BUGSINK_TOKEN are both absent.
func TestProviderConfigureFailsWithNoTokenAnywhere(t *testing.T) {
	t.Setenv("BUGSINK_TOKEN", "")

	schemaResp := providerSchema(t)
	p := bugsinkprovider.New("test")()

	req := fwprovider.ConfigureRequest{Config: providerConfig(schemaResp, "https://bugsink.example.invalid", nil)}
	var resp fwprovider.ConfigureResponse
	p.Configure(context.Background(), req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a configuration error, got none")
	}
	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Token") || strings.Contains(d.Detail(), "BUGSINK_TOKEN") {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics don't mention the missing token: %v", resp.Diagnostics)
	}
}

// 5.2 Configure falls back to BUGSINK_TOKEN when token is unset in config.
func TestProviderConfigureFallsBackToEnvToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/canonical/0/schema/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"info": map[string]string{"version": "2.5.0"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("BUGSINK_TOKEN", "token-from-environment")

	schemaResp := providerSchema(t)
	p := bugsinkprovider.New("test")()

	req := fwprovider.ConfigureRequest{Config: providerConfig(schemaResp, srv.URL, nil)}
	var resp fwprovider.ConfigureResponse
	p.Configure(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure() diagnostics: %v", resp.Diagnostics)
	}
	if resp.ResourceData == nil {
		t.Fatal("expected ResourceData (the *bugsink.Client) to be set")
	}
}

// 5.3 Delete on both resources fails with the terraform state rm guidance
// and (per the Plugin Framework's own contract) an error-level diagnostic
// means the framework will not remove the resource from state.
func TestResourceDeleteDiagnosticsExplainStateRm(t *testing.T) {
	t.Run("team", func(t *testing.T) {
		r := bugsinkprovider.NewTeamResource()
		var resp fwresource.DeleteResponse
		r.Delete(context.Background(), fwresource.DeleteRequest{}, &resp)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected Delete to return an error diagnostic")
		}
		assertMentionsStateRm(t, resp.Diagnostics)
	})

	t.Run("project", func(t *testing.T) {
		r := bugsinkprovider.NewProjectResource()
		var resp fwresource.DeleteResponse
		r.Delete(context.Background(), fwresource.DeleteRequest{}, &resp)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected Delete to return an error diagnostic")
		}
		assertMentionsStateRm(t, resp.Diagnostics)
	})
}

func assertMentionsStateRm(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	for _, d := range diags.Errors() {
		if strings.Contains(d.Detail(), "terraform state rm") {
			return
		}
	}
	t.Fatalf("no diagnostic mentions `terraform state rm`: %v", diags)
}

// 5.4 bugsink_project's team attribute forces replacement on change.
func TestProjectTeamAttributeRequiresReplace(t *testing.T) {
	r := bugsinkprovider.NewProjectResource()
	var resp fwresource.SchemaResponse
	r.Schema(context.Background(), fwresource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}

	teamAttr, ok := resp.Schema.Attributes["team"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("team attribute is %T, want schema.StringAttribute", resp.Schema.Attributes["team"])
	}
	if len(teamAttr.PlanModifiers) == 0 {
		t.Fatal("team attribute has no plan modifiers, want a RequiresReplace modifier")
	}

	found := false
	for _, m := range teamAttr.PlanModifiers {
		if strings.Contains(m.Description(context.Background()), "destroy and recreate") {
			found = true
		}
	}
	if !found {
		t.Fatal("team attribute's plan modifiers don't include RequiresReplace")
	}
}
