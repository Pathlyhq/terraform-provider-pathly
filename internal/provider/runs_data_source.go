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

// runEntry is shared by `pathly_runs` and `pathly_run`: one execution, seen the
// same way whether it was read in a listing or on its own.
type runEntry struct {
	ID              types.String  `tfsdk:"id"`
	ScenarioID      types.String  `tfsdk:"scenario_id"`
	ScenarioName    types.String  `tfsdk:"scenario_name"`
	ScenarioType    types.String  `tfsdk:"scenario_type"`
	Status          types.String  `tfsdk:"status"`
	LatencyMs       types.Int64   `tfsdk:"latency_ms"`
	Message         types.String  `tfsdk:"message"`
	ProbeRegion     types.String  `tfsdk:"probe_region"`
	ComparedToRunID types.String  `tfsdk:"compared_to_run_id"`
	ConfirmsRunID   types.String  `tfsdk:"confirms_run_id"`
	TriageVerdict   types.String  `tfsdk:"triage_verdict"`
	TriageReason    types.String  `tfsdk:"triage_reason"`
	InMaintenance   types.Bool    `tfsdk:"in_maintenance"`
	BrowserSeconds  types.Float64 `tfsdk:"browser_seconds"`
	ScreenshotPath  types.String  `tfsdk:"screenshot_path"`
	VideoPath       types.String  `tfsdk:"video_path"`
	CreatedAt       types.String  `tfsdk:"created_at"`
}

func runToEntry(r *client.Run) runEntry {
	return runEntry{
		ID:              types.StringValue(r.ID),
		ScenarioID:      stringFrom(r.MonitorID),
		ScenarioName:    stringFrom(r.MonitorName),
		ScenarioType:    stringFrom(r.MonitorType),
		Status:          stringFrom(r.Status),
		LatencyMs:       int64From(r.LatencyMs),
		Message:         stringFrom(r.Message),
		ProbeRegion:     stringFrom(r.ProbeRegion),
		ComparedToRunID: stringFrom(r.ComparedToRunID),
		ConfirmsRunID:   stringFrom(r.ConfirmsRunID),
		TriageVerdict:   stringFrom(r.TriageVerdict),
		TriageReason:    stringFrom(r.TriageReason),
		InMaintenance:   boolFrom(r.InMaintenance),
		BrowserSeconds:  numberFrom(r.BrowserSeconds),
		ScreenshotPath:  stringFrom(r.ScreenshotPath),
		VideoPath:       stringFrom(r.VideoPath),
		CreatedAt:       stringFrom(r.CreatedAt),
	}
}

// runAttributes describes an execution.
//
// The step-by-step results and the triage signals are left out: they are
// free-form JSON whose shape follows the scenario, and forcing them into an
// attribute would mean exposing a string nobody can plan against. They are
// readable over the API and in the console, where they belong.
func runAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                 schema.StringAttribute{Computed: true, Description: "Identifier of the run."},
		"scenario_id":        schema.StringAttribute{Computed: true, Description: "Scenario that was executed."},
		"scenario_name":      schema.StringAttribute{Computed: true, Description: "Name of that scenario."},
		"scenario_type":      schema.StringAttribute{Computed: true, Description: "Type of that scenario."},
		"status":             schema.StringAttribute{Computed: true, Description: "`ok`, `fail` or `error`."},
		"latency_ms":         schema.Int64Attribute{Computed: true, Description: "Measured latency, in milliseconds."},
		"message":            schema.StringAttribute{Computed: true, Description: "Reason for the failure, when there is one."},
		"probe_region":       schema.StringAttribute{Computed: true, Description: "Region the run was launched from."},
		"compared_to_run_id": schema.StringAttribute{Computed: true, Description: "Reference run used for the comparison."},
		"confirms_run_id":    schema.StringAttribute{Computed: true, Description: "Run this one confirms, when triage asked for a second pass."},
		"triage_verdict":     schema.StringAttribute{Computed: true, Description: "`real`, `suspect` or `benign`."},
		"triage_reason":      schema.StringAttribute{Computed: true, Description: "What led to that verdict."},
		"in_maintenance":     schema.BoolAttribute{Computed: true, Description: "The run happened inside a maintenance window."},
		"browser_seconds":    schema.Float64Attribute{Computed: true, Description: "Browser time consumed."},
		"screenshot_path":    schema.StringAttribute{Computed: true, Description: "Path of the capture, when one was taken."},
		"video_path":         schema.StringAttribute{Computed: true, Description: "Path of the recording, when one was made."},
		"created_at":         schema.StringAttribute{Computed: true, Description: "Execution timestamp, ISO 8601."},
	}
}

// ---------------------------------------------------------------------------
// pathly_runs
// ---------------------------------------------------------------------------

// NewRunsDataSource declares `pathly_runs`.
//
// A listing of executions is history, not desired state: reading it into a
// configuration makes the plan depend on what the monitoring found, which is
// useful for a report or a `precondition` and misleading anywhere else. Filter
// on a scenario whenever possible — the unfiltered listing walks the whole
// organization.
func NewRunsDataSource() datasource.DataSource { return &runsDataSource{} }

type runsDataSource struct {
	client *client.Client
}

type runsDataSourceModel struct {
	FilterScenarioID types.String `tfsdk:"filter_scenario_id"`
	FilterStatus     types.String `tfsdk:"filter_status"`
	Runs             []runEntry   `tfsdk:"runs"`
}

func (d *runsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runs"
}

func (d *runsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Scenario executions, most recent first.",
		Attributes: map[string]schema.Attribute{
			"filter_scenario_id": schema.StringAttribute{
				Optional: true,
				Description: "Reads only the executions of this scenario. " +
					"Passed to the API, unlike the other filter, so it also lowers the number of pages read.",
			},
			"filter_status": schema.StringAttribute{
				Optional:    true,
				Description: "Keeps only the executions in this state, `ok`, `fail` or `error`.",
				Validators:  []validator.String{stringvalidator.OneOf("ok", "fail", "error")},
			},
			"runs": schema.ListNestedAttribute{
				Computed:     true,
				Description:  "Executions found, in the order returned by the API.",
				NestedObject: schema.NestedAttributeObject{Attributes: runAttributes()},
			},
		},
	}
}

func (d *runsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *runsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}

	all, err := d.client.ListRuns(ctx, strings.TrimSpace(config.FilterScenarioID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the executions", err.Error())
		return
	}

	status := strings.TrimSpace(config.FilterStatus.ValueString())
	config.Runs = nil
	for i := range all {
		r := all[i]
		if status != "" && (r.Status == nil || *r.Status != status) {
			continue
		}
		config.Runs = append(config.Runs, runToEntry(&r))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}

// ---------------------------------------------------------------------------
// pathly_run
// ---------------------------------------------------------------------------

// NewRunDataSource declares `pathly_run`.
//
// Reads a single execution, by identifier. The API answers a narrower
// projection here than in the listing, so a few attributes come back empty —
// that is the API being economical, not the run missing anything.
func NewRunDataSource() datasource.DataSource { return &runDataSource{} }

type runDataSource struct {
	client *client.Client
}

func (d *runDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run"
}

func (d *runDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := runAttributes()
	attributes["id"] = schema.StringAttribute{
		Required:    true,
		Description: "Identifier of the execution to read.",
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
	}
	resp.Schema = schema.Schema{
		Description: "A single scenario execution and its verdict.",
		Attributes:  attributes,
	}
}

func (d *runDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *runDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runEntry
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}

	got, err := d.client.GetRun(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the execution", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, runToEntry(got))...)
}
