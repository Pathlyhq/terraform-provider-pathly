package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewIncidentsDataSource declares `pathly_incidents`.
//
// Read-only, and it will stay that way: an incident is opened by a failing run,
// not by a configuration file. Resolving one or writing its postmortem are
// actions taken at a point in time, which Terraform has no way to express as a
// desired state — an `apply` replayed a week later would reopen a settled
// discussion.
func NewIncidentsDataSource() datasource.DataSource { return &incidentsDataSource{} }

type incidentsDataSource struct {
	client *client.Client
}

type incidentsDataSourceModel struct {
	FilterStatus     types.String    `tfsdk:"filter_status"`
	FilterScenarioID types.String    `tfsdk:"filter_scenario_id"`
	Incidents        []incidentEntry `tfsdk:"incidents"`
}

type incidentEntry struct {
	ID               types.String `tfsdk:"id"`
	ScenarioID       types.String `tfsdk:"scenario_id"`
	ScenarioName     types.String `tfsdk:"scenario_name"`
	Status           types.String `tfsdk:"status"`
	Title            types.String `tfsdk:"title"`
	OpenedAt         types.String `tfsdk:"opened_at"`
	ResolvedAt       types.String `tfsdk:"resolved_at"`
	Postmortem       types.String `tfsdk:"postmortem"`
	PublicPostmortem types.Bool   `tfsdk:"public_postmortem"`
	OpenedRunID      types.String `tfsdk:"opened_run_id"`
	ResolvedRunID    types.String `tfsdk:"resolved_run_id"`
	AssigneeEmail    types.String `tfsdk:"assignee_email"`
	AssigneeName     types.String `tfsdk:"assignee_name"`
	WorkItemID       types.Int64  `tfsdk:"work_item_id"`
	WorkItemURL      types.String `tfsdk:"work_item_url"`
}

func (d *incidentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_incidents"
}

func (d *incidentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Incidents of the organization, most recent first.",
		Attributes: map[string]schema.Attribute{
			"filter_status": schema.StringAttribute{
				Optional:    true,
				Description: "Keeps only the incidents in this state, `open` or `resolved`.",
				Validators:  []validator.String{stringvalidator.OneOf("open", "resolved")},
			},
			"filter_scenario_id": schema.StringAttribute{
				Optional:    true,
				Description: "Keeps only the incidents of this scenario.",
			},
			"incidents": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Incidents found, in the order returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                schema.StringAttribute{Computed: true, Description: "Identifier of the incident."},
						"scenario_id":       schema.StringAttribute{Computed: true, Description: "Scenario whose failure opened it."},
						"scenario_name":     schema.StringAttribute{Computed: true, Description: "Name of that scenario."},
						"status":            schema.StringAttribute{Computed: true, Description: "`open` or `resolved`."},
						"title":             schema.StringAttribute{Computed: true, Description: "Title of the incident."},
						"opened_at":         schema.StringAttribute{Computed: true, Description: "Opening timestamp, ISO 8601."},
						"resolved_at":       schema.StringAttribute{Computed: true, Description: "Resolution timestamp, empty while open."},
						"postmortem":        schema.StringAttribute{Computed: true, Description: "Postmortem, when written."},
						"public_postmortem": schema.BoolAttribute{Computed: true, Description: "The postmortem appears on the public status page."},
						"opened_run_id":     schema.StringAttribute{Computed: true, Description: "Run that opened the incident."},
						"resolved_run_id":   schema.StringAttribute{Computed: true, Description: "Run that closed it."},
						"assignee_email":    schema.StringAttribute{Computed: true, Description: "Email of the assignee."},
						"assignee_name":     schema.StringAttribute{Computed: true, Description: "Name of the assignee."},
						"work_item_id":      schema.Int64Attribute{Computed: true, Description: "Linked Azure DevOps work item."},
						"work_item_url":     schema.StringAttribute{Computed: true, Description: "URL of that work item."},
					},
				},
			},
		},
	}
}

func (d *incidentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *incidentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config incidentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}

	all, err := d.client.ListIncidents(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the incidents", err.Error())
		return
	}

	// Filtered here rather than server-side: the API takes no filter on this
	// endpoint, and a data source that silently ignored the attribute would be
	// worse than one that reads a bit more than needed.
	status := strings.TrimSpace(config.FilterStatus.ValueString())
	scenario := strings.TrimSpace(config.FilterScenarioID.ValueString())
	config.Incidents = nil
	for i := range all {
		in := all[i]
		if status != "" && (in.Status == nil || *in.Status != status) {
			continue
		}
		if scenario != "" && (in.MonitorID == nil || *in.MonitorID != scenario) {
			continue
		}
		config.Incidents = append(config.Incidents, incidentEntry{
			ID:               types.StringValue(in.ID),
			ScenarioID:       stringFrom(in.MonitorID),
			ScenarioName:     stringFrom(in.MonitorName),
			Status:           stringFrom(in.Status),
			Title:            stringFrom(in.Title),
			OpenedAt:         stringFrom(in.OpenedAt),
			ResolvedAt:       stringFrom(in.ResolvedAt),
			Postmortem:       stringFrom(in.Postmortem),
			PublicPostmortem: boolFrom(in.PublicPostmortem),
			OpenedRunID:      stringFrom(in.OpenedRunID),
			ResolvedRunID:    stringFrom(in.ResolvedRunID),
			AssigneeEmail:    stringFrom(in.AssigneeEmail),
			AssigneeName:     stringFrom(in.AssigneeName),
			WorkItemID:       int64From(in.WorkItemID),
			WorkItemURL:      stringFrom(in.WorkItemURL),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
