package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwdsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

/*
Complete life cycle of each resource, against a simulated API.

These tests replace Terraform acceptance tests, which would require the
Terraform binary and a real Pathly organization. They cover what breaks for
real: a creation whose response is not stored into the state, a resource deleted
by hand that makes every subsequent plan fail, a deletion that fails because the
object had already disappeared.
*/

// object completes the attributes that are not supplied with nulls, from the
// type of the schema: tftypes demands the exhaustive list, and forgetting it
// makes it panic.
func object(objType tftypes.Object, set map[string]tftypes.Value) tftypes.Value {
	values := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		if v, ok := set[name]; ok {
			values[name] = v
			continue
		}
		values[name] = tftypes.NewValue(attrType, nil)
	}
	return tftypes.NewValue(objType, values)
}

type harness struct {
	schema  fwschema.Schema
	objType tftypes.Object
	server  *httptest.Server
	client  *client.Client
	// requests keeps the method and the path of each call, in order.
	requests []string
	bodies   []map[string]any
}

func newHarness(t *testing.T, r resource.Resource, handler func(h *harness, w http.ResponseWriter, req *http.Request)) *harness {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	h := &harness{schema: schemaResp.Schema}
	h.objType = schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	h.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h.requests = append(h.requests, req.Method+" "+req.URL.Path)
		body := map[string]any{}
		if raw, err := io.ReadAll(req.Body); err == nil && len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		h.bodies = append(h.bodies, body)
		handler(h, w, req)
	}))
	t.Cleanup(h.server.Close)

	h.client = client.New(h.server.URL, "sp_test")
	configureResp := &resource.ConfigureResponse{}
	if rc, ok := r.(resource.ResourceWithConfigure); ok {
		rc.Configure(ctx, resource.ConfigureRequest{ProviderData: h.client}, configureResp)
		if configureResp.Diagnostics.HasError() {
			t.Fatalf("Configure: %v", configureResp.Diagnostics)
		}
	}
	return h
}

func (h *harness) plan(set map[string]tftypes.Value) tfsdk.Plan {
	return tfsdk.Plan{Schema: h.schema, Raw: object(h.objType, set)}
}

func (h *harness) state(set map[string]tftypes.Value) tfsdk.State {
	return tfsdk.State{Schema: h.schema, Raw: object(h.objType, set)}
}

func (h *harness) emptyState() tfsdk.State {
	return tfsdk.State{Schema: h.schema, Raw: tftypes.NewValue(h.objType, nil)}
}

func str(v string) tftypes.Value  { return tftypes.NewValue(tftypes.String, v) }
func num(v int64) tftypes.Value   { return tftypes.NewValue(tftypes.Number, v) }
func flt(v float64) tftypes.Value { return tftypes.NewValue(tftypes.Number, v) }
func boolean(v bool) tftypes.Value {
	return tftypes.NewValue(tftypes.Bool, v)
}
func strList(values ...string) tftypes.Value {
	elements := make([]tftypes.Value, 0, len(values))
	for _, v := range values {
		elements = append(elements, str(v))
	}
	return tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, elements)
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

const scenarioJSON = `{
  "id": "mon-1", "name": "Checkout", "type": "http", "url": "https://shop.example.com/cart",
  "enabled": true, "intervalSec": 300, "method": "GET", "expectedStatus": 200,
  "maxLatencyMs": 2000, "expectText": "Your cart", "regions": ["eu-west"], "tags": ["prod"],
  "folder": "Shop", "severity": "major", "runbook": null, "cron": null,
  "lastStatus": "ok", "mutedUntil": null, "createdAt": "2026-09-22T10:00:00.000Z"
}`

func TestScenarioCreateStoresEverythingTheAPIReturns(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Idempotency-Key") == "" {
			t.Error("the creation must carry an idempotency key")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(scenarioJSON))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Checkout"),
		"url":          str("https://shop.example.com/cart"),
		"interval_sec": num(300),
		"tags":         strList("prod"),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	var state scenarioModel
	resp.State.Get(ctx, &state)
	if state.ID.ValueString() != "mon-1" {
		t.Errorf("id = %q: without it Terraform loses the resource", state.ID.ValueString())
	}
	// The attributes computed by the API have to be stored, otherwise the next
	// plan announces them as changes to apply.
	if state.LastStatus.ValueString() != "ok" || state.CreatedAt.IsNull() {
		t.Errorf("computed attributes missing: %+v", state)
	}
	if state.Method.ValueString() != "GET" || state.ExpectedStatus.ValueInt64() != 200 {
		t.Errorf("default values of the API not picked up: %+v", state)
	}
	if h.bodies[0]["tags"] == nil {
		t.Error("the tags of the plan must go into the request")
	}
}

