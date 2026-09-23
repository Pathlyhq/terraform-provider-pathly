package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewSlaTargetsDataSource declares `pathly_sla_targets`.
//
// The definitions alone, without the measurement. Adopting an existing estate is
// what this is for: listing the targets created in the console gives the
// identifiers needed to import them, whereas `pathly_sla` would also drag in
// figures that change with every run.
func NewSlaTargetsDataSource() datasource.DataSource { return &slaTargetsDataSource{} }

type slaTargetsDataSource struct {
	client *client.Client
}

type slaTargetsDataSourceModel struct {
	Targets []slaTargetEntry `tfsdk:"targets"`
}

type slaTargetEntry struct {
	ID                 types.String  `tfsdk:"id"`
	ScenarioID         types.String  `tfsdk:"scenario_id"`
	Name               types.String  `tfsdk:"name"`
	ObjectivePct       types.Float64 `tfsdk:"objective_pct"`
	WindowDays         types.Int64   `tfsdk:"window_days"`
	ExcludeMaintenance types.Bool    `tfsdk:"exclude_maintenance"`
	WarnAtBudgetRatio  types.Float64 `tfsdk:"warn_at_budget_ratio"`
	Enabled            types.Bool    `tfsdk:"enabled"`
}

func (d *slaTargetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sla_targets"
}

func (d *slaTargetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Availability objectives as defined, without their measurement. See `pathly_sla` for where each one stands.",
		Attributes: map[string]schema.Attribute{
			"targets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Targets found, in the order returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                   schema.StringAttribute{Computed: true, Description: "Identifier of the target."},
						"scenario_id":          schema.StringAttribute{Computed: true, Description: "Measured scenario, empty when the target covers the organization."},
						"name":                 schema.StringAttribute{Computed: true, Description: "Label displayed in reports."},
						"objective_pct":        schema.Float64Attribute{Computed: true, Description: "Availability aimed for."},
						"window_days":          schema.Int64Attribute{Computed: true, Description: "Rolling measurement window, in days."},
						"exclude_maintenance":  schema.BoolAttribute{Computed: true, Description: "Maintenance windows do not consume the error budget."},
						"warn_at_budget_ratio": schema.Float64Attribute{Computed: true, Description: "Share of the consumed budget that triggers the warning."},
						"enabled":              schema.BoolAttribute{Computed: true, Description: "The target is active."},
					},
				},
			},
		},
	}
}

func (d *slaTargetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *slaTargetsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}
	all, err := d.client.ListSlaTargets(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the availability objectives", err.Error())
		return
	}

	var model slaTargetsDataSourceModel
	for i := range all {
		t := all[i]
		model.Targets = append(model.Targets, slaTargetEntry{
			ID:                 types.StringValue(t.ID),
			ScenarioID:         stringFrom(t.MonitorID),
			Name:               stringFrom(t.Name),
			ObjectivePct:       float64From(t.ObjectivePct),
			WindowDays:         int64From(t.WindowDays),
			ExcludeMaintenance: boolFrom(t.ExcludeMaintenance),
			WarnAtBudgetRatio:  float64From(t.WarnAtBudgetRatio),
			Enabled:            boolFrom(t.Enabled),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
