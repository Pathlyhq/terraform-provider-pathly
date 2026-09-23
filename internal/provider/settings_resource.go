package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// settingsID is the fixed identifier of the singleton.
//
// The organization has exactly one settings object, so there is nothing to
// number. A stable value keeps `terraform import pathly_settings.this
// anything` predictable, and keeps the state from depending on an internal
// identifier the API never exposes.
const settingsID = "settings"

// slugPattern is what a status page slug may contain. Refused early rather than
// by the API: the slug ends up in a public URL, and a value the browser has to
// escape gives a link that does not survive a copy and paste.
var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// NewSettingsResource declares `pathly_settings`.
//
// It is the only singleton of the provider: the settings exist before Terraform
// and survive it. Create and Update are therefore the same call, and Delete
// changes nothing on the Pathly side — see the comment on Delete, this is a
// deliberate decision and not an omission.
func NewSettingsResource() resource.Resource { return &settingsResource{} }

type settingsResource struct {
	client *client.Client
}

type settingsModel struct {
	ID types.String `tfsdk:"id"`

	AlertEmail           types.String  `tfsdk:"alert_email"`
	AlertOnRecovery      types.Bool    `tfsdk:"alert_on_recovery"`
	EscalationAfterFails types.Int64   `tfsdk:"escalation_after_fails"`
	EscalationEmail      types.String  `tfsdk:"escalation_email"`
	WeeklyDigestEmail    types.String  `tfsdk:"weekly_digest_email"`
	WeeklyDigestEnabled  types.Bool    `tfsdk:"weekly_digest_enabled"`
	StatusSlug           types.String  `tfsdk:"status_slug"`
	StatusPublic         types.Bool    `tfsdk:"status_public"`
	Timezone             types.String  `tfsdk:"timezone"`
	SSLWarnDays          types.List    `tfsdk:"ssl_warn_days"`
	DomainWarnDays       types.List    `tfsdk:"domain_warn_days"`
	TriageEnabled        types.Bool    `tfsdk:"triage_enabled"`
	TriageConfirmEnabled types.Bool    `tfsdk:"triage_confirm_enabled"`
	TriageLatencyFactor  types.Float64 `tfsdk:"triage_latency_factor"`

	Name              types.String `tfsdk:"name"`
	PlanID            types.String `tfsdk:"plan_id"`
	HasSlackWebhook   types.Bool   `tfsdk:"has_slack_webhook"`
	HasTeamsWebhook   types.Bool   `tfsdk:"has_teams_webhook"`
	HasDiscordWebhook types.Bool   `tfsdk:"has_discord_webhook"`
	HasPagerduty      types.Bool   `tfsdk:"has_pagerduty"`
	HasOpsgenie       types.Bool   `tfsdk:"has_opsgenie"`
	HasDatadog        types.Bool   `tfsdk:"has_datadog"`
	HasSentry         types.Bool   `tfsdk:"has_sentry"`
}

