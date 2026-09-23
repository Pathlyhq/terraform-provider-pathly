package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

/*
TestMain cuts the environment off before the whole suite.

Without it, a machine where `PATHLY_API_URL` and `PATHLY_API_TOKEN` are set —
the machine of somebody who has just tried the provider by hand, for example —
sees the configuration tests aim at their real API instead of their simulated
server. The failure is then incomprehensible, and worse, a test can pass for the
wrong reason.
*/
func TestMain(m *testing.M) {
	_ = os.Unsetenv(EnvBaseURL)
	_ = os.Unsetenv(EnvToken)
	os.Exit(m.Run())
}

// configFor builds the provider configuration the way Terraform would pass it,
// in order to exercise Configure without launching a Terraform binary.
func configFor(t *testing.T, p fwprovider.Provider, apiURL, apiToken *string) tfsdk.Config {
	t.Helper()
	schemaResp := &fwprovider.SchemaResponse{}
	p.Schema(context.Background(), fwprovider.SchemaRequest{}, schemaResp)

	value := func(v *string) tftypes.Value {
		if v == nil {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, *v)
	}
	objectType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"api_url":   tftypes.String,
		"api_token": tftypes.String,
	}}
	return tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objectType, map[string]tftypes.Value{
			"api_url":   value(apiURL),
			"api_token": value(apiToken),
		}),
	}
}

func configure(t *testing.T, apiURL, apiToken *string) *fwprovider.ConfigureResponse {
	t.Helper()
	p := New("test")()
	resp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), fwprovider.ConfigureRequest{
		Config: configFor(t, p, apiURL, apiToken),
	}, resp)
	return resp
}

func ptr(s string) *string { return &s }

func pathRootID() path.Path { return path.Root("id") }

func TestMetadata(t *testing.T) {
	resp := &fwprovider.MetadataResponse{}
	New("1.2.3")().Metadata(context.Background(), fwprovider.MetadataRequest{}, resp)
	if resp.TypeName != "pathly" {
		t.Errorf("TypeName = %q: it prefixes every resource", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("Version = %q", resp.Version)
	}
}

func TestProviderSchemaKeepsTokenSensitive(t *testing.T) {
	resp := &fwprovider.SchemaResponse{}
	New("test")().Schema(context.Background(), fwprovider.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema: %v", resp.Diagnostics)
	}

	token, ok := resp.Schema.Attributes["api_token"]
	if !ok {
		t.Fatal("api_token attribute missing")
	}
	// Without this flag, the key would show up in plaintext in the plan output
	// and in the CI logs.
	if !token.IsSensitive() {
		t.Error("api_token must be marked Sensitive")
	}
	if token.IsRequired() {
		t.Error("api_token must stay optional: the environment variable is the recommended path")
	}
}

func TestConfigureRefusesMissingToken(t *testing.T) {
	t.Setenv(EnvToken, "")
	resp := configure(t, ptr("https://api.example.com"), nil)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a provider without a key must fail before the first call")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), EnvToken) {
		t.Errorf("the message must name %s: %v", EnvToken, resp.Diagnostics)
	}
}

func TestConfigureRefusesMalformedToken(t *testing.T) {
	t.Setenv(EnvToken, "")
	resp := configure(t, ptr("https://api.example.com"), ptr("ghp_a_github_token"))
	if !resp.Diagnostics.HasError() {
		t.Fatal("a token without the sp_ prefix must be refused right away")
	}
	// Otherwise the error only arrives on the first call, as a silent 401.
	if !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "Malformed") {
		t.Errorf("diagnostics = %v", resp.Diagnostics)
	}
}

func TestConfigureRefusesPlainHTTPRemote(t *testing.T) {
	t.Setenv(EnvToken, "")
	resp := configure(t, ptr("http://api.example.com"), ptr("sp_token"))
	if !resp.Diagnostics.HasError() {
		t.Fatal("the key travels in Authorization: remote http must be refused")
	}
}

func TestConfigureAcceptsLocalHTTPAndPings(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"planId":"pro"}`))
	}))
	defer srv.Close()

	t.Setenv(EnvToken, "")
	resp := configure(t, ptr(srv.URL), ptr("sp_local_token"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("local configuration refused: %v", resp.Diagnostics)
	}
	// The check made at configuration time gives one clear error once, instead
	// of one error per resource during the plan.
	if path != "/v1/usage" {
		t.Errorf("path called = %q, expected /v1/usage", path)
	}
	if _, ok := resp.ResourceData.(*client.Client); !ok {
		t.Error("the client must be passed to the resources")
	}
	if _, ok := resp.DataSourceData.(*client.Client); !ok {
		t.Error("the client must be passed to the data sources")
	}
}

func TestConfigurePrefersEnvironmentOverConfig(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	t.Setenv(EnvToken, "sp_from_environment")
	t.Setenv(EnvBaseURL, srv.URL)
	resp := configure(t, ptr("https://api.example.com"), ptr("sp_from_file"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	// The environment leaves no trace in the repository: it must take
	// precedence.
	if gotAuth != "Bearer sp_from_environment" {
		t.Errorf("Authorization = %q", gotAuth)
	}
}

func TestConfigureReportsRefusedKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Invalid, revoked or expired API key."}`))
	}))
	defer srv.Close()

	t.Setenv(EnvToken, "")
	resp := configure(t, ptr(srv.URL), ptr("sp_revoked"))
	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused key must make the configuration fail")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), EnvToken) {
		t.Errorf("the message must point at the key: %v", resp.Diagnostics)
	}
}

