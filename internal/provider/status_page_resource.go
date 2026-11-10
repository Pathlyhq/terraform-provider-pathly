package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// statusSlugPattern is the address format the API accepts.
var statusSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// NewStatusPageResource declares `pathly_status_page`.
//
// The API has no deletion for a status page: the page of the organization and
// those of the Spaces exist as long as their owner does. A destroy therefore
// unpublishes the page and says so, which is the closest honest outcome — the
// public page stops answering, and its address stays reserved, so the badges
// already pasted on client sites come back the day it is published again.
func NewStatusPageResource() resource.Resource { return &statusPageResource{} }

type statusPageResource struct {
	client *client.Client
}

type statusPageResourceModel struct {
	ID       types.String `tfsdk:"id"`
	SpaceID  types.String `tfsdk:"space_id"`
	Slug     types.String `tfsdk:"slug"`
	Public   types.Bool   `tfsdk:"public"`
	Kind     types.String `tfsdk:"kind"`
	Name     types.String `tfsdk:"name"`
	URL      types.String `tfsdk:"url"`
	BadgeURL types.String `tfsdk:"badge_url"`
}

func (r *statusPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (r *statusPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	followsSlug := []planmodifier.String{keepUnlessSlugChanges{}}
	resp.Schema = schema.Schema{
		Description: "The status page of the organization, or that of a Space, with its badge.",
		MarkdownDescription: "The status page of the organization (without `space_id`) or that of a Space " +
			"(with `space_id`), with the address of its SVG badge.\n\n" +
			"The API has no deletion for a status page: **destroying the resource unpublishes the page**, " +
			"keeps its address, and leaves it to the console.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				Description: "Identifier of the page: that of the organization for its page, " +
					"that of the Space for a Space page.",
				PlanModifiers: keep,
			},
			"space_id": schema.StringAttribute{
				Optional: true,
				Description: "Space whose page this is. Absent for the page of the organization. " +
					"A key limited to a Space can only manage the page of its Space.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"slug": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Address of the organization page, proposed from its name when absent. " +
					"Changing it breaks the links and badges already posted. " +
					"A Space page takes the address of its Space and refuses this attribute.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(statusSlugPattern, "lowercase letters, digits and single dashes"),
					stringvalidator.ConflictsWith(path.MatchRoot("space_id")),
				},
				PlanModifiers: keep,
			},
			"public": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Publishes the page. A new page stays private unless this is true. " +
					"Publishing exposes the scenario names and their availability.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"kind": schema.StringAttribute{
				Computed:      true,
				Description:   "`organization` or `space`.",
				PlanModifiers: keep,
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name shown on the page: that of the organization or of the Space.",
			},
			"url": schema.StringAttribute{
				Computed:      true,
				Description:   "Public address of the page.",
				PlanModifiers: followsSlug,
			},
			"badge_url": schema.StringAttribute{
				Computed: true,
				Description: "Address of the SVG badge. Add `?v=uptime` for the availability. " +
					"On the organization page, add `?monitor=<scenario id>` when it holds several scenarios.",
				PlanModifiers: followsSlug,
			},
		},
	}
}

func (r *statusPageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func statusPageToModel(p *client.StatusPageEntry) statusPageResourceModel {
	return statusPageResourceModel{
		ID:       types.StringValue(p.ID),
		SpaceID:  stringFrom(p.SpaceID),
		Slug:     types.StringValue(p.Slug),
		Public:   types.BoolValue(p.Public),
		Kind:     types.StringValue(p.Kind),
		Name:     types.StringValue(p.Name),
		URL:      types.StringValue(p.URL),
		BadgeURL: types.StringValue(p.BadgeURL),
	}
}

func (r *statusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan statusPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	// The API returns an existing page as it is rather than refusing it: a
	// first apply adopts the page the console already created.
	created, err := r.client.CreateStatusPageEntry(ctx, client.StatusPageInput{
		SpaceID: strPtr(plan.SpaceID),
		Slug:    strPtr(plan.Slug),
		Public:  boolPtr(plan.Public),
	}, newIdempotencyKey("status-page"))
	if err != nil {
		resp.Diagnostics.AddError(
			"Status page creation refused",
			err.Error()+" If the organization page already exists at another address, import it "+
				"(its identifier is `data.pathly_account.this.organization_id`) or drop `slug`.",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, statusPageToModel(created))...)
}

func (r *statusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state statusPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	got, err := r.client.GetStatusPageEntry(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		// The Space was archived, or the organization page lost its address.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the status page", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, statusPageToModel(got))...)
}

func (r *statusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan statusPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state statusPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	// `public` is always known here, kept from the state when the
	// configuration leaves it out: the API refuses an empty body. `slug` only
	// goes out when it changes, since it moves the page.
	input := client.StatusPageInput{Public: boolPtr(plan.Public)}
	if !plan.Slug.Equal(state.Slug) {
		input.Slug = strPtr(plan.Slug)
	}
	updated, err := r.client.UpdateStatusPageEntry(ctx, state.ID.ValueString(), input)
	if client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Status page vanished",
			fmt.Sprintf("Status page %s no longer exists. Run `terraform apply` again to recreate it.", state.ID.ValueString()),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Status page update refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, statusPageToModel(updated))...)
}

func (r *statusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state statusPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	private := false
	_, err := r.client.UpdateStatusPageEntry(ctx, state.ID.ValueString(), client.StatusPageInput{Public: &private})
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Status page unpublishing refused", err.Error())
		return
	}
	resp.Diagnostics.AddWarning(
		"Status page unpublished, not deleted",
		"The Pathly API has no deletion for a status page. The page is now private, its address stays "+
			"reserved, and Terraform stops tracking it.",
	)
}

func (r *statusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// keepUnlessSlugChanges keeps the known value of an address derived from the
// slug, as long as the slug does not move.
//
// Plain UseStateForUnknown would be wrong here: after a change of `slug`, the
// API returns a new address, and Terraform refuses an apply whose result
// differs from what the plan promised. Without any modifier, every change of
// `public` would announce the two addresses as unknown.
type keepUnlessSlugChanges struct{}

func (keepUnlessSlugChanges) Description(context.Context) string {
	return "Once set, the value of this attribute in state will not change, unless `slug` changes."
}

func (m keepUnlessSlugChanges) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (keepUnlessSlugChanges) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || !req.PlanValue.IsUnknown() {
		return
	}
	// The configuration rather than the plan: a slug left out of the
	// configuration is kept from the state, so it does not move either.
	var configured, current types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("slug"), &configured)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("slug"), &current)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if configured.IsNull() || configured.Equal(current) {
		resp.PlanValue = req.StateValue
	}
}
