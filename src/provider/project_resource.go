package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nmehlei/terraform-provider-bugsink/src/bugsink"
)

var (
	_ resource.Resource                = &ProjectResource{}
	_ resource.ResourceWithImportState = &ProjectResource{}
)

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource manages a Bugsink project via create/read/update. Delete
// is intentionally unsupported for the same reason as TeamResource - see
// AGENTS.md's "Delete-endpoint policy" (amended 2026-08-24).
type ProjectResource struct {
	client *bugsink.Client
}

type projectResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Team                   types.String `tfsdk:"team"`
	Name                   types.String `tfsdk:"name"`
	Visibility             types.String `tfsdk:"visibility"`
	AlertOnNewIssue        types.Bool   `tfsdk:"alert_on_new_issue"`
	AlertOnRegression      types.Bool   `tfsdk:"alert_on_regression"`
	AlertOnUnmute          types.Bool   `tfsdk:"alert_on_unmute"`
	RetentionMaxEventCount types.Int64  `tfsdk:"retention_max_event_count"`
	Slug                   types.String `tfsdk:"slug"`
	DSN                    types.String `tfsdk:"dsn"`
	GroupingMechanism      types.String `tfsdk:"grouping_mechanism"`
}

const projectDeleteUnsupportedMessage = "Bugsink has no API to delete a project (see the provider repository's " +
	"docs/upstream-proposal.md). Terraform cannot remove this project remotely. If you want Terraform to stop " +
	"tracking it, run `terraform state rm` on this resource's address - the project will remain in Bugsink."