func TestScenarioCreateRefusesMissingURL(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		t.Error("no call must go out: the error is detectable beforehand")
		w.WriteHeader(http.StatusBadRequest)
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Without an address"),
		"interval_sec": num(300),
	})}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("an HTTP scenario without a url must be refused")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "url") {
		t.Errorf("the message must name the attribute: %v", resp.Diagnostics)
	}
}

func TestScenarioCreateReportsAPIRefusal(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"This key does not carry the scenarios:write scope."}`))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Checkout"),
		"url":          str("https://example.com"),
		"interval_sec": num(60),
	})}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a refusal from the API must surface")
	}
	// The message of the API names the missing scope: losing it would force one
	// to go looking through the logs.
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "scenarios:write") {
		t.Errorf("diagnostics = %v", resp.Diagnostics)
	}
}

func TestScenarioReadRefreshesState(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Replace(scenarioJSON, `"ok"`, `"fail"`, 1)))
	})

	resp := &resource.ReadResponse{State: h.emptyState()}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{
		"id":           str("mon-1"),
		"name":         str("Checkout"),
		"interval_sec": num(300),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state scenarioModel
	resp.State.Get(ctx, &state)
	if state.LastStatus.ValueString() != "fail" {
		t.Errorf("last_status = %q, the read must reflect the API", state.LastStatus.ValueString())
	}
	if h.requests[0] != "GET /v1/scenarios/mon-1" {
		t.Errorf("call = %q", h.requests[0])
	}
}

func TestScenarioReadDropsDeletedResource(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Scenario not found"}`))
	})

	resp := &resource.ReadResponse{State: h.state(map[string]tftypes.Value{
		"id":           str("mon-deleted"),
		"name":         str("Checkout"),
		"interval_sec": num(300),
	})}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("a resource deleted outside of Terraform must not make the plan fail: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("the resource must leave the state to be recreated on the next apply")
	}
}

func TestScenarioReadReportsRealErrors(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"missing scope"}`))
	})

	resp := &resource.ReadResponse{State: h.emptyState()}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{
		"id":           str("mon-1"),
		"name":         str("Checkout"),
		"interval_sec": num(300),
	})}, resp)

	// Mistaking a 403 for a disappearance would erase the resource from the
	// state, and the next apply would create a second one.
	if !resp.Diagnostics.HasError() {
		t.Fatal("a 403 must not be taken for a deletion")
	}
}

func TestScenarioUpdateDoesNotSendType(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Replace(scenarioJSON, "Checkout", "Checkout v2", 1)))
	})

	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan: h.plan(map[string]tftypes.Value{
			"name":         str("Checkout v2"),
			"type":         str("http"),
			"url":          str("https://shop.example.com/cart"),
			"interval_sec": num(300),
		}),
		State: h.state(map[string]tftypes.Value{
			"id":           str("mon-1"),
			"name":         str("Checkout"),
			"interval_sec": num(300),
		}),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}

	if h.requests[0] != "PATCH /v1/scenarios/mon-1" {
		t.Errorf("call = %q", h.requests[0])
	}
	// The API refuses `type` on a modification: sending it would make the whole
	// body fail.
	if _, present := h.bodies[0]["type"]; present {
		t.Error("the type must not go into a modification")
	}
	var state scenarioModel
	resp.State.Get(ctx, &state)
	if state.Name.ValueString() != "Checkout v2" {
		t.Errorf("name = %q", state.Name.ValueString())
	}
}

func TestScenarioUpdateOnVanishedResource(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Scenario not found"}`))
	})

	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan: h.plan(map[string]tftypes.Value{
			"name":         str("Checkout"),
			"url":          str("https://example.com"),
			"interval_sec": num(300),
		}),
		State: h.state(map[string]tftypes.Value{
			"id":           str("mon-vanished"),
			"name":         str("Checkout"),
			"interval_sec": num(300),
		}),
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("modifying a vanished resource must be reported")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "apply") {
		t.Errorf("the message must say what to do: %v", resp.Diagnostics)
	}
}

