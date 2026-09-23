package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewSlaDataSource declares `pathly_sla`.
//
// This is the measurement, not the definition: `pathly_sla_target` declares what
// is aimed for, this reads where each target currently stands. Keeping the two
// apart matters, because a plan that depended on the measurement would change
// with every run.
func NewSlaDataSource() datasource.DataSource { return &slaDataSource{} }

type slaDataSource struct {
	client *client.Client
}

type slaDataSourceModel struct {
	FilterScenarioID types.String `tfsdk:"filter_scenario_id"`
	FilterState      types.String `tfsdk:"filter_state"`
	Targets          []slaEntry   `tfsdk:"targets"`
}

type slaEntry struct {
	ID                 types.String  `tfsdk:"id"`
	ScenarioID         types.String  `tfsdk:"scenario_id"`
	ScenarioName       types.String  `tfsdk:"scenario_name"`
	Name               types.String  `tfsdk:"name"`
	ObjectivePct       types.Float64 `tfsdk:"objective_pct"`
	WindowDays         types.Int64   `tfsdk:"window_days"`
	ExcludeMaintenance types.Bool    `tfsdk:"exclude_maintenance"`
	WarnAtBudgetRatio  types.Float64 `tfsdk:"warn_at_budget_ratio"`
	Enabled            types.Bool    `tfsdk:"enabled"`

	State                    types.String  `tfsdk:"state"`
	UptimePct                types.Float64 `tfsdk:"uptime_pct"`
	EligibleRuns             types.Int64   `tfsdk:"eligible_runs"`
	OkRuns                   types.Int64   `tfsdk:"ok_runs"`
	FailedRuns               types.Int64   `tfsdk:"failed_runs"`
	ExcludedRuns             types.Int64   `tfsdk:"excluded_runs"`
	ErrorBudgetRuns          types.Int64   `tfsdk:"error_budget_runs"`
	ErrorBudgetUsedRatio     types.Float64 `tfsdk:"error_budget_used_ratio"`
	EstimatedDowntimeMinutes types.Float64 `tfsdk:"estimated_downtime_minutes"`
}

func (d *slaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sla"
}

func (d *slaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Availability objectives and their current measurement.",
		Attributes: map[string]schema.Attribute{
			"filter_scenario_id": schema.StringAttribute{
				Optional:    true,
				Description: "Keeps only the target measuring this scenario.",
			},
			"filter_state": schema.StringAttribute{
				Optional: true,
				Description: "Keeps only the targets in this state. " +
					"Not validated against a fixed list: a state added later must not break a working configuration.",
			},
			"targets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Targets found, with their measurement flattened alongside the definition.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                   schema.StringAttribute{Computed: true, Description: "Identifier of the target."},
						"scenario_id":          schema.StringAttribute{Computed: true, Description: "Measured scenario, empty when the target covers the organization."},
						"scenario_name":        schema.StringAttribute{Computed: true, Description: "Name of that scenario."},
						"name":                 schema.StringAttribute{Computed: true, Description: "Label of the target."},
						"objective_pct":        schema.Float64Attribute{Computed: true, Description: "Availability aimed for."},
						"window_days":          schema.Int64Attribute{Computed: true, Description: "Rolling measurement window, in days."},
						"exclude_maintenance":  schema.BoolAttribute{Computed: true, Description: "Maintenance windows do not consume the budget."},
						"warn_at_budget_ratio": schema.Float64Attribute{Computed: true, Description: "Share of the consumed budget that triggers the warning."},
						"enabled":              schema.BoolAttribute{Computed: true, Description: "The target is active."},

						// The measurement is flattened rather than nested: a
						// `precondition` reading `targets[0].state` is easier
						// to write and to read than one going through an
						// intermediate object.
						"state":                   schema.StringAttribute{Computed: true, Description: "`ok`, `warning`, `breached` or `unknown`."},
						"uptime_pct":              schema.Float64Attribute{Computed: true, Description: "Availability measured over the window."},
						"eligible_runs":           schema.Int64Attribute{Computed: true, Description: "Runs taken into account."},
						"ok_runs":                 schema.Int64Attribute{Computed: true, Description: "Successful runs."},
						"failed_runs":             schema.Int64Attribute{Computed: true, Description: "Failed runs."},
						"excluded_runs":           schema.Int64Attribute{Computed: true, Description: "Runs excluded, by maintenance for instance."},
						"error_budget_runs":       schema.Int64Attribute{Computed: true, Description: "Size of the error budget, in runs."},
						"error_budget_used_ratio": schema.Float64Attribute{Computed: true, Description: "Share of the budget consumed, from 0 to 1."},
						"estimated_downtime_minutes": schema.Float64Attribute{
							Computed:    true,
							Description: "Downtime estimated over the window, in minutes.",
						},
					},
				},
			},
		},
	}
}

func (d *slaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *slaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config slaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}

	all, err := d.client.ListSlaOverview(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the availability objectives", err.Error())
		return
	}

	scenario := strings.TrimSpace(config.FilterScenarioID.ValueString())
	state := strings.TrimSpace(config.FilterState.ValueString())
	config.Targets = nil
	for i := range all {
		t := all[i]
		if scenario != "" && (t.MonitorID == nil || *t.MonitorID != scenario) {
			continue
		}
		// A target with no measurement yet has no state: filtering on one must
		// leave it out rather than compare against an empty value.
		if state != "" && (t.Status == nil || t.Status.State == nil || *t.Status.State != state) {
			continue
		}
		config.Targets = append(config.Targets, slaToEntry(&t))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}

func slaToEntry(t *client.SlaOverview) slaEntry {
	entry := slaEntry{
		ID:                 types.StringValue(t.ID),
		ScenarioID:         stringFrom(t.MonitorID),
		ScenarioName:       stringFrom(t.MonitorName),
		Name:               stringFrom(t.Name),
		ObjectivePct:       numberFrom(t.ObjectivePct),
		WindowDays:         int64From(t.WindowDays),
		ExcludeMaintenance: boolFrom(t.ExcludeMaintenance),
		WarnAtBudgetRatio:  numberFrom(t.WarnAtBudgetRatio),
		Enabled:            boolFrom(t.Enabled),
	}
	if t.Status == nil {
		// Every measurement attribute stays null: a zero would read as "no
		// downtime" when the truth is "nothing measured yet".
		return entry
	}
	entry.State = stringFrom(t.Status.State)
	entry.UptimePct = numberFrom(t.Status.UptimePct)
	entry.EligibleRuns = int64From(t.Status.EligibleRuns)
	entry.OkRuns = int64From(t.Status.OkRuns)
	entry.FailedRuns = int64From(t.Status.FailedRuns)
	entry.ExcludedRuns = int64From(t.Status.ExcludedRuns)
	entry.ErrorBudgetRuns = int64From(t.Status.ErrorBudgetRuns)
	entry.ErrorBudgetUsedRatio = numberFrom(t.Status.ErrorBudgetUsedRatio)
	entry.EstimatedDowntimeMinutes = numberFrom(t.Status.EstimatedDowntimeMinutes)
	return entry
}
