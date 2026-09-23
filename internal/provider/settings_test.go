package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// numList builds a list of numbers the way strList builds a list of strings.
func numList(values ...int64) tftypes.Value {
	elements := make([]tftypes.Value, 0, len(values))
	for _, v := range values {
		elements = append(elements, num(v))
	}
	return tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, elements)
}

/*
The settings, the only singleton of the provider.

Two properties carry the whole design. The settings exist before Terraform, so
Create must not reset what the console has already set — an apply that emptied
the alert address would cut the alerting without any error. And they survive
Terraform, so Delete must not attempt anything.
*/

func TestSettingsAlignWithoutErasingWhatWasNotAsked(t *testing.T) {
	ctx := context.Background()
	r := NewSettingsResource()

	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPatch || req.URL.Path != "/v1/settings" {
			t.Errorf("call = %s %s", req.Method, req.URL.Path)
		}
		_, _ = w.Write([]byte(`{"settings":{"name":"Acme","planId":"pro","timezone":"Europe/Paris",
		  "alertEmail":"ops@example.com","alertOnRecovery":true,"escalationAfterFails":3,
		  "sslWarnDays":[30,14,7],"domainWarnDays":[],"triageEnabled":true,
		  "triageLatencyFactor":"2.5","hasSlackWebhook":true}}`))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"timezone":      str("Europe/Paris"),
		"alert_email":   str("ops@example.com"),
		"ssl_warn_days": numList(30, 14, 7),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	// Only what the configuration says travels. Sending the untouched
	// attributes as zeroes would erase, on the first apply, everything set from
	// the console.
	sent := h.bodies[0]
	for _, absent := range []string{"statusSlug", "statusPublic", "weeklyDigestEmail", "escalationEmail"} {
		if _, present := sent[absent]; present {
			t.Errorf("%s must not be sent: nothing was asked about it", absent)
		}
	}
	if sent["timezone"] != "Europe/Paris" || sent["alertEmail"] != "ops@example.com" {
		t.Errorf("sent = %v", sent)
	}

	var state settingsModel
	resp.State.Get(ctx, &state)
	if state.ID.ValueString() != settingsID {
		t.Errorf("id = %q, the singleton always carries the same one", state.ID.ValueString())
	}
	// What the API answers takes over, including the attributes the
	// configuration never mentioned.
	if state.EscalationAfterFails.ValueInt64() != 3 || !state.TriageEnabled.ValueBool() {
		t.Errorf("state = %+v", state)
	}
	if state.TriageLatencyFactor.ValueFloat64() != 2.5 {
		t.Errorf("triage latency factor = %v", state.TriageLatencyFactor)
	}
	if !state.HasSlackWebhook.ValueBool() {
		t.Error("the Slack flag must be reported")
	}
	// An empty list and an absent list are not the same thing: one says "no
	// threshold", the other says "not managed here".
	if state.DomainWarnDays.IsNull() || len(state.DomainWarnDays.Elements()) != 0 {
		t.Errorf("domain_warn_days = %v, want an empty list", state.DomainWarnDays)
	}
}

func TestSettingsCreateSendsAnIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	r := NewSettingsResource()

	var key string
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, req *http.Request) {
		key = req.Header.Get("Idempotency-Key")
		_, _ = w.Write([]byte(`{"settings":{"timezone":"Europe/Paris"}}`))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"timezone": str("Europe/Paris"),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	// A retry after a network timeout must not apply the change twice.
	if key == "" {
		t.Error("Create must carry an idempotency key")
	}
}