func TestScenarioDeleteToleratesAlreadyGone(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		r := NewScenarioResource()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			if status == http.StatusNotFound {
				_, _ = w.Write([]byte(`{"error":"Scenario not found"}`))
			}
		})

		resp := &resource.DeleteResponse{State: h.emptyState()}
		r.Delete(ctx, resource.DeleteRequest{State: h.state(map[string]tftypes.Value{
			"id":           str("mon-1"),
			"name":         str("Checkout"),
			"interval_sec": num(300),
		})}, resp)

		// An object that is already gone is the wanted result: failing would
		// block the destruction of everything else in the plan.
		if resp.Diagnostics.HasError() {
			t.Errorf("HTTP %d: %v", status, resp.Diagnostics)
		}
	}
}

func TestScenarioDeleteReportsRefusal(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"missing scope"}`))
	})

	resp := &resource.DeleteResponse{State: h.emptyState()}
	r.Delete(ctx, resource.DeleteRequest{State: h.state(map[string]tftypes.Value{
		"id":           str("mon-1"),
		"name":         str("Checkout"),
		"interval_sec": num(300),
	})}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused deletion must surface, otherwise Terraform forgets a resource that still exists")
	}
}

// ---------------------------------------------------------------------------
// Maintenance windows
// ---------------------------------------------------------------------------

func TestMaintenanceWindowCreateWeekly(t *testing.T) {
	ctx := context.Background()
	r := NewMaintenanceWindowResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"mw-1","weekday":7,"startMinute":180,"durationMin":120,
		  "startsAt":"2026-09-22T00:00:00.000Z","endsAt":"2036-09-19T00:00:00.000Z","reason":"Backup"}`))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"weekday":      num(7),
		"start_minute": num(180),
		"duration_min": num(120),
		"reason":       str("Backup"),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	var state maintenanceWindowModel
	resp.State.Get(ctx, &state)
	if state.ID.ValueString() != "mw-1" || state.Weekday.ValueInt64() != 7 {
		t.Errorf("state = %+v", state)
	}
	if state.ScenarioID.ValueString() != "" {
		t.Error("without scenario_id, the window covers the organization: the state must stay null")
	}
}

func TestMaintenanceWindowRefusesIncompleteForms(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		plan map[string]tftypes.Value
	}{
		{"weekly without a time", map[string]tftypes.Value{"weekday": num(3)}},
		{"one-off without an end", map[string]tftypes.Value{"starts_at": str("2026-09-22T10:00:00Z")}},
		{"neither one nor the other", map[string]tftypes.Value{"reason": str("nothing")}},
	}
	for _, c := range cases {
		r := NewMaintenanceWindowResource()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			t.Errorf("%s: no call must go out", c.name)
		})
		resp := &resource.CreateResponse{State: h.emptyState()}
		r.Create(ctx, resource.CreateRequest{Plan: h.plan(c.plan)}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("%s: must be refused", c.name)
		}
	}
}

func TestMaintenanceWindowCreateReportsRefusal(t *testing.T) {
	ctx := context.Background()
	r := NewMaintenanceWindowResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"A one-off window cannot exceed 90 days."}`))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"starts_at": str("2026-01-01T00:00:00Z"),
		"ends_at":   str("2030-01-01T00:00:00Z"),
	})}, resp)

	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "90 days") {
		t.Errorf("diagnostics = %v", resp.Diagnostics)
	}
}

func TestMaintenanceWindowReadAndDelete(t *testing.T) {
	ctx := context.Background()
	r := NewMaintenanceWindowResource()
	h := newHarness(t, r, func(hh *harness, w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(`{"id":"mw-1","weekday":7,"startMinute":180,"durationMin":120}`))
	})

	readResp := &resource.ReadResponse{State: h.emptyState()}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{"id": str("mw-1")})}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
	var state maintenanceWindowModel
	readResp.State.Get(ctx, &state)
	if state.DurationMin.ValueInt64() != 120 {
		t.Errorf("duration read back = %d", state.DurationMin.ValueInt64())
	}

	deleteResp := &resource.DeleteResponse{State: h.emptyState()}
	r.Delete(ctx, resource.DeleteRequest{State: h.state(map[string]tftypes.Value{"id": str("mw-1")})}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", deleteResp.Diagnostics)
	}
}

func TestMaintenanceWindowReadDropsDeleted(t *testing.T) {
	ctx := context.Background()
	r := NewMaintenanceWindowResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Maintenance window not found"}`))
	})

	resp := &resource.ReadResponse{State: h.state(map[string]tftypes.Value{"id": str("mw-1")})}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, resp)
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Errorf("the deleted window must leave the state: %v", resp.Diagnostics)
	}
}

