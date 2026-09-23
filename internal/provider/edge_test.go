package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

/*
Guards that protect the Terraform state.

Two families of cases, invisible in acceptance tests but costly for real: a
state the provider cannot read back — a file written by an earlier version, or
edited by hand — and a refusal from the API on an operation the happy tests do
not cover. In both cases the rule is the same: nothing goes out towards the API,
and the error surfaces instead of leaving a mute state.
*/

// corruptState builds a state the schema cannot decode.
func (h *harness) corruptState() tfsdk.State {
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.Bool}}
	return tfsdk.State{
		Schema: h.schema,
		Raw:    tftypes.NewValue(objType, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.Bool, true)}),
	}
}

func (h *harness) corruptPlan() tfsdk.Plan {
	return tfsdk.Plan{Schema: h.corruptState().Schema, Raw: h.corruptState().Raw}
}

func TestUnreadableStateStopsBeforeAnyCall(t *testing.T) {
	ctx := context.Background()
	resources := map[string]func() resource.Resource{
		"scenario":           NewScenarioResource,
		"maintenance_window": NewMaintenanceWindowResource,
		"webhook":            NewWebhookResource,
		"sla_target":         NewSlaTargetResource,
		"settings":           NewSettingsResource,
	}

	for name, factory := range resources {
		r := factory()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			t.Errorf("%s: no call must go out from an unreadable state", name)
			w.WriteHeader(http.StatusInternalServerError)
		})

		createResp := &resource.CreateResponse{State: h.emptyState()}
		r.Create(ctx, resource.CreateRequest{Plan: h.corruptPlan()}, createResp)
		if !createResp.Diagnostics.HasError() {
			t.Errorf("%s: Create on an unreadable plan must fail", name)
		}

		readResp := &resource.ReadResponse{State: h.emptyState()}
		r.Read(ctx, resource.ReadRequest{State: h.corruptState()}, readResp)
		if !readResp.Diagnostics.HasError() {
			t.Errorf("%s: Read on an unreadable state must fail", name)
		}

		deleteResp := &resource.DeleteResponse{State: h.emptyState()}
		r.Delete(ctx, resource.DeleteRequest{State: h.corruptState()}, deleteResp)
		if name == "settings" {
			// The settings are the exception: there is nothing to delete, so
			// Delete reads no state and an unreadable one cannot stop something
			// that was never going to happen.
			if deleteResp.Diagnostics.HasError() {
				t.Errorf("settings: Delete must not fail: %v", deleteResp.Diagnostics)
			}
			continue
		}
		if !deleteResp.Diagnostics.HasError() {
			// A silent deletion would make a resource that keeps running, and
			// keeps billing, disappear from the state.
			t.Errorf("%s: Delete on an unreadable state must fail", name)
		}
	}
}

func TestUnreadablePlanStopsUpdateWhereItExists(t *testing.T) {
	ctx := context.Background()
	// Windows and webhooks refuse every modification: their Update reads
	// neither plan nor state.
	for name, factory := range map[string]func() resource.Resource{
		"scenario":   NewScenarioResource,
		"sla_target": NewSlaTargetResource,
		"settings":   NewSettingsResource,
	} {
		r := factory()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			t.Errorf("%s: no call must go out", name)
			w.WriteHeader(http.StatusInternalServerError)
		})

		resp := &resource.UpdateResponse{State: h.emptyState()}
		r.Update(ctx, resource.UpdateRequest{Plan: h.corruptPlan(), State: h.corruptState()}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("%s: Update on an unreadable plan must fail", name)
		}
	}
}

// A plan whose tags are not strings cannot come from Terraform, but it can come
// from a state written by an older provider. The conversion must then fail
// before the call, not send an empty list that would erase every tag of the
// scenario.
func TestScenarioRefusesTagsThatAreNotStrings(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		t.Error("no call: the conversion of the tags failed beforehand")
		w.WriteHeader(http.StatusInternalServerError)
	})

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	forged := schemaResp.Schema
	attributes := map[string]fwschema.Attribute{}
	for name, attribute := range forged.Attributes {
		attributes[name] = attribute
	}
	attributes["tags"] = fwschema.ListAttribute{ElementType: types.Int64Type, Optional: true}
	forged.Attributes = attributes
	forgedType := forged.Type().TerraformType(ctx).(tftypes.Object)

	raw := object(forgedType, map[string]tftypes.Value{
		"name":         str("Checkout"),
		"url":          str("https://example.com"),
		"interval_sec": num(60),
		"tags": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Number},
			[]tftypes.Value{num(42)},
		),
	})

	createResp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: forged, Raw: raw},
	}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Error("Create must refuse tags that are not textual")
	}

	updateResp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: forged, Raw: raw},
		State: h.state(map[string]tftypes.Value{"id": str("mon-1"), "name": str("Checkout")}),
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Error("Update must refuse tags that are not textual")
	}
}

