package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewWebhooksDataSource declares `pathly_webhooks`.
//
// Main use: auditing what an organization sends outwards. The destination URL is
// never returned by the API, only its fingerprint, so this data source can list
// the outbound alerting without putting a callback address in the state.
func NewWebhooksDataSource() datasource.DataSource { return &webhooksDataSource{} }

type webhooksDataSource struct {
	client *client.Client
}

type webhooksDataSourceModel struct {
	Webhooks []webhookEntry `tfsdk:"webhooks"`
}

type webhookEntry struct {
	ID             types.String `tfsdk:"id"`
	Events         types.List   `tfsdk:"events"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	HasSecret      types.Bool   `tfsdk:"has_secret"`
	URLFingerprint types.String `tfsdk:"url_fingerprint"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func (d *webhooksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhooks"
}

func (d *webhooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Outbound webhooks of the organization. The destination URL is never returned, only its fingerprint.",
		Attributes: map[string]schema.Attribute{
			"webhooks": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Webhooks found, in the order returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":      schema.StringAttribute{Computed: true, Description: "Identifier of the webhook."},
						"events":  schema.ListAttribute{ElementType: types.StringType, Computed: true, Description: "Events that trigger the call."},
						"enabled": schema.BoolAttribute{Computed: true, Description: "The webhook is active."},
						"has_secret": schema.BoolAttribute{
							Computed:    true,
							Description: "A signing secret is set. The secret itself is only returned once, on creation.",
						},
						"url_fingerprint": schema.StringAttribute{
							Computed: true,
							Description: "Fingerprint of the destination. Comparing it is how a change made outside " +
								"Terraform is detected without the URL ever being exposed.",
						},
						"created_at": schema.StringAttribute{Computed: true, Description: "Creation timestamp, ISO 8601."},
					},
				},
			},
		},
	}
}

func (d *webhooksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *webhooksDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}
	all, err := d.client.ListWebhooks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the webhooks", err.Error())
		return
	}

	var model webhooksDataSourceModel
	for i := range all {
		w := all[i]
		model.Webhooks = append(model.Webhooks, webhookEntry{
			ID:             types.StringValue(w.ID),
			Events:         stringsFrom(w.Events),
			Enabled:        boolFrom(w.Enabled),
			HasSecret:      boolFrom(w.HasSecret),
			URLFingerprint: stringFrom(w.URLFingerprint),
			CreatedAt:      stringFrom(w.CreatedAt),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