func TestMaintenanceWindowUpdateIsRefusedExplicitly(t *testing.T) {
	ctx := context.Background()
	r := NewMaintenanceWindowResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		t.Error("no request: the API cannot modify a window")
	})

	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{Plan: h.plan(nil), State: h.state(nil)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("the modification must be refused with a message, not silently ignored")
	}
}

// ---------------------------------------------------------------------------
// Webhooks
// ---------------------------------------------------------------------------

func TestWebhookCreateKeepsSecretAndURL(t *testing.T) {
	ctx := context.Background()
	r := NewWebhookResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"h1","events":["run.failed","run.recovered"],"enabled":true,
		  "urlFingerprint":"fp1","secret":"whsec_abc","createdAt":"2026-09-22T10:00:00.000Z"}`))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"url": str("https://hooks.example.com/pathly"),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	var state webhookModel
	resp.State.Get(ctx, &state)
	// The secret is only returned once: not storing it would lose it for good
	// for the user.
	if state.Secret.ValueString() != "whsec_abc" {
		t.Errorf("secret = %q", state.Secret.ValueString())
	}
	// The URL comes from the plan, the API never reads it back.
	if state.URL.ValueString() != "https://hooks.example.com/pathly" {
		t.Errorf("url = %q", state.URL.ValueString())
	}
	if state.URLFingerprint.ValueString() != "fp1" {
		t.Errorf("fingerprint = %q", state.URLFingerprint.ValueString())
	}
}

func TestWebhookCreateReportsPlanRefusal(t *testing.T) {
	ctx := context.Background()
	r := NewWebhookResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"Outbound webhooks are included in Business."}`))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"url": str("https://hooks.example.com/pathly"),
	})}, resp)

	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "Business") {
		t.Errorf("diagnostics = %v", resp.Diagnostics)
	}
}

func TestWebhookReadPreservesSecretAndURL(t *testing.T) {
	ctx := context.Background()
	r := NewWebhookResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		// The read returns neither the URL nor the secret.
		_, _ = w.Write([]byte(`{"id":"h1","events":["run.failed"],"enabled":false,"hasSecret":true,"urlFingerprint":"fp2"}`))
	})

	resp := &resource.ReadResponse{State: h.emptyState()}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{
		"id":     str("h1"),
		"url":    str("https://hooks.example.com/pathly"),
		"secret": str("whsec_abc"),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state webhookModel
	resp.State.Get(ctx, &state)
	if state.Secret.ValueString() != "whsec_abc" || state.URL.ValueString() == "" {
		t.Error("the read must not erase what the API does not return")
	}
	// The fingerprint changes: that is the signal that a destination was
	// modified outside of Terraform.
	if state.URLFingerprint.ValueString() != "fp2" {
		t.Errorf("fingerprint = %q", state.URLFingerprint.ValueString())
	}
	if state.Enabled.ValueBool() {
		t.Error("a webhook disabled in the console must be visible in the state")
	}
}

