package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewSlaTargetResource declares `pathly_sla_target`.
func NewSlaTargetResource() resource.Resource { return &slaTargetResource{} }

type slaTargetResource struct {
	client *client.Client
}

type slaTargetModel struct {
	ID                 types.String  `tfsdk:"id"`
	ScenarioID         types.String  `tfsdk:"scenario_id"`
	Name               types.String  `tfsdk:"name"`
	ObjectivePct       types.Float64 `tfsdk:"objective_pct"`
	WindowDays         types.Int64   `tfsdk:"window_days"`
	ExcludeMaintenance types.Bool    `tfsdk:"exclude_maintenance"`
	WarnAtBudgetRatio  types.Float64 `tfsdk:"warn_at_budget_ratio"`
	Enabled            types.Bool    `tfsdk:"enabled"`
}

func (r *slaTargetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sla_target"
}

func (r *slaTargetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An SLA target, measured over a rolling window with an error budget.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier assigned by Pathly.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"scenario_id": schema.StringAttribute{
				Optional:    true,
				Description: "Measured scenario. When absent, the target applies to the whole organization.",
				// A target is identified by its scenario: moving it would amount
				// to measuring something else under the same identifier.
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			// The optional and computed attributes keep the value returned by
			// the API when the configuration leaves them empty. Otherwise
			// Terraform declares them unknown as soon as anything else changes,
			// and the plan announces moves that are not ones.
			"name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Label displayed in reports.",
				Validators:    []validator.String{stringvalidator.LengthAtMost(120)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"objective_pct": schema.Float64Attribute{
				Required:    true,
				Description: "Availability aimed for, from 50 to 100. For example 99.9.",
				Validators:  []validator.Float64{float64validator.Between(50, 100)},
			},
			"window_days": schema.Int64Attribute{
				Required:    true,
				Description: "Rolling measurement window, in days.",
				Validators:  []validator.Int64{int64validator.Between(1, 365)},
			},
			"exclude_maintenance": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "When true, maintenance windows do not consume the error budget.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"warn_at_budget_ratio": schema.Float64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Share of the error budget consumed that triggers the warning, from 0.1 to 1.",
				Validators:    []validator.Float64{float64validator.Between(0.1, 1)},
				PlanModifiers: []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "When false, the target stays defined but no longer warns.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *slaTargetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *slaTargetResource) inputFrom(m slaTargetModel) client.SlaTargetInput {
	return client.SlaTargetInput{
		MonitorID:          strPtr(m.ScenarioID),
		Name:               strPtr(m.Name),
		ObjectivePct:       m.ObjectivePct.ValueFloat64(),
		WindowDays:         m.WindowDays.ValueInt64(),
		ExcludeMaintenance: boolPtr(m.ExcludeMaintenance),
		WarnAtBudgetRatio:  float64Ptr(m.WarnAtBudgetRatio),
		Enabled:            boolPtr(m.Enabled),
	}
}

func slaToModel(t *client.SlaTarget) slaTargetModel {
	return slaTargetModel{
		ID:                 types.StringValue(t.ID),
		ScenarioID:         stringFrom(t.MonitorID),
		Name:               stringFrom(t.Name),
		ObjectivePct:       float64From(t.ObjectivePct),
		WindowDays:         int64From(t.WindowDays),
		ExcludeMaintenance: boolFrom(t.ExcludeMaintenance),
		WarnAtBudgetRatio:  float64From(t.WarnAtBudgetRatio),
		Enabled:            boolFrom(t.Enabled),
	}
}

func (r *slaTargetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan slaTargetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	created, err := r.client.UpsertSlaTarget(ctx, r.inputFrom(plan), newIdempotencyKey("sla"))
	if err != nil {
		resp.Diagnostics.AddError("SLA target creation refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, slaToModel(created))...)
}

func (r *slaTargetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state slaTargetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	got, err := r.client.GetSlaTarget(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the SLA target", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, slaToModel(got))...)
}

func (r *slaTargetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan slaTargetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	// The API exposes an upsert: the modification goes through the same call,
	// with the same targeted scenario. No identifier to pass along.
	updated, err := r.client.UpsertSlaTarget(ctx, r.inputFrom(plan), "")
	if err != nil {
		resp.Diagnostics.AddError("SLA target update refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, slaToModel(updated))...)
}

func (r *slaTargetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state slaTargetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	err := r.client.DeleteSlaTarget(ctx, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("SLA target deletion refused", err.Error())
	}
}

func (r *slaTargetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
