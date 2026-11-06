package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewSpaceResource declares `pathly_space`.
//
// A destroy archives the Space: that is what the API does on DELETE, and it is
// the right outcome for monitoring — the scenarios leave the Space and keep
// running, instead of a `terraform destroy` silently stopping the watch on a
// client's sites.
func NewSpaceResource() resource.Resource { return &spaceResource{} }

type spaceResource struct {
	client *client.Client
}

type spaceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Kind         types.String `tfsdk:"kind"`
	Slug         types.String `tfsdk:"slug"`
	StatusPublic types.Bool   `tfsdk:"status_public"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (r *spaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_space"
}

func (r *spaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		Description: "A Space: the scenarios of one client, one brand or one application.",
		MarkdownDescription: "A Space groups the scenarios of one client, one brand or one application.\n\n" +
			"Destroying it **archives** it: its scenarios leave it and keep running, nothing is deleted. " +
			"The API answers 404 while Spaces are not open for the organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier assigned by Pathly.",
				PlanModifiers: keep,
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the Space, unique among the active Spaces of the organization.",
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 80)},
			},
			"kind": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "`client`, `site` or `application`. When absent, Pathly derives it from the " +
					"usage profile of the organization.",
				Validators:    []validator.String{stringvalidator.OneOf("client", "site", "application")},
				PlanModifiers: keep,
			},
			"slug": schema.StringAttribute{
				Computed: true,
				Description: "Address of the Space, derived from its name at creation and kept on rename. " +
					"Its status page lives under the address of the organization page.",
				PlanModifiers: keep,
			},
			"status_public": schema.BoolAttribute{
				Computed: true,
				Description: "Whether the status page of the Space is published. " +
					"Managed by `pathly_status_page`, read only here.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "Creation date.",
				PlanModifiers: keep,
			},
		},
	}
}

func (r *spaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func spaceToModel(s *client.Space) spaceModel {
	return spaceModel{
		ID:           types.StringValue(s.ID),
		Name:         types.StringValue(s.Name),
		Kind:         stringFrom(s.Kind),
		Slug:         stringFrom(s.Slug),
		StatusPublic: boolFrom(s.StatusPublic),
		CreatedAt:    stringFrom(s.CreatedAt),
	}
}

func (r *spaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan spaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	created, err := r.client.CreateSpace(ctx, client.SpaceInput{Name: strPtr(plan.Name), Kind: strPtr(plan.Kind)})
	if err != nil {
		resp.Diagnostics.AddError("Space creation refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, spaceToModel(created))...)
}

func (r *spaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state spaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	got, err := r.client.GetSpace(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		// Archived from the console, or Spaces closed for the organization:
		// either way the Space is gone for this configuration.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the Space", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, spaceToModel(got))...)
}

func (r *spaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan spaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state spaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	updated, err := r.client.UpdateSpace(ctx, state.ID.ValueString(), client.SpaceInput{
		Name: strPtr(plan.Name),
		Kind: strPtr(plan.Kind),
	})
	if client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Space vanished",
			fmt.Sprintf("Space %s is archived or no longer reachable. Run `terraform apply` again to recreate it.", state.ID.ValueString()),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Space update refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, spaceToModel(updated))...)
}

func (r *spaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state spaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	err := r.client.ArchiveSpace(ctx, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Space archiving refused", err.Error())
	}
}

func (r *spaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
