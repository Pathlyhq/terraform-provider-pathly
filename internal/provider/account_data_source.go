package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewAccountDataSource declares `pathly_account`.
//
// It answers the two questions a configuration asks before growing: which
// organization (and which Space) the key belongs to, and how much room the plan
// leaves. A `precondition` on `quotas.flow.remaining` fails the plan before the
// Flow that would exceed the plan is created, instead of failing half an apply.
func NewAccountDataSource() datasource.DataSource { return &accountDataSource{} }

type accountDataSource struct {
	client *client.Client
}

type quotaModel struct {
	Used      types.Float64 `tfsdk:"used"`
	Limit     types.Float64 `tfsdk:"limit"`
	Remaining types.Float64 `tfsdk:"remaining"`
}

type periodQuotaModel struct {
	Used      types.Float64 `tfsdk:"used"`
	Limit     types.Float64 `tfsdk:"limit"`
	Remaining types.Float64 `tfsdk:"remaining"`
	Period    types.String  `tfsdk:"period"`
}

type runsQuotaModel struct {
	Used              types.Float64 `tfsdk:"used"`
	Limit             types.Float64 `tfsdk:"limit"`
	Remaining         types.Float64 `tfsdk:"remaining"`
	Period            types.String  `tfsdk:"period"`
	PackRunsRemaining types.Float64 `tfsdk:"pack_runs_remaining"`
}

type accountQuotasModel struct {
	Ping            quotaModel       `tfsdk:"ping"`
	Chain           quotaModel       `tfsdk:"chain"`
	Flow            quotaModel       `tfsdk:"flow"`
	Runs            runsQuotaModel   `tfsdk:"runs"`
	AssistantTrials periodQuotaModel `tfsdk:"assistant_trials"`
	Spaces          *quotaModel      `tfsdk:"spaces"`
}

type accountModel struct {
	OrganizationID      types.String       `tfsdk:"organization_id"`
	OrganizationName    types.String       `tfsdk:"organization_name"`
	SpaceID             types.String       `tfsdk:"space_id"`
	SpaceName           types.String       `tfsdk:"space_name"`
	PlanID              types.String       `tfsdk:"plan_id"`
	PlanName            types.String       `tfsdk:"plan_name"`
	PlanPriceMonthlyEur types.Float64      `tfsdk:"plan_price_monthly_eur"`
	BillingURL          types.String       `tfsdk:"billing_url"`
	Quotas              accountQuotasModel `tfsdk:"quotas"`
}

func (d *accountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

// quotaAttributes are the three counters every quota carries. Numbers rather
// than integers, as in `pathly_usage`: the runs counter comes from a decimal
// column, and one type for every quota keeps the comparisons simple.
func quotaAttributes(extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"used": schema.Float64Attribute{
			Computed: true,
			Description: "Consumed. For a key limited to a Space, only what that Space consumes, " +
				"while limit and remaining stay those of the organization.",
		},
		"limit": schema.Float64Attribute{
			Computed:    true,
			Description: "What the plan allows. Null when the plan sets no quota.",
		},
		"remaining": schema.Float64Attribute{
			Computed:    true,
			Description: "What the organization has left. Null when the plan sets no quota.",
		},
	}
	for name, attr := range extra {
		attrs[name] = attr
	}
	return attrs
}

func (d *accountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	period := schema.StringAttribute{Computed: true, Description: "Period after which the counter resets: `month` or `day`."}
	resp.Schema = schema.Schema{
		Description: "Organization and Space of the key, active plan, quotas used and remaining.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Computed: true,
				Description: "Identifier of the organization. It is also the identifier of its status page, " +
					"to import into `pathly_status_page`.",
			},
			"organization_name": schema.StringAttribute{Computed: true, Description: "Name of the organization."},
			"space_id": schema.StringAttribute{
				Computed:    true,
				Description: "Space the key is limited to. Null for a key of the whole organization.",
			},
			"space_name":             schema.StringAttribute{Computed: true, Description: "Name of that Space, null otherwise."},
			"plan_id":                schema.StringAttribute{Computed: true, Description: "Active plan."},
			"plan_name":              schema.StringAttribute{Computed: true, Description: "Display name of the plan."},
			"plan_price_monthly_eur": schema.Float64Attribute{Computed: true, Description: "Monthly list price of the plan, in euros excluding VAT."},
			"billing_url":            schema.StringAttribute{Computed: true, Description: "Plan screen of the app, to upgrade or buy runs."},
			"quotas": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Quotas of the plan.",
				Attributes: map[string]schema.Attribute{
					"ping": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "Active Ping, scheduled heartbeats included.",
						Attributes:  quotaAttributes(nil),
					},
					"chain": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "Active Chain.",
						Attributes:  quotaAttributes(nil),
					},
					"flow": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "Active Flow.",
						Attributes:  quotaAttributes(nil),
					},
					"runs": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "Browser runs of the month.",
						Attributes: quotaAttributes(map[string]schema.Attribute{
							"period": period,
							"pack_runs_remaining": schema.Float64Attribute{
								Computed:    true,
								Description: "Runs still available in the packs, on top of the plan.",
							},
						}),
					},
					"assistant_trials": schema.SingleNestedAttribute{
						Computed:    true,
						Description: "Flow assistant trials of the day.",
						Attributes:  quotaAttributes(map[string]schema.Attribute{"period": period}),
					},
					"spaces": schema.SingleNestedAttribute{
						Computed: true,
						Description: "Active Spaces. Null for a key limited to a Space, and while Spaces " +
							"are not open for the organization.",
						Attributes: quotaAttributes(nil),
					},
				},
			},
		},
	}
}

func (d *accountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func quotaFrom(q client.Quota) quotaModel {
	return quotaModel{Used: numberFrom(q.Used), Limit: numberFrom(q.Limit), Remaining: numberFrom(q.Remaining)}
}

func (d *accountDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}
	got, err := d.client.GetAccount(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the account", err.Error())
		return
	}

	q := got.Quotas
	model := accountModel{
		OrganizationID:      types.StringValue(got.Organization.ID),
		OrganizationName:    types.StringValue(got.Organization.Name),
		SpaceID:             types.StringNull(),
		SpaceName:           types.StringNull(),
		PlanID:              types.StringValue(got.Plan.ID),
		PlanName:            types.StringValue(got.Plan.Name),
		PlanPriceMonthlyEur: numberFrom(got.Plan.PriceMonthlyEur),
		BillingURL:          types.StringValue(got.BillingURL),
		Quotas: accountQuotasModel{
			Ping:  quotaFrom(q.Ping),
			Chain: quotaFrom(q.Chain),
			Flow:  quotaFrom(q.Flow),
			Runs: runsQuotaModel{
				Used:              numberFrom(q.Runs.Used),
				Limit:             numberFrom(q.Runs.Limit),
				Remaining:         numberFrom(q.Runs.Remaining),
				Period:            stringFrom(q.Runs.Period),
				PackRunsRemaining: numberFrom(q.Runs.PackRunsRemaining),
			},
			AssistantTrials: periodQuotaModel{
				Used:      numberFrom(q.AssistantTrials.Used),
				Limit:     numberFrom(q.AssistantTrials.Limit),
				Remaining: numberFrom(q.AssistantTrials.Remaining),
				Period:    stringFrom(q.AssistantTrials.Period),
			},
		},
	}
	if got.Space != nil {
		model.SpaceID = types.StringValue(got.Space.ID)
		model.SpaceName = types.StringValue(got.Space.Name)
	}
	if q.Spaces != nil {
		spaces := quotaFrom(*q.Spaces)
		model.Quotas.Spaces = &spaces
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
