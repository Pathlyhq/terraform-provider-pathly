package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewMaintenanceWindowResource declares `pathly_maintenance_window`.
func NewMaintenanceWindowResource() resource.Resource { return &maintenanceWindowResource{} }

type maintenanceWindowResource struct {
	client *client.Client
}

type maintenanceWindowModel struct {
	ID          types.String `tfsdk:"id"`
	ScenarioID  types.String `tfsdk:"scenario_id"`
	StartsAt    types.String `tfsdk:"starts_at"`
	EndsAt      types.String `tfsdk:"ends_at"`
	Reason      types.String `tfsdk:"reason"`
	Weekday     types.Int64  `tfsdk:"weekday"`
	StartMinute types.Int64  `tfsdk:"start_minute"`
	DurationMin types.Int64  `tfsdk:"duration_min"`
}

func (r *maintenanceWindowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

func (r *maintenanceWindowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Every attribute forces replacement: the API cannot modify a window, and
	// simulating a modification through an implicit destruction would hide the
	// fact that a range of silence disappears for a few seconds.
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	// For a weekly window, `starts_at` and `ends_at` are computed by the API.
	// Without keeping the known value, they would become unknown on the
	// slightest change of pattern, and the plan would announce a replacement
	// whose displayed cause would be wrong.
	replaceOrKeep := []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
		stringplanmodifier.RequiresReplace(),
	}
	resp.Schema = schema.Schema{
		Description: "A maintenance window, one-off or weekly. The failures that fall inside it are qualified instead of alerting.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier assigned by Pathly.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"scenario_id": schema.StringAttribute{
				Optional:      true,
				Description:   "Targeted scenario. When absent, the window covers the whole organization.",
				PlanModifiers: replace,
			},
			"starts_at": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Start, in ISO 8601 format. Required for a one-off window.",
				PlanModifiers: replaceOrKeep,
			},
			"ends_at": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "End, in ISO 8601 format. Required for a one-off window, 90 days after the start at most.",
				PlanModifiers: replaceOrKeep,
			},
			"reason": schema.StringAttribute{
				Optional:      true,
				Description:   "Reason, displayed in the console and on the status page.",
				Validators:    []validator.String{stringvalidator.LengthAtMost(500)},
				PlanModifiers: replace,
			},
			"weekday": schema.Int64Attribute{
				Optional:      true,
				Description:   "1 for Monday, 7 for Sunday. When present, the window is weekly and `starts_at` / `ends_at` are ignored.",
				Validators:    []validator.Int64{int64validator.Between(1, 7)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"start_minute": schema.Int64Attribute{
				Optional:      true,
				Description:   "Start time in minutes since midnight UTC. Required for a weekly window.",
				Validators:    []validator.Int64{int64validator.Between(0, 1439)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"duration_min": schema.Int64Attribute{
				Optional:      true,
				Description:   "Duration in minutes, from 15 to 1440. Required for a weekly window.",
				Validators:    []validator.Int64{int64validator.Between(15, 1440)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *maintenanceWindowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func maintenanceToModel(w *client.MaintenanceWindow) maintenanceWindowModel {
	return maintenanceWindowModel{
		ID:          types.StringValue(w.ID),
		ScenarioID:  stringFrom(w.MonitorID),
		StartsAt:    stringFrom(w.StartsAt),
		EndsAt:      stringFrom(w.EndsAt),
		Reason:      stringFrom(w.Reason),
		Weekday:     int64From(w.Weekday),
		StartMinute: int64From(w.StartMinute),
		DurationMin: int64From(w.DurationMin),
	}
}

func (r *maintenanceWindowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan maintenanceWindowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	weekly := !plan.Weekday.IsNull() && !plan.Weekday.IsUnknown()
	if weekly && (plan.StartMinute.IsNull() || plan.DurationMin.IsNull()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("start_minute"),
			"Incomplete weekly window",
			"With `weekday`, set `start_minute` and `duration_min`.",
		)
		return
	}
	if !weekly && (plan.StartsAt.IsNull() || plan.EndsAt.IsNull()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("starts_at"),
			"Incomplete one-off window",
			"Without `weekday`, set `starts_at` and `ends_at`. For a recurring range, prefer `weekday`.",
		)
		return
	}

	created, err := r.client.CreateMaintenanceWindow(ctx, client.MaintenanceWindowInput{
		MonitorID:   strPtr(plan.ScenarioID),
		StartsAt:    strPtr(plan.StartsAt),
		EndsAt:      strPtr(plan.EndsAt),
		Reason:      strPtr(plan.Reason),
		Weekday:     int64Ptr(plan.Weekday),
		StartMinute: int64Ptr(plan.StartMinute),
		DurationMin: int64Ptr(plan.DurationMin),
	}, newIdempotencyKey("maintenance"))
	if err != nil {
		resp.Diagnostics.AddError("Maintenance window creation refused", err.Error())
		return
	}

	state := maintenanceToModel(created)
	// The reason and the scenario are not read back: the API returns them, but
	// keeping the value of the plan avoids a difference if it was normalized.
	state.Reason = plan.Reason
	if plan.ScenarioID.IsNull() {
		state.ScenarioID = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *maintenanceWindowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state maintenanceWindowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	got, err := r.client.GetMaintenanceWindow(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the maintenance window", err.Error())
		return
	}
	fresh := maintenanceToModel(got)
	if state.Reason.IsNull() && fresh.Reason.ValueString() == "" {
		fresh.Reason = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, fresh)...)
}

// Update does not exist on the API side: every attribute forces replacement,
// and the framework will never call this method. It stays explicit so that
// adding a modifiable attribute does not go unnoticed.
func (r *maintenanceWindowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Maintenance window cannot be modified",
		"The API cannot modify a window. Destroy and recreate the resource.",
	)
}

func (r *maintenanceWindowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state maintenanceWindowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	err := r.client.DeleteMaintenanceWindow(ctx, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Maintenance window deletion refused", err.Error())
	}
}

func (r *maintenanceWindowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
