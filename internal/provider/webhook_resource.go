package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewWebhookResource declares `pathly_webhook`.
func NewWebhookResource() resource.Resource { return &webhookResource{} }

type webhookResource struct {
	client *client.Client
}

type webhookModel struct {
	ID             types.String `tfsdk:"id"`
	URL            types.String `tfsdk:"url"`
	Events         types.List   `tfsdk:"events"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	URLFingerprint types.String `tfsdk:"url_fingerprint"`
	Secret         types.String `tfsdk:"secret"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An outbound webhook, called on every failure and every recovery.",
		MarkdownDescription: "An outbound webhook, called on every failure and every recovery.\n\n" +
			"The API never reads the stored URL back, only its fingerprint: it is encrypted at rest. " +
			"The provider therefore compares `url_fingerprint` to detect that a destination changed outside of Terraform.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier assigned by Pathly.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"url": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Description: "Destination, over https and public. Private addresses, internal domains " +
					"and cloud metadata IPs are refused: the worker calls from the platform network.",
				Validators: []validator.String{stringvalidator.LengthBetween(8, 500)},
				// The API cannot rewrite the URL of a webhook, and the
				// recreation renews the signing secret: the plan has to say so.
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"events": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Description: "`run.failed` and `run.recovered`. Both by default.",
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 2),
					// Without this closed list, a made-up event name — or one
					// borrowed from another tool, `incident.opened` for example
					// — was only refused at apply time, after the previous
					// resources of the same plan had been created.
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("run.failed", "run.recovered"),
					),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Computed:      true,
				Description:   "A webhook disabled from the console stays in the state, without being called.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"url_fingerprint": schema.StringAttribute{
				Computed:      true,
				Description:   "Fingerprint of the destination. It changes if the URL was modified elsewhere.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"secret": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				Description: "HMAC signing secret of the deliveries, returned once only at creation time. " +
					"It is therefore kept in the Terraform state: encrypt the backend, or read it once " +
					"to store it in your vault then remove it from your outputs.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "Creation date.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	created, err := r.client.CreateWebhook(ctx, client.WebhookInput{
		URL:    plan.URL.ValueString(),
		Events: stringsTo(ctx, plan.Events, &resp.Diagnostics),
	}, newIdempotencyKey("webhook"))
	if err != nil {
		resp.Diagnostics.AddError("Webhook creation refused", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, webhookModel{
		ID:     types.StringValue(created.ID),
		URL:    plan.URL,
		Events: stringsFrom(created.Events),
		// The API returns `enabled` at creation time; a brand new webhook is
		// active.
		Enabled:        boolFrom(created.Enabled),
		URLFingerprint: stringFrom(created.URLFingerprint),
		Secret:         stringFrom(created.Secret),
		CreatedAt:      stringFrom(created.CreatedAt),
	})...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	got, err := r.client.GetWebhook(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the webhook", err.Error())
		return
	}

	// The URL and the secret come from the state: the API does not return them
	// on read, overwriting them with null would make the secret disappear from
	// the state and would propose a replacement on every plan.
	state.Events = stringsFrom(got.Events)
	state.Enabled = boolFrom(got.Enabled)
	state.URLFingerprint = stringFrom(got.URLFingerprint)
	state.CreatedAt = stringFrom(got.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update is unreachable: `url` and `events` force replacement, the rest is
// computed. The message says so rather than letting one believe in a silent
// update.
func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Webhook cannot be modified",
		"The API does not rewrite a webhook: it is destroyed then recreated, and its signing secret changes.",
	)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	err := r.client.DeleteWebhook(ctx, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Webhook deletion refused", err.Error())
	}
}

// ImportState deliberately stays short of a part of its purpose: the URL and
// the secret cannot be read back, so an imported webhook requires `url` to be
// stated again in the configuration.
func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.AddWarning(
		"Webhook imported without its URL or its secret",
		"The API never returns the stored URL nor the signing secret. State `url` again in the configuration, "+
			"and fetch the secret back from your vault: it was only displayed at creation time.",
	)
}