func (r *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Bugsink project.\n\n" +
			"~> **`terraform destroy` fails by design.** Bugsink has no API to delete a project. Deleting " +
			"this resource always fails with a diagnostic explaining how to run `terraform state rm` " +
			"instead. Do not use this resource in any workflow that assumes destroy succeeds " +
			"unconditionally (e.g. ephemeral or CI-provisioned environments).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Project ID, assigned by Bugsink.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team": schema.StringAttribute{
				MarkdownDescription: "ID of the team this project belongs to. Bugsink has no API to move a " +
					"project between teams, so changing this forces replacement.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Project name.",
				Required:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "Project visibility.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"alert_on_new_issue": schema.BoolAttribute{
				MarkdownDescription: "Whether to alert on a new issue.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"alert_on_regression": schema.BoolAttribute{
				MarkdownDescription: "Whether to alert on a regression.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"alert_on_unmute": schema.BoolAttribute{
				MarkdownDescription: "Whether to alert when an issue is unmuted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"retention_max_event_count": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of events retained for this project.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "Project slug, computed by Bugsink.",
				Computed:            true,
			},
			"dsn": schema.StringAttribute{
				MarkdownDescription: "Project DSN. Not present until after the first apply completes - " +
					"Bugsink's create response omits it, so this provider reads it back immediately " +
					"afterward.",
				Computed:  true,
				Sensitive: true,
			},
			"grouping_mechanism": schema.StringAttribute{
				MarkdownDescription: "Issue grouping mechanism, computed by Bugsink.",
				Computed:            true,
			},
		},
	}
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	create := bugsink.CreateProjectRequest{
		Team: plan.Team.ValueString(),
		Name: plan.Name.ValueString(),
	}
	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		create.Visibility = plan.Visibility.ValueString()
	}
	if !plan.AlertOnNewIssue.IsNull() && !plan.AlertOnNewIssue.IsUnknown() {
		v := plan.AlertOnNewIssue.ValueBool()
		create.AlertOnNewIssue = &v
	}
	if !plan.AlertOnRegression.IsNull() && !plan.AlertOnRegression.IsUnknown() {
		v := plan.AlertOnRegression.ValueBool()
		create.AlertOnRegression = &v
	}
	if !plan.AlertOnUnmute.IsNull() && !plan.AlertOnUnmute.IsUnknown() {
		v := plan.AlertOnUnmute.ValueBool()
		create.AlertOnUnmute = &v
	}
	if !plan.RetentionMaxEventCount.IsNull() && !plan.RetentionMaxEventCount.IsUnknown() {
		v := int(plan.RetentionMaxEventCount.ValueInt64())
		create.RetentionMaxEventCount = &v
	}

	created, err := r.client.CreateProject(ctx, create)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Bugsink Project", err.Error())
		return
	}

	// slug/dsn/grouping_mechanism aren't in the create response (confirmed
	// empirically, see docs/api-contract.md) - read back immediately so
	// state is correct before apply finishes.
	model, diags := r.readProject(ctx, created.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if model == nil {
		resp.Diagnostics.AddError("Error Reading Newly Created Bugsink Project",
			"the project was created but could not be read back immediately afterward")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, diags := r.readProject(ctx, bugsink.ProjectID(state.ID.ValueString()))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if model == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := bugsink.UpdateProjectRequest{}
	if !plan.Name.Equal(state.Name) {
		v := plan.Name.ValueString()
		update.Name = &v
	}
	if !plan.Visibility.Equal(state.Visibility) {
		v := plan.Visibility.ValueString()
		update.Visibility = &v
	}
	if !plan.AlertOnNewIssue.Equal(state.AlertOnNewIssue) {
		v := plan.AlertOnNewIssue.ValueBool()
		update.AlertOnNewIssue = &v
	}
	if !plan.AlertOnRegression.Equal(state.AlertOnRegression) {
		v := plan.AlertOnRegression.ValueBool()
		update.AlertOnRegression = &v
	}
	if !plan.AlertOnUnmute.Equal(state.AlertOnUnmute) {
		v := plan.AlertOnUnmute.ValueBool()
		update.AlertOnUnmute = &v
	}
	if !plan.RetentionMaxEventCount.Equal(state.RetentionMaxEventCount) {
		v := int(plan.RetentionMaxEventCount.ValueInt64())
		update.RetentionMaxEventCount = &v
	}

	projectID := bugsink.ProjectID(state.ID.ValueString())
	if _, err := r.client.UpdateProject(ctx, projectID, update); err != nil {
		resp.Diagnostics.AddError("Error Updating Bugsink Project", err.Error())
		return
	}

	model, diags := r.readProject(ctx, projectID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if model == nil {
		resp.Diagnostics.AddError("Error Reading Bugsink Project After Update",
			"the project was updated but could not be read back immediately afterward")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Delete deliberately never removes the resource from state on success:
// it always returns a diagnostics-level error, so the Plugin Framework
// leaves state untouched (see design.md - "Delete needs no special 'don't
// remove state' logic").
func (r *ProjectResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Cannot Delete Bugsink Project", projectDeleteUnsupportedMessage)
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readProject is the shared helper Create, Read, and Update all use (see
// design.md - "Read-after-write for computed project fields"). A nil model
// with no error diagnostics means the project is genuinely gone (404);
// callers decide what that means in their own context (RemoveResource for
// Read, an explicit error for Create/Update where a fresh 404 is
// unexpected).
func (r *ProjectResource) readProject(ctx context.Context, id bugsink.ProjectID) (*projectResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	project, err := r.client.GetProject(ctx, id)
	if err != nil {
		var apiErr *bugsink.APIError
		if errors.As(err, &apiErr) && apiErr.NotFound() {
			return nil, diags
		}
		diags.AddError("Error Reading Bugsink Project", err.Error())
		return nil, diags
	}

	return projectModelFromAPI(project), diags
}

func projectModelFromAPI(project *bugsink.Project) *projectResourceModel {
	return &projectResourceModel{
		ID:                     types.StringValue(project.ID.String()),
		Team:                   types.StringValue(project.Team),
		Name:                   types.StringValue(project.Name),
		Visibility:             types.StringValue(project.Visibility),
		AlertOnNewIssue:        types.BoolValue(project.AlertOnNewIssue),
		AlertOnRegression:      types.BoolValue(project.AlertOnRegression),
		AlertOnUnmute:          types.BoolValue(project.AlertOnUnmute),
		RetentionMaxEventCount: types.Int64Value(int64(project.RetentionMaxEventCount)),
		Slug:                   types.StringValue(project.Slug),
		DSN:                    types.StringValue(project.DSN),
		GroupingMechanism:      types.StringValue(project.GroupingMechanism),
	}
}
