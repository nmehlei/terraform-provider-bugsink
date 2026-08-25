// Package provider is the Terraform Plugin Framework provider for Bugsink.
// It wraps the standalone src/bugsink client; it does not talk to Bugsink
// directly itself.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nmehlei/terraform-provider-bugsink/src/bugsink"
)

var _ provider.Provider = &BugsinkProvider{}

// New returns a provider.Provider factory, as required by
// providerserver.Serve. version is injected at build time (see src/main.go).
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BugsinkProvider{version: version}
	}
}

type BugsinkProvider struct {
	version string
}

type bugsinkProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func (p *BugsinkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bugsink"
	resp.Version = p.version
}

func (p *BugsinkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages resources on a Bugsink instance's canonical API (`/api/canonical/0/`).\n\n" +
			"~> `bugsink_team` and `bugsink_project` cannot currently be deleted through the API: " +
			"`terraform destroy` on them fails by design and requires `terraform state rm` to detach. " +
			"See each resource's documentation before adopting them.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Bugsink instance, e.g. `https://bugsink.example.com`.",
				Required:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "Bearer token for the Bugsink API. Falls back to the `BUGSINK_TOKEN` " +
					"environment variable when unset; configuration fails if neither is present.",
				Optional:  true,
				Sensitive: true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Allow a plain-HTTP endpoint. Only for local development against a " +
					"disposable instance - never set this against a production endpoint.",
				Optional: true,
			},
		},
	}
}

func (p *BugsinkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data bugsinkProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Endpoint.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing Bugsink Endpoint",
			"The endpoint attribute is required, e.g. https://bugsink.example.com.",
		)
		return
	}

	token := data.Token.ValueString()
	if token == "" {
		token = os.Getenv("BUGSINK_TOKEN")
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Bugsink API Token",
			"Set the token attribute in the provider configuration, or the BUGSINK_TOKEN environment variable.",
		)
		return
	}

	client, err := bugsink.New(ctx, bugsink.Config{
		Endpoint:      data.Endpoint.ValueString(),
		Token:         token,
		AllowInsecure: data.Insecure.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Configure Bugsink Client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *BugsinkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTeamResource,
		NewProjectResource,
	}
}

func (p *BugsinkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