func TestWebhookReadDropsDeletedAndDeleteTolerates(t *testing.T) {
	ctx := context.Background()
	r := NewWebhookResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Webhook not found"}`))
	})

	readResp := &resource.ReadResponse{State: h.state(map[string]tftypes.Value{"id": str("h1"), "url": str("https://x.example.com")})}
	r.Read(ctx, resource.ReadRequest{State: readResp.State}, readResp)
	if readResp.Diagnostics.HasError() || !readResp.State.Raw.IsNull() {
		t.Errorf("read: %v", readResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{State: h.emptyState()}
	r.Delete(ctx, resource.DeleteRequest{State: h.state(map[string]tftypes.Value{
		"id": str("h1"), "url": str("https://x.example.com"),
	})}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Errorf("deletion of a webhook that is already gone: %v", deleteResp.Diagnostics)
	}
}

func TestWebhookUpdateIsRefusedExplicitly(t *testing.T) {
	ctx := context.Background()
	r := NewWebhookResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		t.Error("no request: the API does not rewrite a webhook")
	})

	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{Plan: h.plan(nil), State: h.state(nil)}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("the modification must be refused, the secret changes on recreation")
	}
}

func TestWebhookImportWarnsAboutUnreadableFields(t *testing.T) {
	ctx := context.Background()
	r := NewWebhookResource().(resource.ResourceWithImportState)
	schemaResp := &resource.SchemaResponse{}
	NewWebhookResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "h1"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import: %v", resp.Diagnostics)
	}
	// Without this warning, the user believes they imported a complete webhook
	// and discovers the missing secret on the first signed delivery.
	if resp.Diagnostics.WarningsCount() == 0 {
		t.Error("the import must warn that the URL and the secret cannot be read back")
	}
}

// ---------------------------------------------------------------------------
// SLA targets
// ---------------------------------------------------------------------------

const slaJSON = `{"id":"sla-1","monitorId":"mon-1","name":"Checkout 99.9","objectivePct":99.9,
  "windowDays":30,"excludeMaintenance":true,"warnAtBudgetRatio":0.8,"enabled":true}`

func TestSlaTargetCreateAndUpdateUseUpsert(t *testing.T) {
	ctx := context.Background()
	r := NewSlaTargetResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(slaJSON))
	})

	plan := h.plan(map[string]tftypes.Value{
		"scenario_id":         str("mon-1"),
		"name":                str("Checkout 99.9"),
		"objective_pct":       flt(99.9),
		"window_days":         num(30),
		"exclude_maintenance": boolean(true),
	})

	createResp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics)
	}
	var state slaTargetModel
	createResp.State.Get(ctx, &state)
	if state.ObjectivePct.ValueFloat64() != 99.9 || state.WindowDays.ValueInt64() != 30 {
		t.Errorf("state = %+v", state)
	}

	updateResp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: createResp.State}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", updateResp.Diagnostics)
	}
	// Both go through PUT: the target is identified by its scenario.
	if h.requests[0] != "PUT /v1/sla-targets" || h.requests[1] != "PUT /v1/sla-targets" {
		t.Errorf("calls = %v", h.requests)
	}
}

func TestSlaTargetCreateReportsRefusal(t *testing.T) {
	ctx := context.Background()
	r := NewSlaTargetResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"Scenario not found"}`))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"scenario_id":   str("mon-missing"),
		"objective_pct": flt(99),
		"window_days":   num(7),
	})}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a target on an unknown scenario must fail")
	}
}

func TestSlaTargetUpdateReportsRefusal(t *testing.T) {
	ctx := context.Background()
	r := NewSlaTargetResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"missing sla:write scope"}`))
	})

	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  h.plan(map[string]tftypes.Value{"objective_pct": flt(99), "window_days": num(7)}),
		State: h.state(map[string]tftypes.Value{"id": str("sla-1")}),
	}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a refusal must surface")
	}
}

func TestSlaTargetReadAndDelete(t *testing.T) {
	ctx := context.Background()
	r := NewSlaTargetResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(slaJSON))
	})

	readResp := &resource.ReadResponse{State: h.emptyState()}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{"id": str("sla-1")})}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{State: h.emptyState()}
	r.Delete(ctx, resource.DeleteRequest{State: h.state(map[string]tftypes.Value{"id": str("sla-1")})}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", deleteResp.Diagnostics)
	}
}

