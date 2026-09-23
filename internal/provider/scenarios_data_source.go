package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewScenariosDataSource declares `pathly_scenarios`.
//
// Main use: taking over an existing estate. Without it, adopting scenarios
// created in the console meant collecting their identifiers one by one in order
// to import them.
func NewScenariosDataSource() datasource.DataSource { return &scenariosDataSource{} }

type scenariosDataSource struct {
	client *client.Client
}

type scenariosDataSourceModel struct {
	FilterTag    types.String        `tfsdk:"filter_tag"`
	FilterFolder types.String        `tfsdk:"filter_folder"`
	Scenarios    []scenarioListEntry `tfsdk:"scenarios"`
}

type scenarioListEntry struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	URL         types.String `tfsdk:"url"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	IntervalSec types.Int64  `tfsdk:"interval_sec"`
	Folder      types.String `tfsdk:"folder"`
	Severity    types.String `tfsdk:"severity"`
	Tags        types.List   `tfsdk:"tags"`
	LastStatus  types.String `tfsdk:"last_status"`
}

func (d *scenariosDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scenarios"
}

func (d *scenariosDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Every scenario of the organization, filterable by tag or by folder.",
		Attributes: map[string]schema.Attribute{
			"filter_tag": schema.StringAttribute{
				Optional:    true,
				Description: "Keeps only the scenarios carrying this tag.",
			},
			"filter_folder": schema.StringAttribute{
				Optional:    true,
				Description: "Keeps only the scenarios in this folder.",
			},
			"scenarios": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Scenarios found, in the order returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.StringAttribute{Computed: true, Description: "Identifier of the scenario."},
						"name":         schema.StringAttribute{Computed: true, Description: "Name shown in the console and on the status page."},
						"type":         schema.StringAttribute{Computed: true, Description: "`http` or `browser`."},
						"url":          schema.StringAttribute{Computed: true, Description: "Address that is checked."},
						"enabled":      schema.BoolAttribute{Computed: true, Description: "The scenario runs on its schedule."},
						"interval_sec": schema.Int64Attribute{Computed: true, Description: "Seconds between two executions."},
						"folder":       schema.StringAttribute{Computed: true, Description: "Folder the scenario is filed under."},
						"severity":     schema.StringAttribute{Computed: true, Description: "Weight given to a failure of this scenario."},
						"tags":         schema.ListAttribute{ElementType: types.StringType, Computed: true, Description: "Tags carried by the scenario."},
						"last_status":  schema.StringAttribute{Computed: true, Description: "Result of the most recent execution."},
					},
				},
			},
		},
	}
}

func (d *scenariosDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *scenariosDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scenariosDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}

	all, err := d.client.ListScenarios(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the scenarios", err.Error())
		return
	}

	tag := strings.TrimSpace(config.FilterTag.ValueString())
	folder := strings.TrimSpace(config.FilterFolder.ValueString())
	config.Scenarios = nil
	for i := range all {
		s := all[i]
		if tag != "" && !hasTag(s.Tags, tag) {
			continue
		}
		if folder != "" && (s.Folder == nil || *s.Folder != folder) {
			continue
		}
		config.Scenarios = append(config.Scenarios, scenarioListEntry{
			ID:          types.StringValue(s.ID),
			Name:        types.StringValue(s.Name),
			Type:        types.StringValue(s.Type),
			URL:         stringFrom(s.URL),
			Enabled:     boolFrom(s.Enabled),
			IntervalSec: int64From(s.IntervalSec),
			Folder:      stringFrom(s.Folder),
			Severity:    stringFrom(s.Severity),
			Tags:        stringsFrom(s.Tags),
			LastStatus:  stringFrom(s.LastStatus),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}

func hasTag(tags []string, wanted string) bool {
	for _, t := range tags {
		if t == wanted {
			return true
		}
	}
	return false
}