// ---------------------------------------------------------------------------
// Refusals from the API on the paths the happy tests do not walk
// ---------------------------------------------------------------------------

func TestScenarioUpdateReportsRefusal(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"This key does not carry the scenarios:write scope."}`))
	})

	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan: h.plan(map[string]tftypes.Value{
			"name":         str("Checkout"),
			"url":          str("https://example.com"),
			"interval_sec": num(60),
		}),
		State: h.state(map[string]tftypes.Value{"id": str("mon-1"), "name": str("Checkout")}),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused modification must surface")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "scenarios:write") {
		t.Errorf("diagnostics = %v", resp.Diagnostics)
	}
}

// Mistaking a refusal for a disappearance would remove the resource from the
// state, and the next apply would create a second one next to the one that
// already exists.
func TestReadRefusalsAreNotTakenForDeletions(t *testing.T) {
	ctx := context.Background()
	cases := map[string]struct {
		factory func() resource.Resource
		state   map[string]tftypes.Value
	}{
		"maintenance_window": {NewMaintenanceWindowResource, map[string]tftypes.Value{"id": str("mw-1")}},
		"webhook": {NewWebhookResource, map[string]tftypes.Value{
			"id": str("h1"), "url": str("https://hooks.example.com"),
		}},
		"sla_target": {NewSlaTargetResource, map[string]tftypes.Value{"id": str("sla-1")}},
	}

	for name, c := range cases {
		r := c.factory()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"missing scope"}`))
		})

		resp := &resource.ReadResponse{State: h.state(c.state)}
		r.Read(ctx, resource.ReadRequest{State: resp.State}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("%s: a 403 on read must surface", name)
		}
		if resp.State.Raw.IsNull() {
			t.Errorf("%s: the resource must not leave the state on a refusal", name)
		}
	}
}

