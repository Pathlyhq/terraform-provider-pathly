package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewSpacesDataSource declares `pathly_spaces`.
//
// Main use: adopting the Spaces an agency already created in the console, and
// filing scenarios into them by name without copying identifiers around.
func NewSpacesDataSource() datasource.DataSource { return &spacesDataSource{} }

type spacesDataSource struct {
	client *client.Client
}

type spacesDataSourceModel struct {
	MaxSpaces types.Int64      `tfsdk:"max_spaces"`
	Spaces    []spaceListEntry `tfsdk:"spaces"`
}

type spaceListEntry struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Slug          types.String `tfsdk:"slug"`
	Kind          types.String `tfsdk:"kind"`
	StatusPublic  types.Bool   `tfsdk:"status_public"`
	ScenarioCount types.Int64  `tfsdk:"scenario_count"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (d *spacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spaces"
}

func (d *spacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Active Spaces of the organization, with the limit of the plan. " +
			"A key limited to a Space only sees its own.",
		Attributes: map[string]schema.Attribute{
			"max_spaces": schema.Int64Attribute{
				Computed:    true,
				Description: "Active Spaces the plan allows. Null when the plan sets no limit.",
			},
			"spaces": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Active Spaces, sorted by name. Archived Spaces are not listed.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Computed: true, Description: "Identifier of the Space."},
						"name":           schema.StringAttribute{Computed: true, Description: "Name of the Space."},
						"slug":           schema.StringAttribute{Computed: true, Description: "Address of the Space, used by its status page."},
						"kind":           schema.StringAttribute{Computed: true, Description: "`client`, `site` or `application`."},
						"status_public":  schema.BoolAttribute{Computed: true, Description: "Whether the status page of the Space is published."},
						"scenario_count": schema.Int64Attribute{Computed: true, Description: "Scenarios filed in the Space."},
						"created_at":     schema.StringAttribute{Computed: true, Description: "Creation date."},
					},
				},
			},
		},
	}
}

func (d *spacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *spacesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}
	got, err := d.client.ListSpaces(ctx)
	if client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Spaces are not open for this organization",
			"The API answers 404 on /v1/spaces until Spaces are enabled for the organization. "+
				"An empty list here would read as \"no client\" and feed a `for_each` that destroys.",
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the Spaces", err.Error())
		return
	}
	model := spacesDataSourceModel{MaxSpaces: int64From(got.MaxSpaces)}
	for i := range got.Items {
		s := got.Items[i]
		model.Spaces = append(model.Spaces, spaceListEntry{
			ID:            types.StringValue(s.ID),
			Name:          types.StringValue(s.Name),
			Slug:          stringFrom(s.Slug),
			Kind:          stringFrom(s.Kind),
			StatusPublic:  boolFrom(s.StatusPublic),
			ScenarioCount: int64From(s.Sites),
			CreatedAt:     stringFrom(s.CreatedAt),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
