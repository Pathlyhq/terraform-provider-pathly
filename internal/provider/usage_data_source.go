package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// NewUsageDataSource declares `pathly_usage`.
//
// Useful to make a configuration refuse to grow past what the plan covers: a
// `precondition` on the remaining runs fails the plan before creating the
// scenario that would have exceeded the quota, instead of discovering it on the
// invoice.
func NewUsageDataSource() datasource.DataSource { return &usageDataSource{} }

type usageDataSource struct {
	client *client.Client
}

type usageModel struct {
	PlanID                 types.String  `tfsdk:"plan_id"`
	BrowserRunsUsed        types.Float64 `tfsdk:"browser_runs_used"`
	PackRunsUsedThisPeriod types.Float64 `tfsdk:"pack_runs_used_this_period"`
	PackRunsRemaining      types.Float64 `tfsdk:"pack_runs_remaining"`
}

func (d *usageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_usage"
}

func (d *usageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Consumption of the running period.",
		Attributes: map[string]schema.Attribute{
			"plan_id": schema.StringAttribute{
				Computed:    true,
				Description: "Active plan.",
			},
			// Numbers rather than integers: the API derives them from decimal
			// counters, and rounding them here would hide a partial
			// consumption.
			"browser_runs_used": schema.Float64Attribute{
				Computed:    true,
				Description: "Browser runs consumed over the period.",
			},
			"pack_runs_used_this_period": schema.Float64Attribute{
				Computed:    true,
				Description: "Runs taken from the packs over the period.",
			},
			"pack_runs_remaining": schema.Float64Attribute{
				Computed:    true,
				Description: "Runs still available in the packs.",
			},
		},
	}
}

func (d *usageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *usageDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}
	got, err := d.client.GetUsage(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot read the consumption", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, usageModel{
		PlanID:                 types.StringValue(got.PlanID),
		BrowserRunsUsed:        numberFrom(got.BrowserRunsUsed),
		PackRunsUsedThisPeriod: numberFrom(got.PackRunsUsedThisPeriod),
		PackRunsRemaining:      numberFrom(got.PackRunsRemaining),
	})...)
}
