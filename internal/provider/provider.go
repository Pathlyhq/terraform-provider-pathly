// Package provider exposes the Pathly resources to Terraform.
//
// Core choice: the provider only writes what makes sense as declarative
// infrastructure — scenarios, maintenance windows, webhooks, SLA targets and the
// organization settings. API keys are not there, the API refuses them to a key,
// and members are readable but not writable: a Terraform file that invites an
// owner turns a repository write into an access grant.
package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// EnvToken is the recommended environment variable for the token.
const EnvToken = "PATHLY_API_TOKEN"

// EnvBaseURL makes it possible to aim at a preproduction without editing code.
const EnvBaseURL = "PATHLY_API_URL"

// New builds the provider. The version is injected at build time.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &pathlyProvider{version: version}
	}
}

type pathlyProvider struct {
	version string
}

type providerModel struct {
	APIURL   types.String `tfsdk:"api_url"`
	APIToken types.String `tfsdk:"api_token"`
}

func (p *pathlyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pathly"
	resp.Version = p.version
}

func (p *pathlyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Pathly synthetic monitoring: scenarios, maintenance windows, webhooks and SLA targets.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "API base. Defaults to " + client.DefaultBaseURL + ", or to the " + EnvBaseURL + " variable.",
			},
			"api_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Organization API key, `sp_` prefix. " +
					"Better supplied through the " + EnvToken + " variable: written into a `.tf` file, " +
					"it ends up in the repository, and reading it from a Terraform variable puts it in the state in plaintext.",
			},
		},
	}
}

// resolveBaseURL picks the API being aimed at: environment, then configuration,
// then production. Extracted from Configure so it can be checked without a
// network call.
func resolveBaseURL(configured types.String) string {
	if fromEnv := strings.TrimSpace(os.Getenv(EnvBaseURL)); fromEnv != "" {
		return fromEnv
	}
	if !configured.IsNull() {
		if value := strings.TrimSpace(configured.ValueString()); value != "" {
			return value
		}
	}
	return client.DefaultBaseURL
}

func (p *pathlyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The environment comes before the configuration: this is the path that
	// leaves no trace in the repository, it has to be the easiest one to take.
	token := strings.TrimSpace(os.Getenv(EnvToken))
	if token == "" && !config.APIToken.IsNull() {
		token = strings.TrimSpace(config.APIToken.ValueString())
	}
	baseURL := resolveBaseURL(config.APIURL)

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Pathly API key",
			"Set the "+EnvToken+" environment variable, or the `api_token` attribute of the provider. "+
				"The key is created under Settings then API keys, from an owner or administrator account.",
		)
		return
	}
	if !strings.HasPrefix(token, "sp_") {
		// Immediate and explicit refusal: without it, the error would only show
		// up on the first call, as an indistinct 401.
		resp.Diagnostics.AddError(
			"Malformed Pathly API key",
			"A Pathly key starts with `sp_`. Check that the value was not truncated or mistaken for another token.",
		)
		return
	}
	if !strings.HasPrefix(baseURL, "https://") && !isLocalHTTP(baseURL) {
		// The key travels in the Authorization header: in plaintext, it is
		// readable along the way. `localhost` stays tolerated for local
		// development.
		resp.Diagnostics.AddError(
			"Plaintext Pathly API",
			"`api_url` must use https, except on localhost. The API key travels in the Authorization header.",
		)
		return
	}

	c := client.New(baseURL, token, client.WithUserAgent("terraform-provider-pathly/"+p.version))
	if err := c.Ping(ctx); err != nil {
		// Failing once at configuration time rather than once per resource
		// during the plan: the message stays readable.
		resp.Diagnostics.AddError("Pathly API unreachable or key refused", err.Error())
		return
	}

	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *pathlyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewScenarioResource,
		NewMaintenanceWindowResource,
		NewWebhookResource,
		NewSlaTargetResource,
		NewSettingsResource,
	}
}

func (p *pathlyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewScenariosDataSource,
	}
}

// isLocalHTTP tolerates plaintext local development, and nothing else.
//
// The API key travels in the Authorization header: towards a remote host over
// http, it is readable by every intermediary.
func isLocalHTTP(baseURL string) bool {
	return strings.HasPrefix(baseURL, "http://localhost") ||
		strings.HasPrefix(baseURL, "http://127.0.0.1")
}

// clientFrom fetches the shared client, refusing a badly configured provider
// clearly rather than dereferencing nil in the middle of an apply.
func clientFrom(data any, diags interface{ AddError(string, string) }) *client.Client {
	if data == nil {
		return nil
	}
	c, ok := data.(*client.Client)
	if !ok {
		diags.AddError(
			"Pathly provider incorrectly initialized",
			"The expected client was not passed through. Please report this anomaly, it does not come from your configuration.",
		)
		return nil
	}
	return c
}