func TestSlaTargetReadDropsDeleted(t *testing.T) {
	ctx := context.Background()
	r := NewSlaTargetResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"SLA target not found"}`))
	})

	resp := &resource.ReadResponse{State: h.state(map[string]tftypes.Value{"id": str("sla-1")})}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, resp)
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Errorf("deleted target: %v", resp.Diagnostics)
	}
}

func TestSlaTargetDeleteReportsRefusal(t *testing.T) {
	ctx := context.Background()
	r := NewSlaTargetResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"missing scope"}`))
	})

	resp := &resource.DeleteResponse{State: h.emptyState()}
	r.Delete(ctx, resource.DeleteRequest{State: h.state(map[string]tftypes.Value{"id": str("sla-1")})}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused deletion must surface")
	}
}

// ---------------------------------------------------------------------------
// Data source
// ---------------------------------------------------------------------------

func TestScenariosDataSourceFiltersAndPaginates(t *testing.T) {
	ctx := context.Background()
	d := NewScenariosDataSource()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		page++
		if page == 1 {
			_, _ = w.Write([]byte(`{"items":[
			  {"id":"a","name":"A","type":"http","tags":["prod"],"folder":"Shop","lastStatus":"ok"},
			  {"id":"b","name":"B","type":"http","tags":["staging"],"folder":"Shop"}
			],"nextCursor":"c1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[
		  {"id":"c","name":"C","type":"http","tags":["prod"],"folder":"Internal"}
		],"nextCursor":null}`))
	}))
	defer srv.Close()

	configureResp := &datasource.ConfigureResponse{}
	d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{
		ProviderData: client.New(srv.URL, "sp_test"),
	}, configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("Configure: %v", configureResp.Diagnostics)
	}

	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: dataObject(objType, map[string]tftypes.Value{
		"filter_tag": str("prod"),
	})}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	d.Read(ctx, datasource.ReadRequest{Config: config}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state scenariosDataSourceModel
	resp.State.Get(ctx, &state)
	// Both pages are walked, and only the requested tag remains.
	if len(state.Scenarios) != 2 {
		t.Fatalf("scenarios = %+v", state.Scenarios)
	}
	if state.Scenarios[0].ID.ValueString() != "a" || state.Scenarios[1].ID.ValueString() != "c" {
		t.Errorf("unexpected filtering: %+v", state.Scenarios)
	}
}

func TestScenariosDataSourceFiltersByFolder(t *testing.T) {
	ctx := context.Background()
	d := NewScenariosDataSource()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[
		  {"id":"a","name":"A","type":"http","folder":"Shop"},
		  {"id":"b","name":"B","type":"http"}
		],"nextCursor":null}`))
	}))
	defer srv.Close()

	d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{
		ProviderData: client.New(srv.URL, "sp_test"),
	}, &datasource.ConfigureResponse{})

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    dataObject(objType, map[string]tftypes.Value{"filter_folder": str("Shop")}),
	}}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state scenariosDataSourceModel
	resp.State.Get(ctx, &state)
	if len(state.Scenarios) != 1 || state.Scenarios[0].ID.ValueString() != "a" {
		t.Errorf("filtering by folder: %+v", state.Scenarios)
	}
}

func TestScenariosDataSourceReportsError(t *testing.T) {
	ctx := context.Background()
	d := NewScenariosDataSource()
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"This key does not carry the scenarios:read scope."}`))
	}))
	defer srv.Close()

	d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{
		ProviderData: client.New(srv.URL, "sp_test"),
	}, &datasource.ConfigureResponse{})

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    dataObject(objType, nil),
	}}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused read must surface instead of returning an empty list")
	}
}

// dataObject is worth object() for a data source schema: same need, different
// types on the framework side.
func dataObject(objType tftypes.Object, set map[string]tftypes.Value) tftypes.Value {
	return object(objType, set)
}

// The schema of the data source must describe its filter attributes, otherwise
// its registry page does not say how to use it.
func TestScenariosDataSourceDocumentsFilters(t *testing.T) {
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	NewScenariosDataSource().Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	for _, name := range []string{"filter_tag", "filter_folder", "scenarios"} {
		attr, ok := schemaResp.Schema.Attributes[name].(fwdsschema.Attribute)
		if !ok {
			t.Fatalf("attribute %s missing", name)
		}
		if attr.GetDescription() == "" {
			t.Errorf("%s without a description", name)
		}
	}
	// Typing guard rail: the resource schema and the data source schema are two
	// distinct packages, confusing them does not compile.
	var _ fwschema.Schema
}