func TestRegisteredResourcesAreUniqueAndDocumented(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	seen := map[string]bool{}
	for _, factory := range p.Resources(ctx) {
		r := factory()
		meta := &resource.MetadataResponse{}
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pathly"}, meta)
		if !strings.HasPrefix(meta.TypeName, "pathly_") {
			t.Errorf("resource name = %q, it must be prefixed", meta.TypeName)
		}
		if seen[meta.TypeName] {
			t.Errorf("resource %s declared twice", meta.TypeName)
		}
		seen[meta.TypeName] = true

		schemaResp := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Fatalf("%s: %v", meta.TypeName, schemaResp.Diagnostics)
		}
		if schemaResp.Schema.Description == "" {
			t.Errorf("%s without a description: the registry documentation would be empty", meta.TypeName)
		}
		for name, attr := range schemaResp.Schema.Attributes {
			// An attribute without a description produces a mute registry page,
			// and nobody knows what it does nor what it exposes.
			if attr.GetDescription() == "" && attr.GetMarkdownDescription() == "" {
				t.Errorf("%s.%s without a description", meta.TypeName, name)
			}
		}
		// ImportState is indispensable to adopt an existing estate.
		if _, ok := r.(resource.ResourceWithImportState); !ok {
			t.Errorf("%s cannot import itself", meta.TypeName)
		}
	}

	for _, expected := range []string{
		"pathly_scenario",
		"pathly_maintenance_window",
		"pathly_webhook",
		"pathly_sla_target",
	} {
		if !seen[expected] {
			t.Errorf("resource %s missing from the provider", expected)
		}
	}
}

func TestImportPassesIdentifierToState(t *testing.T) {
	ctx := context.Background()
	for _, factory := range New("test")().Resources(ctx) {
		r, ok := factory().(resource.ResourceWithImportState)
		if !ok {
			continue
		}
		schemaResp := &resource.SchemaResponse{}
		factory().Schema(ctx, resource.SchemaRequest{}, schemaResp)
		objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

		resp := &resource.ImportStateResponse{
			State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
		}
		r.ImportState(ctx, resource.ImportStateRequest{ID: "imported-resource"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("import: %v", resp.Diagnostics)
		}

		// Without the identifier in state, the first plan following the import
		// would propose to create the adopted resource a second time.
		var id types.String
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, pathRootID(), &id)...)

		meta := &resource.MetadataResponse{}
		factory().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pathly"}, meta)

		// The settings are the documented exception: the organization holds
		// exactly one, so the identifier typed on the command line is replaced
		// by the fixed one rather than trusted. A typo must not produce a state
		// pointing at nothing.
		want := "imported-resource"
		if meta.TypeName == "pathly_settings" {
			want = settingsID
		}
		if id.ValueString() != want {
			t.Errorf("%s: imported identifier = %q, want %q", meta.TypeName, id.ValueString(), want)
		}
	}
}

func TestWebhookSecretsAreMarkedSensitive(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	NewWebhookResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)

	for _, name := range []string{"url", "secret"} {
		attr, ok := schemaResp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("attribute %s missing", name)
		}
		// The URL points at an internal alerting channel, the secret signs the
		// deliveries: neither must show up in a plan output.
		if !attr.IsSensitive() {
			t.Errorf("pathly_webhook.%s must be Sensitive", name)
		}
	}
}

