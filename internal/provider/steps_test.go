package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const browserJSON = `{
  "id": "mon-b", "name": "Checkout", "type": "browser", "url": "https://shop.example.com/login",
  "enabled": true, "intervalSec": 300, "lastStatus": "ok",
  "scenarioFingerprint": "sha256:abc", "createdAt": "2026-09-23T10:00:00.000Z"
}`

func stepsType(t *testing.T) tftypes.Object {
	t.Helper()
	schemaResp := &resource.SchemaResponse{}
	NewScenarioResource().Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	attr, ok := schemaResp.Schema.Attributes["steps"].(fwschema.ListNestedAttribute)
	if !ok {
		t.Fatal("steps missing from the schema")
	}
	return attr.NestedObject.Type().TerraformType(context.Background()).(tftypes.Object)
}

func stepsList(t *testing.T, steps ...map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	typ := stepsType(t)
	elems := make([]tftypes.Value, 0, len(steps))
	for _, step := range steps {
		elems = append(elems, object(typ, step))
	}
	return tftypes.NewValue(tftypes.List{ElementType: typ}, elems)
}

func TestBrowserScenarioSendsTheJourneyAndKeepsIt(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte(browserJSON))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Checkout"),
		"type":         str("browser"),
		"interval_sec": num(300),
		"steps": stepsList(t,
			map[string]tftypes.Value{"op": str("goto"), "url": str("https://shop.example.com/login")},
			map[string]tftypes.Value{"op": str("fill"), "selector": str("#email"), "value": str("ops@example.com")},
			map[string]tftypes.Value{"op": str("fill"), "selector": str("#password"), "value": str("secret")},
			map[string]tftypes.Value{"op": str("click"), "selector": str("button[type=submit]")},
			map[string]tftypes.Value{"op": str("assert_text"), "text": str("Your cart")},
		),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	tree, ok := h.bodies[0]["scenario"].(map[string]any)
	if !ok {
		t.Fatalf("the journey must travel under scenario: %v", h.bodies[0])
	}
	steps, ok := tree["steps"].([]any)
	if !ok || len(steps) != 5 {
		t.Fatalf("steps = %v", tree["steps"])
	}
	first, _ := steps[0].(map[string]any)
	if first["op"] != "goto" || first["url"] != "https://shop.example.com/login" {
		t.Errorf("first step = %v", first)
	}
	// A password belongs in the request: without it the run cannot log in.
	third, _ := steps[2].(map[string]any)
	if third["value"] != "secret" {
		t.Errorf("password fill = %v", third)
	}

	var state scenarioModel
	resp.State.Get(ctx, &state)
	if state.ScenarioFingerprint.ValueString() != "sha256:abc" {
		t.Errorf("fingerprint = %q", state.ScenarioFingerprint.ValueString())
	}
	// The API does not echo the tree: losing it here would make the next plan
	// announce the deletion of every step.
	if state.Steps.IsNull() || len(state.Steps.Elements()) != 5 {
		t.Errorf("steps in state = %v", state.Steps)
	}
}

func TestBrowserScenarioRefusesAJourneyWithoutGoto(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		t.Error("no call: the first step is not a goto")
		w.WriteHeader(400)
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Broken"),
		"type":         str("browser"),
		"interval_sec": num(300),
		"steps": stepsList(t,
			map[string]tftypes.Value{"op": str("click"), "selector": str("#go")},
		),
	})}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a journey that does not start with goto must be refused")
	}
}

func TestBrowserScenarioRefusesMissingSteps(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		t.Error("no call: a browser scenario without steps is empty")
		w.WriteHeader(400)
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Empty"),
		"type":         str("browser"),
		"interval_sec": num(300),
	})}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a browser scenario without steps must be refused")
	}
}

func TestStepMissingItsRequiredFieldIsNamed(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		t.Error("no call: fill without a value is incomplete")
		w.WriteHeader(400)
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Login"),
		"type":         str("browser"),
		"interval_sec": num(300),
		"steps": stepsList(t,
			map[string]tftypes.Value{"op": str("goto"), "url": str("https://shop.example.com")},
			map[string]tftypes.Value{"op": str("fill"), "selector": str("#password")},
		),
	})}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("fill without a value must be refused")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "value") {
		t.Errorf("the message must name the field: %v", resp.Diagnostics)
	}
}