func TestSettingsReadRefreshesTheState(t *testing.T) {
	ctx := context.Background()
	r := NewSettingsResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			t.Errorf("Read must not write: %s", req.Method)
		}
		_, _ = w.Write([]byte(`{"settings":{"timezone":"UTC","statusSlug":"acme","statusPublic":true}}`))
	})

	resp := &resource.ReadResponse{State: h.emptyState()}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{
		"id":       str(settingsID),
		"timezone": str("Europe/Paris"),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state settingsModel
	resp.State.Get(ctx, &state)
	// A change made from the console has to show up as a drift on the next plan.
	if state.Timezone.ValueString() != "UTC" {
		t.Errorf("timezone = %q", state.Timezone.ValueString())
	}
	if state.StatusSlug.ValueString() != "acme" || !state.StatusPublic.ValueBool() {
		t.Errorf("status page = %+v", state)
	}
}

func TestSettingsUpdateKeepsTheSingleton(t *testing.T) {
	ctx := context.Background()
	r := NewSettingsResource()

	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"settings":{"statusSlug":"acme","statusPublic":true,"timezone":"UTC"}}`))
	})

	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan: h.plan(map[string]tftypes.Value{
			"status_slug":   str("acme"),
			"status_public": boolean(true),
		}),
		State: h.state(map[string]tftypes.Value{"id": str(settingsID)}),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if sent := h.bodies[0]; sent["statusSlug"] != "acme" || sent["statusPublic"] != true {
		t.Errorf("sent = %v", sent)
	}

	var state settingsModel
	resp.State.Get(ctx, &state)
	if state.ID.ValueString() != settingsID {
		t.Errorf("id = %q", state.ID.ValueString())
	}
}

// A settings object that no longer answers has been removed from underneath
// Terraform, which happens when the organization itself is gone. The resource
// leaves the state rather than failing every plan from then on.
func TestSettingsReadForgetsWhatTheAPINoLongerKnows(t *testing.T) {
	ctx := context.Background()
	r := NewSettingsResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Organisation introuvable"}`))
	})

	resp := &resource.ReadResponse{State: h.state(map[string]tftypes.Value{"id": str(settingsID)})}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{"id": str(settingsID)})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("settings that no longer exist must leave the state")
	}
}

// Delete is the sensitive one. There is no endpoint to remove the settings, and
// a `terraform destroy` on a test workspace must not cut the alerting of the
// whole organization.
func TestSettingsDeleteTouchesNothingAndSaysSo(t *testing.T) {
	ctx := context.Background()
	r := NewSettingsResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, req *http.Request) {
		t.Errorf("Delete must call nothing: %s %s", req.Method, req.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})

	resp := &resource.DeleteResponse{State: h.state(map[string]tftypes.Value{"id": str(settingsID)})}
	r.Delete(ctx, resource.DeleteRequest{State: h.state(map[string]tftypes.Value{"id": str(settingsID)})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete must not fail: %v", resp.Diagnostics)
	}
	// Silence would be worse than a failure: the operator has to know the
	// settings are still live.
	if resp.Diagnostics.WarningsCount() == 0 {
		t.Error("Delete must warn that the settings stay in place")
	}
}

func TestSettingsImportIgnoresWhatWasTypedOnTheCommandLine(t *testing.T) {
	ctx := context.Background()
	r := NewSettingsResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, req *http.Request) {
		t.Errorf("import must call nothing: %s", req.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})

	resp := &resource.ImportStateResponse{State: h.emptyState()}
	r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: "whatever-was-typed"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", resp.Diagnostics)
	}

	var id types.String
	resp.State.GetAttribute(ctx, pathRootID(), &id)
	// A typo in the identifier must not produce a state pointing at nothing:
	// there is only one settings object.
	if id.ValueString() != settingsID {
		t.Errorf("imported id = %q, want %q", id.ValueString(), settingsID)
	}
}

