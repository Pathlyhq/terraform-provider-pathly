package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewMembersDataSource declares `pathly_members`.
//
// Reading only, deliberately. Inviting a member from a Terraform file would turn
// a write on a repository into an access grant: whoever can open a merge request
// would be able to add themselves. The reading side, on the other hand, is what
// lets a configuration assign an escalation address to someone who really is
// part of the organization.
func NewMembersDataSource() datasource.DataSource { return &membersDataSource{} }

type membersDataSource struct {
	client *client.Client
}

type membersDataSourceModel struct {
	FilterRole types.String  `tfsdk:"filter_role"`
	Members    []memberEntry `tfsdk:"members"`
}

type memberEntry struct {
	ID    types.String `tfsdk:"id"`
	Email types.String `tfsdk:"email"`
	Name  types.String `tfsdk:"name"`
	Role  types.String `tfsdk:"role"`
}

func (d *membersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_members"
}

func (d *membersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Members of the organization, read-only.",
		Attributes: map[string]schema.Attribute{
			"filter_role": schema.StringAttribute{
				Optional: true,
				Description: "Keeps only the members holding this role. " +
					"The accepted values follow the Pathly roles, so they are not validated here: " +
					"a new role must not break a configuration that is already correct.",
			},
			"members": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Members found, in the order returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":    schema.StringAttribute{Computed: true, Description: "Identifier of the member."},
						"email": schema.StringAttribute{Computed: true, Description: "Email address."},
						"name":  schema.StringAttribute{Computed: true, Description: "Display name."},
						"role":  schema.StringAttribute{Computed: true, Description: "Role held in the organization."},
					},
				},
			},
		},
	}
}

func (d *membersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *membersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config membersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}

	all, err := d.client.ListMembers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the members", err.Error())
		return
	}

	role := strings.TrimSpace(config.FilterRole.ValueString())
	config.Members = nil
	for i := range all {
		m := all[i]
		if role != "" && (m.Role == nil || *m.Role != role) {
			continue
		}
		config.Members = append(config.Members, memberEntry{
			ID:    types.StringValue(m.ID),
			Email: stringFrom(m.Email),
			Name:  stringFrom(m.Name),
			Role:  stringFrom(m.Role),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