func TestScenarioReadKeepsTheJourneyAndRefreshesTheFingerprint(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"mon-b","name":"Checkout","type":"browser",
		  "url":"https://shop.example.com/login","intervalSec":300,
		  "scenarioFingerprint":"sha256:changed"}`))
	})

	kept := stepsList(t, map[string]tftypes.Value{
		"op": str("goto"), "url": str("https://shop.example.com/login"),
	})
	resp := &resource.ReadResponse{State: h.emptyState()}
	r.Read(ctx, resource.ReadRequest{State: h.state(map[string]tftypes.Value{
		"id":                   str("mon-b"),
		"name":                 str("Checkout"),
		"type":                 str("browser"),
		"interval_sec":         num(300),
		"steps":                kept,
		"scenario_fingerprint": str("sha256:abc"),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var state scenarioModel
	resp.State.Get(ctx, &state)
	if state.ScenarioFingerprint.ValueString() != "sha256:changed" {
		t.Errorf("fingerprint = %q, a console edit must show up as drift", state.ScenarioFingerprint.ValueString())
	}
	if state.Steps.IsNull() || len(state.Steps.Elements()) != 1 {
		t.Errorf("steps lost on read: %v", state.Steps)
	}
}

func TestScenarioUpdateSendsAChangedJourney(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(browserJSON))
	})

	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan: h.plan(map[string]tftypes.Value{
			"name":         str("Checkout"),
			"type":         str("browser"),
			"interval_sec": num(300),
			"steps": stepsList(t,
				map[string]tftypes.Value{"op": str("goto"), "url": str("https://shop.example.com/login")},
				map[string]tftypes.Value{"op": str("assert_text"), "text": str("Welcome")},
			),
		}),
		State: h.state(map[string]tftypes.Value{"id": str("mon-b")}),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	tree, _ := h.bodies[0]["scenario"].(map[string]any)
	steps, _ := tree["steps"].([]any)
	if len(steps) != 2 {
		t.Errorf("updated steps = %v", tree)
	}
}

func TestJourneyOptionsTravelWithTheSteps(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte(browserJSON))
	})

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	authAttr := schemaResp.Schema.Attributes["basic_auth"].(fwschema.SingleNestedAttribute)
	authType := authAttr.GetType().TerraformType(ctx).(tftypes.Object)

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":              str("Checkout"),
		"type":              str("browser"),
		"interval_sec":      num(300),
		"click_delay_ms":    num(500),
		"viewport":          str("mobile"),
		"locale":            str("fr-FR"),
		"scenario_timezone": str("Europe/Paris"),
		"headers": tftypes.NewValue(
			tftypes.Map{ElementType: tftypes.String},
			map[string]tftypes.Value{"X-Test": str("1")},
		),
		"basic_auth": object(authType, map[string]tftypes.Value{
			"username": str("probe"),
			"password": str("s3cret"),
		}),
		"steps": stepsList(t,
			map[string]tftypes.Value{"op": str("goto"), "url": str("https://shop.example.com")},
			map[string]tftypes.Value{"op": str("wait"), "ms": num(200)},
			map[string]tftypes.Value{"op": str("press"), "key": str("Enter")},
			map[string]tftypes.Value{
				"op":           str("wait_for_response"),
				"url_includes": str("/api/cart"),
				"status":       num(200),
			},
			map[string]tftypes.Value{
				"op":       str("http_auth"),
				"username": str("probe"),
				"password": str("s3cret"),
			},
			map[string]tftypes.Value{
				"op":          str("assert_json_path"),
				"path":        str("$.ok"),
				"json_equals": str("true"),
			},
			map[string]tftypes.Value{
				"op":       str("assert_header"),
				"name":     str("content-type"),
				"includes": str("json"),
			},
			map[string]tftypes.Value{
				"op":        str("assert_url"),
				"includes":  str("/checkout"),
				"url_regex": str("^/checkout"),
			},
		),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	tree, _ := h.bodies[0]["scenario"].(map[string]any)
	if tree["clickDelayMs"] != float64(500) || tree["viewport"] != "mobile" {
		t.Errorf("options = %v", tree)
	}
	if tree["locale"] != "fr-FR" || tree["timezone"] != "Europe/Paris" {
		t.Errorf("locale/timezone = %v", tree)
	}
	headers, _ := tree["headers"].(map[string]any)
	if headers["X-Test"] != "1" {
		t.Errorf("headers = %v", headers)
	}
	auth, _ := tree["basicAuth"].(map[string]any)
	if auth["username"] != "probe" || auth["password"] != "s3cret" {
		t.Errorf("basicAuth = %v", auth)
	}
	if steps, _ := tree["steps"].([]any); len(steps) != 8 {
		t.Errorf("steps = %v", tree["steps"])
	}
}

func TestIncompleteStepsNameTheMissingField(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		step  map[string]tftypes.Value
		field string
	}{
		{name: "goto", step: map[string]tftypes.Value{"op": str("goto")}, field: "url"},
		{name: "click", step: map[string]tftypes.Value{"op": str("click")}, field: "selector"},
		{name: "wait", step: map[string]tftypes.Value{"op": str("wait")}, field: "ms"},
		{name: "assert_text", step: map[string]tftypes.Value{"op": str("assert_text")}, field: "text"},
		{name: "press", step: map[string]tftypes.Value{"op": str("press")}, field: "key"},
		{name: "wait_for_response", step: map[string]tftypes.Value{"op": str("wait_for_response")}, field: "url_includes"},
		{name: "http_auth", step: map[string]tftypes.Value{"op": str("http_auth")}, field: "username"},
		{name: "assert_json_path", step: map[string]tftypes.Value{"op": str("assert_json_path")}, field: "path"},
		{name: "assert_header name", step: map[string]tftypes.Value{"op": str("assert_header")}, field: "name"},
		{name: "assert_header includes", step: map[string]tftypes.Value{"op": str("assert_header"), "name": str("content-type")}, field: "includes"},
		{name: "upload", step: map[string]tftypes.Value{"op": str("upload"), "selector": str("#file")}, field: "file"},
		{name: "select", step: map[string]tftypes.Value{"op": str("select"), "selector": str("#country")}, field: "value"},
		{name: "if_text", step: map[string]tftypes.Value{"op": str("if_text")}, field: "text"},
		{name: "http_auth password", step: map[string]tftypes.Value{"op": str("http_auth"), "username": str("probe")}, field: "password"},
		{name: "json_equals", step: map[string]tftypes.Value{"op": str("assert_json_path"), "path": str("$.ok")}, field: "json_equals"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewScenarioResource()
			h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
				t.Error("no call from an incomplete step")
				w.WriteHeader(400)
			})
			resp := &resource.CreateResponse{State: h.emptyState()}
			r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
				"name":         str("X"),
				"type":         str("browser"),
				"interval_sec": num(60),
				"steps": stepsList(t,
					map[string]tftypes.Value{"op": str("goto"), "url": str("https://example.com")},
					tc.step,
				),
			})}, resp)
			if !resp.Diagnostics.HasError() {
				t.Fatalf("%s: must be refused", tc.name)
			}
			if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), tc.field) {
				t.Errorf("%s: %v", tc.name, resp.Diagnostics)
			}
		})
	}
}

func TestAstFromRefusesBrokenConversionsAndDropsEmptyHeaders(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(browserJSON))
	})
	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Checkout"),
		"type":         str("browser"),
		"interval_sec": num(300),
		"steps": stepsList(t, map[string]tftypes.Value{
			"op": str("goto"), "url": str("https://shop.example.com"),
		}),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	var state scenarioModel
	resp.State.Get(ctx, &state)

	var diags diag.Diagnostics
	state.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{})
	ast := astFrom(ctx, state, &diags)
	if diags.HasError() || ast == nil {
		t.Fatalf("readable journey: %v", diags)
	}
	if ast.Headers != nil {
		t.Errorf("an empty headers map must stay out of the request: %v", ast.Headers)
	}

	authTypes := map[string]attr.Type{"username": types.StringType, "password": types.StringType}
	state.BasicAuth = types.ObjectValueMust(authTypes, map[string]attr.Value{
		"username": types.StringValue("probe"),
		"password": types.StringNull(),
	})
	diags = nil
	ast = astFrom(ctx, state, &diags)
	if diags.HasError() || ast == nil || ast.BasicAuth == nil || ast.BasicAuth.Password != "" {
		t.Errorf("a login without a password must still be sent: %+v %v", ast, diags)
	}
	state.BasicAuth = types.ObjectValueMust(authTypes, map[string]attr.Value{
		"username": types.StringNull(),
		"password": types.StringValue("x"),
	})
	diags = nil
	ast = astFrom(ctx, state, &diags)
	if ast == nil || ast.BasicAuth != nil {
		t.Errorf("auth without a login must stay out: %+v", ast)
	}

	diags = nil
	state.Headers = types.MapValueMust(types.Int64Type, map[string]attr.Value{"X": types.Int64Value(1)})
	if astFrom(ctx, state, &diags) != nil || !diags.HasError() {
		t.Error("headers that are not strings must stop the conversion")
	}

	diags = nil
	broken := scenarioModel{Steps: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("nope")})}
	if astFrom(ctx, broken, &diags) != nil || !diags.HasError() {
		t.Error("steps that are not objects must stop the conversion")
	}

	diags = nil
	firstStepMustBeGoto(ctx, scenarioModel{Steps: types.ListNull(types.StringType)}, &diags)
	firstStepMustBeGoto(ctx, scenarioModel{Steps: types.ListValueMust(types.StringType, nil)}, &diags)
	firstStepMustBeGoto(ctx, broken, &diags)
	if !diags.HasError() {
		t.Error("unreadable steps must surface on the first-step check")
	}
}
