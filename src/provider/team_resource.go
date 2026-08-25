package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nmehlei/terraform-provider-bugsink/src/bugsink"
)

var (
	_ resource.Resource                = &TeamResource{}
	_ resource.ResourceWithImportState = &TeamResource{}
)

func NewTeamResource() resource.Resource {
	return &TeamResource{}
}

// TeamResource manages a Bugsink team via create/read/update. Delete is
// intentionally unsupported - see AGENTS.md's "Delete-endpoint policy"
// (amended 2026-08-24): Bugsink has no delete endpoint for teams, and
// silently removing this resource from state while it still exists
// remotely is exactly the drift Terraform resources exist to prevent.
type TeamResource struct {
	client *bugsink.Client
}

type teamResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Visibility types.String `tfsdk:"visibility"`
}

const teamDeleteUnsupportedMessage = "Bugsink has no API to delete a team (see the provider repository's " +
	"docs/upstream-proposal.md). Terraform cannot remove this team remotely. If you want Terraform to stop " +
	"tracking it, run `terraform state rm` on this resource's address - the team will remain in Bugsink."

func (r *TeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *TeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Bugsink team.\n\n" +
			"~> **`terraform destroy` fails by design.** Bugsink has no API to delete a team. Deleting this " +
			"resource always fails with a diagnostic explaining how to run `terraform state rm` instead. Do " +
			"not use this resource in any workflow that assumes destroy succeeds unconditionally (e.g. " +
			"ephemeral or CI-provisioned environments).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Team ID (UUID), assigned by Bugsink.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Team name.",
				Required:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "Team visibility (e.g. `joinable`, `discoverable`, `hidden`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TeamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*bugsink.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("expected *bugsink.Client, got: %T. Report this issue to the provider maintainers.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	team, err := r.client.CreateTeam(ctx, bugsink.CreateTeamRequest{
		Name:       plan.Name.ValueString(),
		Visibility: plan.Visibility.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Bugsink Team", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, teamModelFromAPI(team))...)
}

func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	team, err := r.client.GetTeam(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *bugsink.APIError
		if errors.As(err, &apiErr) && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Bugsink Team", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, teamModelFromAPI(team))...)
}

func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := bugsink.UpdateTeamRequest{}
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		update.Name = &name
	}
	if !plan.Visibility.Equal(state.Visibility) {
		visibility := plan.Visibility.ValueString()
		update.Visibility = &visibility
	}

	team, err := r.client.UpdateTeam(ctx, state.ID.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Bugsink Team", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, teamModelFromAPI(team))...)
}

// Delete deliberately never removes the resource from state on success:
// it always returns a diagnostics-level error, so the Plugin Framework
// leaves state untouched (see design.md - "Delete needs no special 'don't
// remove state' logic").
func (r *TeamResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Cannot Delete Bugsink Team", teamDeleteUnsupportedMessage)
}

func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func teamModelFromAPI(team *bugsink.Team) teamResourceModel {
	return teamResourceModel{
		ID:         types.StringValue(team.ID),
		Name:       types.StringValue(team.Name),
		Visibility: types.StringValue(team.Visibility),
	}
}