// TestDataSourcesAreDeclared pins the read surface of the provider.
//
// The expected list is written out rather than derived from the code: a data
// source dropped from the registration list compiles perfectly and disappears
// in silence, breaking every configuration that read it. The names themselves
// are part of the published contract and cannot be changed once released.
func TestDataSourcesAreDeclared(t *testing.T) {
	ctx := context.Background()
	want := map[string]bool{
		"pathly_scenarios":           false,
		"pathly_settings":            false,
		"pathly_usage":               false,
		"pathly_incidents":           false,
		"pathly_members":             false,
		"pathly_runs":                false,
		"pathly_run":                 false,
		"pathly_sla":                 false,
		"pathly_sla_targets":         false,
		"pathly_webhooks":            false,
		"pathly_maintenance_windows": false,
		"pathly_status_page":         false,
	}

	for _, factory := range New("test")().DataSources(ctx) {
		d := factory()
		meta := &datasource.MetadataResponse{}
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "pathly"}, meta)

		seen, expected := want[meta.TypeName]
		if !expected {
			t.Errorf("data source %q is not in the expected list", meta.TypeName)
			continue
		}
		if seen {
			t.Errorf("data source %q declared twice", meta.TypeName)
		}
		want[meta.TypeName] = true

		schemaResp := &datasource.SchemaResponse{}
		d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Fatalf("schema of %s: %v", meta.TypeName, schemaResp.Diagnostics)
		}
		if schemaResp.Schema.Description == "" {
			t.Errorf("%s: no description, the registry would publish an empty page", meta.TypeName)
		}
	}

	for name, seen := range want {
		if !seen {
			t.Errorf("data source %q is no longer registered", name)
		}
	}
}

func TestScenarioSchemaRefusesBrowserType(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	NewScenarioResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)

	attr, ok := schemaResp.Schema.Attributes["type"]
	if !ok {
		t.Fatal("type attribute missing")
	}
	// The steps of a browser journey carry login credentials: managing them in
	// Terraform would put them into the state.
	if !strings.Contains(attr.GetDescription(), "http") {
		t.Errorf("description of type = %q", attr.GetDescription())
	}
	if !strings.Contains(schemaResp.Schema.MarkdownDescription, "Browser") {
		t.Error("the resource must say why browser journeys are excluded")
	}
}

func TestMutedUntilIsReadOnly(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	NewScenarioResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)

	attr := schemaResp.Schema.Attributes["muted_until"]
	// A mute expires on its own: declaring it as a desired state would set the
	// mute again on every apply, or remove it in the middle of an intervention.
	if attr.IsOptional() || attr.IsRequired() {
		t.Error("muted_until must stay read only")
	}
}

func TestClientFromRejectsWrongData(t *testing.T) {
	diags := &collectingDiagnostics{}
	if got := clientFrom("not a client", diags); got != nil {
		t.Error("unexpected data must not be taken for a client")
	}
	if len(diags.errors) == 0 {
		t.Error("the case must be reported rather than silent")
	}
	if got := clientFrom(nil, diags); got != nil {
		t.Error("a nil ProviderData legitimately happens before Configure: no error, no client")
	}
}

type collectingDiagnostics struct{ errors []string }

func (c *collectingDiagnostics) AddError(summary, detail string) {
	c.errors = append(c.errors, summary+" "+detail)
}

func TestIdempotencyKeysAreUniqueAndPrefixed(t *testing.T) {
	first := newIdempotencyKey("scenario")
	second := newIdempotencyKey("scenario")
	if first == second {
		t.Fatal("two creations must not share the same key: the second would be replayed in place of the first")
	}
	if !strings.HasPrefix(first, "tf-scenario-") {
		t.Errorf("key = %q", first)
	}
	if len(first) < 20 {
		t.Errorf("key too short, it would be guessable: %q", first)
	}
}

func TestConversionsIgnoreNullAndUnknown(t *testing.T) {
	if strPtr(types.StringNull()) != nil || strPtr(types.StringUnknown()) != nil {
		t.Error("an absent value must not go into the request")
	}
	if int64Ptr(types.Int64Null()) != nil || boolPtr(types.BoolNull()) != nil {
		t.Error("same for integers and booleans")
	}
	if float64Ptr(types.Float64Unknown()) != nil {
		t.Error("same for floats")
	}
	if got := strPtr(types.StringValue("value")); got == nil || *got != "value" {
		t.Error("a value that is set must go through")
	}
}

func TestListConversionDistinguishesNullFromEmpty(t *testing.T) {
	if !stringsFrom(nil).IsNull() {
		t.Error("a nil slice gives a null list")
	}
	empty := stringsFrom([]string{})
	if empty.IsNull() {
		t.Error("the API returns [] for \"no tag\": the list must be empty, not null")
	}
	if len(empty.Elements()) != 0 {
		t.Error("empty list expected")
	}

	filled := stringsFrom([]string{"prod", "checkout"})
	var back []string
	diags := filled.ElementsAs(context.Background(), &back, false)
	if diags.HasError() {
		t.Fatalf("reverse conversion: %v", diags)
	}
	if len(back) != 2 || back[0] != "prod" {
		t.Errorf("round trip = %v", back)
	}
}

func TestHasTag(t *testing.T) {
	if !hasTag([]string{"a", "b"}, "b") {
		t.Error("tag present not found")
	}
	if hasTag(nil, "b") || hasTag([]string{"a"}, "b") {
		t.Error("absent tag reported as present")
	}
}