func (r *settingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

func (r *settingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Organization-wide settings: alerting, escalation, status page, timezone and triage. " +
			"There is one per organization, so declaring this resource twice would have the two definitions " +
			"overwrite each other on every apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Always `" + settingsID + "`: the organization has a single settings object.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			// Every writable attribute is Optional and Computed: what the
			// configuration leaves out keeps the value already set in the
			// console, instead of being reset to a zero on the first apply.
			"alert_email": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Address that receives the failure alerts.",
				Validators:    []validator.String{stringvalidator.LengthAtMost(320)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"alert_on_recovery": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "When true, a recovery is announced as well as a failure.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"escalation_after_fails": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Number of consecutive failures before escalating.",
				Validators:    []validator.Int64{int64validator.Between(1, 100)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"escalation_email": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Address warned once the escalation threshold is reached.",
				Validators:    []validator.String{stringvalidator.LengthAtMost(320)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"weekly_digest_email": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Recipient of the weekly summary.",
				Validators:    []validator.String{stringvalidator.LengthAtMost(320)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"weekly_digest_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Sends the weekly summary.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"status_slug": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Slug of the public status page. Changing it breaks the previous link.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 64),
					stringvalidator.RegexMatches(slugPattern, "only lowercase letters, digits and hyphens"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status_public": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Exposes the status page to anyone holding the link. " +
					"Turning it on publishes the scenario names and their availability, so check " +
					"that none of them names an internal host.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"timezone": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "IANA timezone used by the reports and the maintenance windows, for example `Europe/Paris`.",
				Validators:    []validator.String{stringvalidator.LengthAtMost(64)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ssl_warn_days": schema.ListAttribute{
				ElementType:   types.Int64Type,
				Optional:      true,
				Computed:      true,
				Description:   "Days before a certificate expires at which to warn, for example `[30, 14, 7]`.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"domain_warn_days": schema.ListAttribute{
				ElementType:   types.Int64Type,
				Optional:      true,
				Computed:      true,
				Description:   "Days before a domain expires at which to warn.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"triage_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Sorts failures between real incidents and transient noise.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"triage_confirm_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Runs a confirmation execution before opening an incident.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"triage_latency_factor": schema.Float64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Multiple of the usual latency above which a run is considered degraded.",
				Validators:    []validator.Float64{float64validator.Between(1, 100)},
				PlanModifiers: []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
			},

			// Read-only. The integration flags report that a destination is
			// wired without ever returning its URL or its key: the state stays
			// free of secrets, and a leaked state file reveals no alerting
			// endpoint.
			"name": schema.StringAttribute{
				Computed:      true,
				Description:   "Organization name, read-only.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"plan_id": schema.StringAttribute{
				Computed:      true,
				Description:   "Active plan, read-only.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"has_slack_webhook":   computedFlag("A Slack destination is configured."),
			"has_teams_webhook":   computedFlag("A Microsoft Teams destination is configured."),
			"has_discord_webhook": computedFlag("A Discord destination is configured."),
			"has_pagerduty":       computedFlag("A PagerDuty integration is configured."),
			"has_opsgenie":        computedFlag("An Opsgenie integration is configured."),
			"has_datadog":         computedFlag("A Datadog integration is configured."),
			"has_sentry":          computedFlag("A Sentry integration is configured."),
		},
	}
}

// computedFlag builds the read-only boolean of an integration. Written once
// rather than seven times: these attributes differ only by their wording.
func computedFlag(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Computed:      true,
		Description:   description + " The URL and the key are never returned by the API.",
		PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

func (r *settingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *settingsResource) inputFrom(ctx context.Context, m settingsModel, diags *diag.Diagnostics) client.SettingsInput {
	return client.SettingsInput{
		AlertEmail:           strPtr(m.AlertEmail),
		AlertOnRecovery:      boolPtr(m.AlertOnRecovery),
		EscalationAfterFails: int64Ptr(m.EscalationAfterFails),
		EscalationEmail:      strPtr(m.EscalationEmail),
		WeeklyDigestEmail:    strPtr(m.WeeklyDigestEmail),
		WeeklyDigestEnabled:  boolPtr(m.WeeklyDigestEnabled),
		StatusSlug:           strPtr(m.StatusSlug),
		StatusPublic:         boolPtr(m.StatusPublic),
		Timezone:             strPtr(m.Timezone),
		SSLWarnDays:          int64sTo(ctx, m.SSLWarnDays, diags),
		DomainWarnDays:       int64sTo(ctx, m.DomainWarnDays, diags),
		TriageEnabled:        boolPtr(m.TriageEnabled),
		TriageConfirmEnabled: boolPtr(m.TriageConfirmEnabled),
		TriageLatencyFactor:  float64Ptr(m.TriageLatencyFactor),
	}
}

func settingsToModel(s *client.Settings) settingsModel {
	return settingsModel{
		ID:                   types.StringValue(settingsID),
		AlertEmail:           stringFrom(s.AlertEmail),
		AlertOnRecovery:      boolFrom(s.AlertOnRecovery),
		EscalationAfterFails: int64From(s.EscalationAfterFails),
		EscalationEmail:      stringFrom(s.EscalationEmail),
		WeeklyDigestEmail:    stringFrom(s.WeeklyDigestEmail),
		WeeklyDigestEnabled:  boolFrom(s.WeeklyDigestEnabled),
		StatusSlug:           stringFrom(s.StatusSlug),
		StatusPublic:         boolFrom(s.StatusPublic),
		Timezone:             stringFrom(s.Timezone),
		SSLWarnDays:          int64sFrom(s.SSLWarnDays),
		DomainWarnDays:       int64sFrom(s.DomainWarnDays),
		TriageEnabled:        boolFrom(s.TriageEnabled),
		TriageConfirmEnabled: boolFrom(s.TriageConfirmEnabled),
		TriageLatencyFactor:  numberFrom(s.TriageLatencyFactor),
		Name:                 stringFrom(s.Name),
		PlanID:               stringFrom(s.PlanID),
		HasSlackWebhook:      boolFrom(s.HasSlackWebhook),
		HasTeamsWebhook:      boolFrom(s.HasTeamsWebhook),
		HasDiscordWebhook:    boolFrom(s.HasDiscordWebhook),
		HasPagerduty:         boolFrom(s.HasPagerduty),
		HasOpsgenie:          boolFrom(s.HasOpsgenie),
		HasDatadog:           boolFrom(s.HasDatadog),
		HasSentry:            boolFrom(s.HasSentry),
	}
}

// Create does not create anything: it aligns the existing settings with the
// configuration. Taking over an organization already set up from the console is
// the normal case, and asking the user to import first would only add a step
// with no effect on the result.
func (r *settingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	input := r.inputFrom(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateSettings(ctx, input, newIdempotencyKey("settings"))
	if err != nil {
		resp.Diagnostics.AddError("Settings update refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, settingsToModel(updated))...)
}

func (r *settingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state settingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	got, err := r.client.GetSettings(ctx)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the settings", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, settingsToModel(got))...)
}

func (r *settingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	input := r.inputFrom(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateSettings(ctx, input, "")
	if err != nil {
		resp.Diagnostics.AddError("Settings update refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, settingsToModel(updated))...)
}

// Delete only forgets the resource.
//
// There is no endpoint to delete the settings, and there should not be one: a
// `terraform destroy` on a test workspace would silently cut the alerting of
// the whole organization. Removing the resource therefore hands the settings
// back to the console with their current values.
func (r *settingsResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Settings left in place",
		"Pathly settings are organization-wide and have no deletion. They keep their current values, "+
			"and Terraform simply stops tracking them.",
	)
}

// ImportState ignores the identifier given on the command line: there is only
// one settings object, so a typo must not produce a state pointing at nothing.
func (r *settingsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), settingsID)...)
}