func TestSettingsReportRefusals(t *testing.T) {
	ctx := context.Background()
	refuse := func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"This key does not carry the org:write scope."}`))
	}

	r := NewSettingsResource()
	h := newHarness(t, r, refuse)
	createResp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{"timezone": str("UTC")})}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Error("a refused Create must surface")
	}

	r = NewSettingsResource()
	h = newHarness(t, r, refuse)
	readResp := &resource.ReadResponse{State: h.emptyState()}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{"id": str(settingsID)})}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Error("a refused Read must surface")
	}

	r = NewSettingsResource()
	h = newHarness(t, r, refuse)
	updateResp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  h.plan(map[string]tftypes.Value{"timezone": str("UTC")}),
		State: h.state(map[string]tftypes.Value{"id": str(settingsID)}),
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Error("a refused Update must surface")
	}
}

// Same reasoning as the scenario tags: a list of days that does not hold
// numbers cannot come from Terraform, but it can come from a state written by an
// older provider. The conversion must fail before the call rather than send an
// empty list, which would drop every certificate expiry warning.
func TestSettingsRefuseWarnDaysThatAreNotNumbers(t *testing.T) {
	ctx := context.Background()
	r := NewSettingsResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		t.Error("no call: the conversion of the days failed beforehand")
		w.WriteHeader(http.StatusInternalServerError)
	})

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	forged := schemaResp.Schema
	attributes := map[string]fwschema.Attribute{}
	for name, attribute := range forged.Attributes {
		attributes[name] = attribute
	}
	attributes["ssl_warn_days"] = fwschema.ListAttribute{ElementType: types.StringType, Optional: true}
	forged.Attributes = attributes
	forgedType := forged.Type().TerraformType(ctx).(tftypes.Object)

	raw := object(forgedType, map[string]tftypes.Value{
		"ssl_warn_days": strList("thirty"),
	})

	createResp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: forged, Raw: raw}}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Error("Create must refuse days that are not numbers")
	}

	updateResp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: forged, Raw: raw},
		State: h.state(map[string]tftypes.Value{"id": str(settingsID)}),
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Error("Update must refuse days that are not numbers")
	}
}

// The status page is the one setting that publishes something. Turning it on
// must say what becomes visible, because the scenario names often name internal
// hosts.
func TestStatusPublicWarnsAboutWhatItPublishes(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	NewSettingsResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)

	description := schemaResp.Schema.Attributes["status_public"].GetDescription()
	for _, expected := range []string{"scenario names", "internal"} {
		if !strings.Contains(description, expected) {
			t.Errorf("status_public description = %q, it must mention %q", description, expected)
		}
	}
}

// ---------------------------------------------------------------------------
// Conversions used only by the settings
// ---------------------------------------------------------------------------

func TestNumberListsCrossBothWays(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Absent stays absent, empty stays empty: the difference is what keeps a
	// plan from showing a change on every run.
	if got := int64sFrom(nil); !got.IsNull() {
		t.Errorf("an absent list must stay null: %v", got)
	}
	empty := int64sFrom([]int64{})
	if empty.IsNull() || len(empty.Elements()) != 0 {
		t.Errorf("an empty list must stay an empty list: %v", empty)
	}

	full := int64sFrom([]int64{30, 14, 7})
	back := int64sTo(ctx, full, &diags)
	if diags.HasError() {
		t.Fatalf("round trip: %v", diags)
	}
	if len(back) != 3 || back[0] != 30 || back[2] != 7 {
		t.Errorf("round trip = %v", back)
	}

	// Null and unknown both mean "nothing to send", not "send an empty list".
	if got := int64sTo(ctx, types.ListNull(types.Int64Type), &diags); got != nil {
		t.Errorf("null = %v", got)
	}
	if got := int64sTo(ctx, types.ListUnknown(types.Int64Type), &diags); got != nil {
		t.Errorf("unknown = %v", got)
	}
	if diags.HasError() {
		t.Errorf("neither case is an error: %v", diags)
	}
}

func TestNumberFromKeepsAbsenceApart(t *testing.T) {
	if got := numberFrom(nil); !got.IsNull() {
		t.Errorf("an unmeasured number must stay null: %v", got)
	}
	measured := client.Number(0)
	if got := numberFrom(&measured); got.IsNull() || got.ValueFloat64() != 0 {
		t.Errorf("a measured zero must stay a zero: %v", got)
	}
}
