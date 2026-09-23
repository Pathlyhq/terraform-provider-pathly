package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewStatusPageDataSource declares `pathly_status_page`.
//
// Reads what the public page shows — which is precisely what makes it worth
// checking from a configuration: a `postcondition` on the component names is how
// a team notices that an internal hostname ended up on a page open to anyone
// holding the link.
//
// The API answers 404 when the organization has no page, or keeps it private.
// That is reported as an error rather than an empty page: an empty list would
// read as "nothing to show" when the truth is "nothing published".
func NewStatusPageDataSource() datasource.DataSource { return &statusPageDataSource{} }

type statusPageDataSource struct {
	client *client.Client
}

type statusPageModel struct {
	Uptime30d  types.Float64         `tfsdk:"uptime_30d"`
	Components []statusComponentItem `tfsdk:"components"`
}

type statusComponentItem struct {
	Name      types.String  `tfsdk:"name"`
	Uptime30d types.Float64 `tfsdk:"uptime_30d"`
	Uptime90d types.Float64 `tfsdk:"uptime_90d"`
}

func (d *statusPageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (d *statusPageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Components of the public status page and their availability.",
		Attributes: map[string]schema.Attribute{
			"uptime_30d": schema.Float64Attribute{
				Computed:    true,
				Description: "Overall availability over 30 days, as shown on the page.",
			},
			"components": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Published components, in the order the page displays them.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name shown publicly. Anyone holding the link reads it.",
						},
						"uptime_30d": schema.Float64Attribute{Computed: true, Description: "Availability over 30 days."},
						"uptime_90d": schema.Float64Attribute{Computed: true, Description: "Availability over 90 days."},
					},
				},
			},
		},
	}
}

func (d *statusPageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *statusPageDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}
	got, err := d.client.GetStatusPage(ctx)
	if client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"No public status page",
			"This organization has no status page, or keeps it private. Set `status_slug` and `status_public` "+
				"on the `pathly_settings` resource before reading this data source.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the status page", err.Error())
		return
	}

	model := statusPageModel{Uptime30d: numberFrom(got.Uptime30d)}
	for i := range got.Components {
		c := got.Components[i]
		model.Components = append(model.Components, statusComponentItem{
			Name:      types.StringValue(c.Name),
			Uptime30d: numberFrom(c.Uptime30d),
			Uptime90d: numberFrom(c.Uptime90d),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