// A refused deletion that passed for a success would make Terraform forget a
// resource that is still running.
func TestDeleteRefusalsAreReported(t *testing.T) {
	ctx := context.Background()
	cases := map[string]struct {
		factory func() resource.Resource
		state   map[string]tftypes.Value
	}{
		"maintenance_window": {NewMaintenanceWindowResource, map[string]tftypes.Value{"id": str("mw-1")}},
		"webhook": {NewWebhookResource, map[string]tftypes.Value{
			"id": str("h1"), "url": str("https://hooks.example.com"),
		}},
	}

	for name, c := range cases {
		r := c.factory()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"missing scope"}`))
		})

		resp := &resource.DeleteResponse{State: h.emptyState()}
		r.Delete(ctx, resource.DeleteRequest{State: h.state(c.state)}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("%s: a refused deletion must surface", name)
		}
	}
}

func TestScenariosDataSourceRefusesUnreadableConfig(t *testing.T) {
	ctx := context.Background()
	d := NewScenariosDataSource()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("no call must go out from an unreadable configuration")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{
		ProviderData: client.New(srv.URL, "sp_test"),
	}, &datasource.ConfigureResponse{})

	badType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"filter_tag": tftypes.Bool}}
	resp := &datasource.ReadResponse{State: tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, nil),
	}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(badType, map[string]tftypes.Value{
			"filter_tag": tftypes.NewValue(tftypes.Bool, true),
		}),
	}}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("an unreadable filter must be reported, not ignored")
	}
}

// ---------------------------------------------------------------------------
// Conversions
// ---------------------------------------------------------------------------

// An absent value must not go into the request: sending a zero would overwrite
// a setting made from the console.
func TestNullValuesNeverReachTheAPI(t *testing.T) {
	if strPtr(types.StringNull()) != nil || strPtr(types.StringUnknown()) != nil {
		t.Error("a null or unknown string must stay absent")
	}
	if int64Ptr(types.Int64Null()) != nil || boolPtr(types.BoolNull()) != nil {
		t.Error("a null integer or boolean must stay absent")
	}
	if float64Ptr(types.Float64Null()) != nil || float64Ptr(types.Float64Unknown()) != nil {
		t.Error("a null or unknown float must stay absent")
	}
	if got := float64Ptr(types.Float64Value(99.9)); got == nil || *got != 99.9 {
		t.Errorf("float that is set = %v", got)
	}
}

// A value the API does not return becomes null, not zero: a target of 0% or a
// duration of 0 minutes would be a nonsense displayed in the plan.
func TestMissingAPIFieldsBecomeNull(t *testing.T) {
	if !stringFrom(nil).IsNull() || !int64From(nil).IsNull() {
		t.Error("absent string and integer must be null")
	}
	if !boolFrom(nil).IsNull() || !float64From(nil).IsNull() {
		t.Error("absent boolean and float must be null")
	}
	value := 0.8
	if got := float64From(&value); got.ValueFloat64() != 0.8 {
		t.Errorf("float read back = %v", got)
	}
	// `[]` and absence are two different things: the API returns `[]` for
	// "no tag", and converting it into null would make a difference on every
	// plan.
	if !stringsFrom(nil).IsNull() {
		t.Error("an absent slice must give a null list")
	}
	if got := stringsFrom([]string{}); got.IsNull() || len(got.Elements()) != 0 {
		t.Errorf("an empty slice must give an empty, non-null list: %v", got)
	}
}

func TestKnownHelpersResolveUnknown(t *testing.T) {
	ctx := context.Background()
	if !knownString(types.StringUnknown()).IsNull() {
		t.Error("unknown string must become null")
	}
	if !knownInt64(types.Int64Unknown()).IsNull() {
		t.Error("unknown integer must become null")
	}
	if !knownList(ctx, types.ListUnknown(types.StringType)).IsNull() {
		t.Error("unknown list must become null")
	}
	if !knownMap(ctx, types.MapUnknown(types.StringType)).IsNull() {
		t.Error("unknown map must become null")
	}
	if !knownObject(ctx, types.ObjectUnknown(map[string]attr.Type{"username": types.StringType})).IsNull() {
		t.Error("unknown object must become null")
	}
	if got := knownString(types.StringValue("kept")); got.ValueString() != "kept" {
		t.Errorf("known string = %v", got)
	}
	if got := knownInt64(types.Int64Value(3)); got.ValueInt64() != 3 {
		t.Errorf("known integer = %v", got)
	}
	keptList := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")})
	if got := knownList(ctx, keptList); len(got.Elements()) != 1 {
		t.Errorf("known list = %v", got)
	}
	keptMap := types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")})
	if got := knownMap(ctx, keptMap); len(got.Elements()) != 1 {
		t.Errorf("known map = %v", got)
	}
	keptObj := types.ObjectValueMust(map[string]attr.Type{"username": types.StringType}, map[string]attr.Value{"username": types.StringValue("u")})
	if got := knownObject(ctx, keptObj); got.IsNull() {
		t.Errorf("known object became null: %v", got)
	}
}

func TestStringsToRejectsWrongElementType(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	if got := stringsTo(ctx, types.ListNull(types.StringType), &diags); got != nil {
		t.Errorf("null list = %v, expected absent", got)
	}
	if got := stringsTo(ctx, types.ListUnknown(types.StringType), &diags); got != nil {
		t.Errorf("unknown list = %v, expected absent", got)
	}
	stringsTo(ctx, types.ListValueMust(types.Int64Type, []attr.Value{types.Int64Value(1)}), &diags)
	if !diags.HasError() {
		t.Error("a list of integers converted into strings must produce a diagnostic")
	}
}

func TestResolveBaseURLPrefersEnvThenConfigThenProduction(t *testing.T) {
	t.Setenv(EnvBaseURL, "")
	if got := resolveBaseURL(types.StringNull()); got != client.DefaultBaseURL {
		t.Errorf("without configuration, the API aimed at must be production: %q", got)
	}
	if got := resolveBaseURL(types.StringValue("   ")); got != client.DefaultBaseURL {
		t.Errorf("an empty value must not be taken for a URL: %q", got)
	}
	if got := resolveBaseURL(types.StringValue("https://api.preprod.example.com")); got != "https://api.preprod.example.com" {
		t.Errorf("configured URL = %q", got)
	}
	t.Setenv(EnvBaseURL, "https://api.env.example.com")
	if got := resolveBaseURL(types.StringValue("https://api.preprod.example.com")); got != "https://api.env.example.com" {
		t.Errorf("the environment must take precedence: %q", got)
	}
}

func TestConfigureRefusesUnreadableConfiguration(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("no call must go out from an unreadable configuration")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv(EnvToken, "sp_valid")
	t.Setenv(EnvBaseURL, srv.URL)

	p := New("test")()
	schemaResp := &fwprovider.SchemaResponse{}
	p.Schema(ctx, fwprovider.SchemaRequest{}, schemaResp)

	// A type that does not match the schema: this is what a state file or a
	// configuration written by an earlier version produces.
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"api_url": tftypes.Bool}}
	resp := &fwprovider.ConfigureResponse{}
	p.Configure(ctx, fwprovider.ConfigureRequest{Config: tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"api_url": tftypes.NewValue(tftypes.Bool, true),
		}),
	}}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("an unreadable configuration must be reported")
	}
}
