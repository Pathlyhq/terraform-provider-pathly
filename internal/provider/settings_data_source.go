package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewSettingsDataSource declares `pathly_settings`.
//
// Reading without managing: a configuration often needs the status page slug or
// the timezone to build a URL or a schedule, without claiming ownership of the
// settings. Using the resource for that would make an `apply` write them back.
func NewSettingsDataSource() datasource.DataSource { return &settingsDataSource{} }

type settingsDataSource struct {
	client *client.Client
}

func (d *settingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

func (d *settingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Organization settings, read-only. Use the `pathly_settings` resource to change them.",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Computed: true, Description: "Always `" + settingsID + "`."},
			"alert_email":            schema.StringAttribute{Computed: true, Description: "Address that receives the failure alerts."},
			"alert_on_recovery":      schema.BoolAttribute{Computed: true, Description: "A recovery is announced as well as a failure."},
			"escalation_after_fails": schema.Int64Attribute{Computed: true, Description: "Consecutive failures before escalating."},
			"escalation_email":       schema.StringAttribute{Computed: true, Description: "Address warned on escalation."},
			"weekly_digest_email":    schema.StringAttribute{Computed: true, Description: "Recipient of the weekly summary."},
			"weekly_digest_enabled":  schema.BoolAttribute{Computed: true, Description: "The weekly summary is sent."},
			"status_slug":            schema.StringAttribute{Computed: true, Description: "Slug of the public status page."},
			"status_public":          schema.BoolAttribute{Computed: true, Description: "The status page is reachable by anyone holding the link."},
			"timezone":               schema.StringAttribute{Computed: true, Description: "IANA timezone of the reports and maintenance windows."},
			"ssl_warn_days": schema.ListAttribute{
				ElementType: types.Int64Type,
				Computed:    true,
				Description: "Days before certificate expiry at which a warning is sent.",
			},
			"domain_warn_days": schema.ListAttribute{
				ElementType: types.Int64Type,
				Computed:    true,
				Description: "Days before domain expiry at which a warning is sent.",
			},
			"triage_enabled":         schema.BoolAttribute{Computed: true, Description: "Failures are sorted between real incidents and noise."},
			"triage_confirm_enabled": schema.BoolAttribute{Computed: true, Description: "A confirmation run precedes opening an incident."},
			"triage_latency_factor":  schema.Float64Attribute{Computed: true, Description: "Latency multiple above which a run counts as degraded."},
			"name":                   schema.StringAttribute{Computed: true, Description: "Organization name."},
			"plan_id":                schema.StringAttribute{Computed: true, Description: "Active plan."},

			// Presence only, never the URL nor the key: this data source can be
			// used to check that alerting is wired without putting a secret in
			// the state.
			"has_slack_webhook":   schema.BoolAttribute{Computed: true, Description: "A Slack destination is configured."},
			"has_teams_webhook":   schema.BoolAttribute{Computed: true, Description: "A Microsoft Teams destination is configured."},
			"has_discord_webhook": schema.BoolAttribute{Computed: true, Description: "A Discord destination is configured."},
			"has_pagerduty":       schema.BoolAttribute{Computed: true, Description: "A PagerDuty integration is configured."},
			"has_opsgenie":        schema.BoolAttribute{Computed: true, Description: "An Opsgenie integration is configured."},
			"has_datadog":         schema.BoolAttribute{Computed: true, Description: "A Datadog integration is configured."},
			"has_sentry":          schema.BoolAttribute{Computed: true, Description: "A Sentry integration is configured."},
		},
	}
}

func (d *settingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *settingsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}
	got, err := d.client.GetSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the settings", err.Error())
		return
	}
	// Same model as the resource: one schema to keep in step instead of two that
	// would drift apart the day a field is added.
	resp.Diagnostics.Append(resp.State.Set(ctx, settingsToModel(got))...)
}
