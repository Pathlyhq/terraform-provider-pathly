package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewMaintenanceWindowsDataSource declares `pathly_maintenance_windows`.
//
// Windows are often created by whoever runs the deployment, outside Terraform.
// Reading them back is what lets a configuration check that a change is not
// landing during a period when the monitoring is deliberately silent.
func NewMaintenanceWindowsDataSource() datasource.DataSource {
	return &maintenanceWindowsDataSource{}
}

type maintenanceWindowsDataSource struct {
	client *client.Client
}

type maintenanceWindowsDataSourceModel struct {
	FilterScenarioID types.String             `tfsdk:"filter_scenario_id"`
	Windows          []maintenanceWindowEntry `tfsdk:"windows"`
}

type maintenanceWindowEntry struct {
	ID          types.String `tfsdk:"id"`
	ScenarioID  types.String `tfsdk:"scenario_id"`
	StartsAt    types.String `tfsdk:"starts_at"`
	EndsAt      types.String `tfsdk:"ends_at"`
	Reason      types.String `tfsdk:"reason"`
	Weekday     types.Int64  `tfsdk:"weekday"`
	StartMinute types.Int64  `tfsdk:"start_minute"`
	DurationMin types.Int64  `tfsdk:"duration_min"`
}

func (d *maintenanceWindowsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_windows"
}

func (d *maintenanceWindowsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Maintenance windows of the organization, one-off and recurring alike.",
		Attributes: map[string]schema.Attribute{
			"filter_scenario_id": schema.StringAttribute{
				Optional:    true,
				Description: "Keeps only the windows covering this scenario.",
			},
			"windows": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Windows found, in the order returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true, Description: "Identifier of the window."},
						"scenario_id": schema.StringAttribute{Computed: true, Description: "Covered scenario, empty when the window covers the organization."},
						"starts_at":   schema.StringAttribute{Computed: true, Description: "Start of a one-off window, ISO 8601."},
						"ends_at":     schema.StringAttribute{Computed: true, Description: "End of a one-off window, ISO 8601."},
						"reason":      schema.StringAttribute{Computed: true, Description: "Reason stated for the window."},
						// The recurring form uses these three instead of the two
						// timestamps, and never both at once.
						"weekday":      schema.Int64Attribute{Computed: true, Description: "Day of the week of a recurring window."},
						"start_minute": schema.Int64Attribute{Computed: true, Description: "Start of a recurring window, in minutes past midnight."},
						"duration_min": schema.Int64Attribute{Computed: true, Description: "Duration of a recurring window, in minutes."},
					},
				},
			},
		},
	}
}

func (d *maintenanceWindowsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *maintenanceWindowsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config maintenanceWindowsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}

	all, err := d.client.ListMaintenanceWindows(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the maintenance windows", err.Error())
		return
	}

	scenario := strings.TrimSpace(config.FilterScenarioID.ValueString())
	config.Windows = nil
	for i := range all {
		w := all[i]
		if scenario != "" && (w.MonitorID == nil || *w.MonitorID != scenario) {
			continue
		}
		config.Windows = append(config.Windows, maintenanceWindowEntry{
			ID:          types.StringValue(w.ID),
			ScenarioID:  stringFrom(w.MonitorID),
			StartsAt:    stringFrom(w.StartsAt),
			EndsAt:      stringFrom(w.EndsAt),
			Reason:      stringFrom(w.Reason),
			Weekday:     int64From(w.Weekday),
			StartMinute: int64From(w.StartMinute),
			DurationMin: int64From(w.DurationMin),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
