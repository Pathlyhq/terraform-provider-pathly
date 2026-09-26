package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewScenarioResource declares `pathly_scenario`.
func NewScenarioResource() resource.Resource { return &scenarioResource{} }

type scenarioResource struct {
	client *client.Client
}

type scenarioModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	URL            types.String `tfsdk:"url"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	IntervalSec    types.Int64  `tfsdk:"interval_sec"`
	Method         types.String `tfsdk:"method"`
	ExpectedStatus types.Int64  `tfsdk:"expected_status"`
	MaxLatencyMs   types.Int64  `tfsdk:"max_latency_ms"`
	ExpectText     types.String `tfsdk:"expect_text"`
	Regions        types.List   `tfsdk:"regions"`
	Tags           types.List   `tfsdk:"tags"`
	Folder         types.String `tfsdk:"folder"`
	Severity       types.String `tfsdk:"severity"`
	Runbook        types.String `tfsdk:"runbook"`
	Cron           types.String `tfsdk:"cron"`
	LastStatus     types.String `tfsdk:"last_status"`
	MutedUntil     types.String `tfsdk:"muted_until"`
	CreatedAt      types.String `tfsdk:"created_at"`

	Steps               types.List   `tfsdk:"steps"`
	Headers             types.Map    `tfsdk:"headers"`
	ClickDelayMs        types.Int64  `tfsdk:"click_delay_ms"`
	Viewport            types.String `tfsdk:"viewport"`
	Locale              types.String `tfsdk:"locale"`
	ScenarioTimezone    types.String `tfsdk:"scenario_timezone"`
	BasicAuth           types.Object `tfsdk:"basic_auth"`
	ScenarioFingerprint types.String `tfsdk:"scenario_fingerprint"`
	HttpChain           types.List   `tfsdk:"http_chain"`
}

func (r *scenarioResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scenario"
}

func (r *scenarioResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	/*
	 * An optional and computed attribute left empty in the configuration keeps
	 * the value returned by the API. Without these modifiers, Terraform
	 * declares it "known after apply" as soon as anything else changes in the
	 * resource — and on `type`, which forces a replacement, that used to
	 * destroy and recreate the scenario on every interval change, along with
	 * its whole run history and its incidents.
	 *
	 * Accepted trade-off: removing a line from the configuration does not
	 * revert the attribute to the default of the plan, the wanted value has to
	 * be set.
	 */
	keepString := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	keepBool := []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}
	keepInt64 := []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
	keepList := []planmodifier.List{listplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		Description: "A Ping, a Flow, or a Chain.",
		MarkdownDescription: "A Ping (HTTP check), a Flow (browser journey), or a Chain (HTTP hops).\n\n" +
			"A browser journey is write-only: the API never returns the steps, only `scenario_fingerprint`. " +
			"A `fill` or `http_auth` step can carry a password — keep those values in a secret store, " +
			"not in the repository. The state still holds them, marked sensitive.",
		Attributes: func() map[string]schema.Attribute {
			attrs := map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Computed:      true,
					Description:   "Identifier assigned by Pathly.",
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"name": schema.StringAttribute{
					Required:    true,
					Description: "Name displayed in the console and in alerts.",
					Validators:  []validator.String{stringvalidator.LengthBetween(1, 120)},
				},
				"type": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "`http` or `browser`. The type of a scenario cannot be changed after creation.",
					Validators:  []validator.String{stringvalidator.OneOf("http", "browser")},
					// Changing the type would destroy the history of the scenario:
					// the replacement has to be explicit in the plan. `keepString`
					// comes first, otherwise a type absent from the configuration
					// passes for a change and takes the scenario away with it.
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
				},
				"url": schema.StringAttribute{
					Optional:      true,
					Computed:      true,
					Description:   "Monitored address, over http or https.",
					PlanModifiers: keepString,
				},
				"enabled": schema.BoolAttribute{
					Optional:      true,
					Computed:      true,
					Description:   "When false, the scenario exists but does not run.",
					PlanModifiers: keepBool,
				},
				"interval_sec": schema.Int64Attribute{
					Required:    true,
					Description: "Period between two runs, in seconds. 0 for a scenario driven by `cron` only.",
					Validators:  []validator.Int64{int64validator.Between(0, 2_592_000)},
				},
				"method": schema.StringAttribute{
					Optional:      true,
					Computed:      true,
					Description:   "`GET` or `HEAD`. A monitor that posts or deletes would act on the site at every run.",
					Validators:    []validator.String{stringvalidator.OneOf("GET", "HEAD")},
					PlanModifiers: keepString,
				},
				"expected_status": schema.Int64Attribute{
					Optional:      true,
					Computed:      true,
					Description:   "Expected HTTP status.",
					Validators:    []validator.Int64{int64validator.Between(100, 599)},
					PlanModifiers: keepInt64,
				},
				"max_latency_ms": schema.Int64Attribute{
					Optional:      true,
					Computed:      true,
					Description:   "Above this, the run is a performance failure.",
					Validators:    []validator.Int64{int64validator.Between(100, 600_000)},
					PlanModifiers: keepInt64,
				},
				"expect_text": schema.StringAttribute{
					Optional:      true,
					Computed:      true,
					Description:   "Text expected in the response. A 200 served by an error page is still a failure.",
					Validators:    []validator.String{stringvalidator.LengthAtMost(500)},
					PlanModifiers: keepString,
				},
				"regions": schema.ListAttribute{
					ElementType:   types.StringType,
					Optional:      true,
					Computed:      true,
					Description:   "Probe regions. When empty, Pathly picks the default region of the plan.",
					Validators:    []validator.List{listvalidator.SizeAtMost(8)},
					PlanModifiers: keepList,
				},
				"tags": schema.ListAttribute{
					ElementType:   types.StringType,
					Optional:      true,
					Computed:      true,
					Description:   "Free-form tags, used to filter and group.",
					Validators:    []validator.List{listvalidator.SizeAtMost(10)},
					PlanModifiers: keepList,
				},
				"folder": schema.StringAttribute{
					Optional:      true,
					Computed:      true,
					Description:   "Folder used to organize the console.",
					Validators:    []validator.String{stringvalidator.LengthAtMost(60)},
					PlanModifiers: keepString,
				},
				"severity": schema.StringAttribute{
					Optional:      true,
					Computed:      true,
					Description:   "`critical`, `major` or `minor`. Drives escalation.",
					Validators:    []validator.String{stringvalidator.OneOf("critical", "major", "minor")},
					PlanModifiers: keepString,
				},
				"runbook": schema.StringAttribute{
					Optional:      true,
					Computed:      true,
					Description:   "On-call instructions, attached to the alert.",
					Validators:    []validator.String{stringvalidator.LengthAtMost(2000)},
					PlanModifiers: keepString,
				},
				"cron": schema.StringAttribute{
					Optional:      true,
					Computed:      true,
					Description:   "Cron schedule, in addition to or instead of `interval_sec`.",
					Validators:    []validator.String{stringvalidator.LengthAtMost(120)},
					PlanModifiers: keepString,
				},
				"last_status": schema.StringAttribute{
					Computed:    true,
					Description: "Verdict of the last known run.",
				},
				"muted_until": schema.StringAttribute{
					Computed: true,
					Description: "Mute deadline, set from the console or the API. " +
						"Read only here: a mute is a temporary operational gesture, not a desired state.",
				},
				"created_at": schema.StringAttribute{
					Computed:      true,
					Description:   "Creation date.",
					PlanModifiers: keepString,
				},
			}
			for name, attr := range journeyAttributes() {
				attrs[name] = attr
			}
			return attrs
		}(),
	}
}

func (r *scenarioResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *scenarioResource) inputFrom(ctx context.Context, m scenarioModel, diags *diag.Diagnostics) client.ScenarioInput {
	return client.ScenarioInput{
		Name:           strPtr(m.Name),
		Type:           strPtr(m.Type),
		URL:            strPtr(m.URL),
		IntervalSec:    int64Ptr(m.IntervalSec),
		Method:         strPtr(m.Method),
		ExpectedStatus: int64Ptr(m.ExpectedStatus),
		MaxLatencyMs:   int64Ptr(m.MaxLatencyMs),
		ExpectText:     strPtr(m.ExpectText),
		Regions:        stringsTo(ctx, m.Regions, diags),
		Tags:           stringsTo(ctx, m.Tags, diags),
		Folder:         strPtr(m.Folder),
		Severity:       strPtr(m.Severity),
		Runbook:        strPtr(m.Runbook),
		Cron:           strPtr(m.Cron),
		Enabled:        boolPtr(m.Enabled),
		Scenario:       astFrom(ctx, m, diags),
		HttpChain:      hopsFrom(ctx, m, diags),
	}
}

func scenarioToModel(ctx context.Context, s *client.Scenario, keep scenarioModel) scenarioModel {
	return scenarioModel{
		ID:             types.StringValue(s.ID),
		Name:           types.StringValue(s.Name),
		Type:           types.StringValue(s.Type),
		URL:            stringFrom(s.URL),
		Enabled:        boolFrom(s.Enabled),
		IntervalSec:    int64From(s.IntervalSec),
		Method:         stringFrom(s.Method),
		ExpectedStatus: int64From(s.ExpectedStatus),
		MaxLatencyMs:   int64From(s.MaxLatencyMs),
		ExpectText:     stringFrom(s.ExpectText),
		Regions:        stringsFrom(s.Regions),
		Tags:           stringsFrom(s.Tags),
		Folder:         stringFrom(s.Folder),
		Severity:       stringFrom(s.Severity),
		Runbook:        stringFrom(s.Runbook),
		Cron:           stringFrom(s.Cron),
		LastStatus:     stringFrom(s.LastStatus),
		MutedUntil:     stringFrom(s.MutedUntil),
		CreatedAt:      stringFrom(s.CreatedAt),
		// The API never returns the tree: keeping the plan's copy is the only
		// way the next plan does not announce the deletion of every step.
		// Unknown (omitted computed) becomes a typed null: Terraform refuses a
		// state that still carries unknown after apply.
		Steps:               knownList(ctx, keep.Steps),
		Headers:             knownMap(ctx, keep.Headers),
		ClickDelayMs:        knownInt64(keep.ClickDelayMs),
		Viewport:            knownString(keep.Viewport),
		Locale:              knownString(keep.Locale),
		ScenarioTimezone:    knownString(keep.ScenarioTimezone),
		BasicAuth:           knownObject(ctx, keep.BasicAuth),
		ScenarioFingerprint: stringFrom(s.ScenarioFingerprint),
		HttpChain:           knownList(ctx, keep.HttpChain),
	}
}

func (r *scenarioResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scenarioModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	requireJourneyOrURL(plan, &resp.Diagnostics)
	firstStepMustBeGoto(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	input := r.inputFrom(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateScenario(ctx, input, newIdempotencyKey("scenario"))
	if err != nil {
		resp.Diagnostics.AddError("Scenario creation refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, scenarioToModel(ctx, created, plan))...)
}

func (r *scenarioResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scenarioModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	got, err := r.client.GetScenario(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		// Deleted outside of Terraform: dropping it from the state lets the
		// next plan recreate it, instead of failing forever.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the scenario", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, scenarioToModel(ctx, got, state))...)
}

func (r *scenarioResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scenarioModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state scenarioModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	firstStepMustBeGoto(ctx, plan, &resp.Diagnostics)
	input := r.inputFrom(ctx, plan, &resp.Diagnostics)
	// The type cannot be modified: sending it would make the whole request be
	// refused.
	input.Type = nil
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateScenario(ctx, state.ID.ValueString(), input)
	if client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Scenario vanished",
			fmt.Sprintf("Scenario %s no longer exists. Run `terraform apply` again to recreate it.", state.ID.ValueString()),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Scenario update refused", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, scenarioToModel(ctx, updated, plan))...)
}

func (r *scenarioResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scenarioModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	err := r.client.DeleteScenario(ctx, state.ID.ValueString())
	// Already gone: the destruction has the wanted result, failing would be a
	// false negative that would block the deletion of everything else.
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Scenario deletion refused", err.Error())
	}
}

func (r *scenarioResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
