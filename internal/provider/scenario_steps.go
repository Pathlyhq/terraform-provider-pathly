package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

// stepOps is the closed list the API accepts. A made-up name is refused here
// rather than after the rest of the plan has already been applied.
var stepOps = []string{
	"goto", "click", "hover", "fill", "select", "upload", "scroll",
	"switch_tab", "wait", "wait_for", "assert_text", "assert_visible",
	"assert_no_cmp", "assert_url", "press", "wait_for_response",
	"wait_for_networkidle", "if_visible", "if_text", "assert_amount",
	"assert_json_path", "assert_header", "http_auth",
}

var viewports = []string{
	"desktop", "tablet", "mobile", "iphone_se", "iphone_14",
	"pixel_7", "galaxy_s21", "ipad_mini",
}

type stepModel struct {
	Op          types.String  `tfsdk:"op"`
	URL         types.String  `tfsdk:"url"`
	Selector    types.String  `tfsdk:"selector"`
	Value       types.String  `tfsdk:"value"`
	Text        types.String  `tfsdk:"text"`
	Ms          types.Int64   `tfsdk:"ms"`
	TimeoutMs   types.Int64   `tfsdk:"timeout_ms"`
	Includes    types.String  `tfsdk:"includes"`
	File        types.String  `tfsdk:"file"`
	NewTab      types.Bool    `tfsdk:"new_tab"`
	Href        types.String  `tfsdk:"href"`
	URLIncludes types.String  `tfsdk:"url_includes"`
	IgnoreCase  types.Bool    `tfsdk:"ignore_case"`
	Regex       types.Bool    `tfsdk:"regex"`
	URLRegex    types.String  `tfsdk:"url_regex"`
	Path        types.String  `tfsdk:"path"`
	IgnoreHash  types.Bool    `tfsdk:"ignore_hash"`
	IgnoreQuery types.Bool    `tfsdk:"ignore_query"`
	Key         types.String  `tfsdk:"key"`
	Status      types.Int64   `tfsdk:"status"`
	RetryTimes  types.Int64   `tfsdk:"retry_times"`
	Currency    types.String  `tfsdk:"currency"`
	Min         types.Float64 `tfsdk:"min"`
	Max         types.Float64 `tfsdk:"max"`
	Equals      types.Float64 `tfsdk:"equals"`
	Name        types.String  `tfsdk:"name"`
	Username    types.String  `tfsdk:"username"`
	Password    types.String  `tfsdk:"password"`
	JSONEquals  types.String  `tfsdk:"json_equals"`
}

type basicAuthModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

// stepAttributes describes one action of a browser journey.
//
// Every field except `op` is optional at the schema level: which ones are
// required depends on the operation, and that check lives in validateSteps so
// the error names both the index and the missing field.
func stepAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"op": schema.StringAttribute{
			Required:    true,
			Description: "Action to run. See the resource documentation for the closed list.",
			Validators:  []validator.String{stringvalidator.OneOf(stepOps...)},
		},
		"url":          schema.StringAttribute{Optional: true, Description: "Address to open. Required on `goto`."},
		"selector":     schema.StringAttribute{Optional: true, Description: "CSS selector of the target element.", Validators: []validator.String{stringvalidator.LengthBetween(1, 500)}},
		"value":        schema.StringAttribute{Optional: true, Sensitive: true, Description: "Value to type or to select. Marked sensitive: a `fill` often carries a password."},
		"text":         schema.StringAttribute{Optional: true, Description: "Visible text, used by `assert_text`, `if_text` and as a fallback on `click`."},
		"ms":           schema.Int64Attribute{Optional: true, Description: "Pause, in milliseconds. Required on `wait`.", Validators: []validator.Int64{int64validator.Between(0, 30_000)}},
		"timeout_ms":   schema.Int64Attribute{Optional: true, Description: "How long to wait for the condition, in milliseconds.", Validators: []validator.Int64{int64validator.Between(100, 60_000)}},
		"includes":     schema.StringAttribute{Optional: true, Description: "Substring expected in the URL (`assert_url`) or in a header."},
		"file":         schema.StringAttribute{Optional: true, Description: "File name to upload. Letters, digits, dots, underscores and hyphens only."},
		"new_tab":      schema.BoolAttribute{Optional: true, Description: "On `click`, wait for a popup and switch to it."},
		"href":         schema.StringAttribute{Optional: true, Description: "Link address, used as a fallback when the CSS selector of a `click` breaks."},
		"url_includes": schema.StringAttribute{Optional: true, Description: "Substring of the URL, used by `switch_tab` and `wait_for_response`."},
		"ignore_case":  schema.BoolAttribute{Optional: true, Description: "On `assert_text`, compare without regard to case."},
		"regex":        schema.BoolAttribute{Optional: true, Description: "On `assert_text`, treat `text` as a regular expression."},
		"url_regex":    schema.StringAttribute{Optional: true, Description: "On `assert_url`, regular expression the address must match."},
		"path":         schema.StringAttribute{Optional: true, Description: "URL path (`assert_url`) or JSON path (`assert_json_path`)."},
		"ignore_hash":  schema.BoolAttribute{Optional: true, Description: "On `assert_url`, ignore the fragment."},
		"ignore_query": schema.BoolAttribute{Optional: true, Description: "On `assert_url`, ignore the query string."},
		"key": schema.StringAttribute{
			Optional:    true,
			Description: "Key to press: `Enter`, `Escape`, `Tab`, `ArrowDown`, `ArrowUp` or `Space`.",
			Validators:  []validator.String{stringvalidator.OneOf("Enter", "Escape", "Tab", "ArrowDown", "ArrowUp", "Space")},
		},
		"status":      schema.Int64Attribute{Optional: true, Description: "HTTP status expected by `wait_for_response`.", Validators: []validator.Int64{int64validator.Between(100, 599)}},
		"retry_times": schema.Int64Attribute{Optional: true, Description: "Retries of a flaky action, 0 to 5.", Validators: []validator.Int64{int64validator.Between(0, 5)}},
		"currency":    schema.StringAttribute{Optional: true, Description: "Currency symbol or code expected by `assert_amount`."},
		"min":         schema.Float64Attribute{Optional: true, Description: "Lower bound of `assert_amount`."},
		"max":         schema.Float64Attribute{Optional: true, Description: "Upper bound of `assert_amount`."},
		"equals":      schema.Float64Attribute{Optional: true, Description: "Exact amount expected by `assert_amount`.", Validators: []validator.Float64{float64validator.AtLeast(0)}},
		"name":        schema.StringAttribute{Optional: true, Description: "Header name, used by `assert_header`."},
		"username":    schema.StringAttribute{Optional: true, Sensitive: true, Description: "Login of an `http_auth` step."},
		"password":    schema.StringAttribute{Optional: true, Sensitive: true, Description: "Password of an `http_auth` step."},
		"json_equals": schema.StringAttribute{Optional: true, Description: "Expected value of `assert_json_path`, as text."},
	}
}

func journeyAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"http_chain": schema.ListNestedAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Chain hops (login → API). Sent as `httpChain` on POST /v1/scenarios. Bodies and headers can carry secrets — keep them in a secret store.",
			Validators:    []validator.List{listvalidator.SizeBetween(1, 10)},
			PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"name":   schema.StringAttribute{Optional: true, Description: "Label shown in the timeline.", Validators: []validator.String{stringvalidator.LengthBetween(1, 80)}},
				"method": schema.StringAttribute{Required: true, Description: "HTTP method.", Validators: []validator.String{stringvalidator.OneOf("GET", "POST", "PUT", "PATCH", "HEAD", "DELETE")}},
				"url":    schema.StringAttribute{Required: true, Description: "Hop address."},
				"headers": schema.MapAttribute{
					Optional:    true,
					Sensitive:   true,
					ElementType: types.StringType,
					Description: "Hop headers. Can carry secrets — keep them in a secret store. Never returned by the API.",
				},
				"body":              schema.StringAttribute{Optional: true, Sensitive: true, Description: "JSON body. Marked sensitive."},
				"wait_ms":           schema.Int64Attribute{Optional: true, Description: "Pause after the response, in milliseconds.", Validators: []validator.Int64{int64validator.Between(0, 30_000)}},
				"assert_status":     schema.Int64Attribute{Optional: true, Description: "Expected HTTP status.", Validators: []validator.Int64{int64validator.Between(100, 599)}},
				"expect_text":       schema.StringAttribute{Optional: true, Description: "Substring expected in the response."},
				"extract_json_path": schema.StringAttribute{Optional: true, Description: "JSON path to extract for the next hop."},
				"extract_json_as":   schema.StringAttribute{Optional: true, Description: "Variable name for the extracted value."},
				"extract_cookie":    schema.StringAttribute{Optional: true, Description: "Cookie name to keep for the next hop."},
			}},
		},
		"steps": schema.ListNestedAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Browser journey, in order. The first step must be `goto`. Never returned by the API: only `scenario_fingerprint` changes when the tree is edited outside of Terraform.",
			Validators:    []validator.List{listvalidator.SizeBetween(1, 50)},
			PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			NestedObject:  schema.NestedAttributeObject{Attributes: stepAttributes()},
		},
		"headers": schema.MapAttribute{
			ElementType:   types.StringType,
			Optional:      true,
			Computed:      true,
			Description:   "Extra HTTP headers sent with the first request, 10 at most. `Host`, `Content-Length`, `Connection` and `Transfer-Encoding` are refused.",
			Validators:    []validator.Map{mapvalidator.SizeAtMost(10)},
			PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
		},
		"click_delay_ms": schema.Int64Attribute{
			Optional:      true,
			Computed:      true,
			Description:   "Pause between two consecutive clicks, in milliseconds. 2000 when omitted, 0 for none.",
			Validators:    []validator.Int64{int64validator.Between(0, 30_000)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"viewport": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Browser window: `desktop`, `tablet`, `mobile`, `iphone_se`, `iphone_14`, `pixel_7`, `galaxy_s21` or `ipad_mini`.",
			Validators:    []validator.String{stringvalidator.OneOf(viewports...)},
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"locale": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "Browser locale, for example `fr-FR`.",
			Validators:    []validator.String{stringvalidator.LengthAtMost(16)},
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"scenario_timezone": schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "IANA timezone of the browser, for example `Europe/Paris`. Distinct from the organization timezone.",
			Validators:    []validator.String{stringvalidator.LengthAtMost(64)},
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"basic_auth": schema.SingleNestedAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "HTTP authentication of the first request. The password is never returned by the API.",
			PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
			Attributes: map[string]schema.Attribute{
				"username": schema.StringAttribute{Required: true, Sensitive: true, Description: "Login."},
				"password": schema.StringAttribute{Required: true, Sensitive: true, Description: "Password."},
			},
		},
		"scenario_fingerprint": schema.StringAttribute{
			Computed:      true,
			Description:   "Hash of the stored journey. Changes when the tree is edited, including from the console.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
}

func astFrom(ctx context.Context, m scenarioModel, diags *diag.Diagnostics) *client.ScenarioAst {
	if m.Steps.IsNull() || m.Steps.IsUnknown() {
		return nil
	}
	var steps []stepModel
	diags.Append(m.Steps.ElementsAs(ctx, &steps, false)...)
	if diags.HasError() {
		return nil
	}

	ast := &client.ScenarioAst{Steps: make([]client.ScenarioStep, 0, len(steps))}
	for i, step := range steps {
		converted, ok := stepToClient(step, i, diags)
		if !ok {
			return nil
		}
		ast.Steps = append(ast.Steps, converted)
	}

	if !m.Headers.IsNull() && !m.Headers.IsUnknown() {
		headers := map[string]string{}
		diags.Append(m.Headers.ElementsAs(ctx, &headers, false)...)
		if len(headers) > 0 {
			ast.Headers = headers
		}
	}
	ast.ClickDelayMs = int64Ptr(m.ClickDelayMs)
	ast.Viewport = strPtr(m.Viewport)
	ast.Locale = strPtr(m.Locale)
	ast.Timezone = strPtr(m.ScenarioTimezone)

	if !m.BasicAuth.IsNull() && !m.BasicAuth.IsUnknown() {
		var auth basicAuthModel
		diags.Append(m.BasicAuth.As(ctx, &auth, basetypes.ObjectAsOptions{})...)
		if user := strPtr(auth.Username); user != nil {
			password := ""
			if p := strPtr(auth.Password); p != nil {
				password = *p
			}
			ast.BasicAuth = &client.BasicAuth{Username: *user, Password: password}
		}
	}
	if diags.HasError() {
		return nil
	}
	return ast
}

func stepToClient(step stepModel, index int, diags *diag.Diagnostics) (client.ScenarioStep, bool) {
	op := step.Op.ValueString()
	missing := func(field string) {
		diags.AddAttributeError(
			path.Root("steps").AtListIndex(index).AtName(field),
			"Incomplete step",
			fmt.Sprintf("Step %d (`%s`) needs `%s`.", index, op, field),
		)
	}
	switch op {
	case "goto":
		if step.URL.IsNull() || step.URL.ValueString() == "" {
			missing("url")
		}
	case "click", "hover", "fill", "select", "upload", "scroll", "wait_for", "assert_visible", "if_visible":
		if step.Selector.IsNull() || step.Selector.ValueString() == "" {
			missing("selector")
		}
		if op == "fill" || op == "select" {
			if step.Value.IsNull() {
				missing("value")
			}
		}
		if op == "upload" && (step.File.IsNull() || step.File.ValueString() == "") {
			missing("file")
		}
	case "wait":
		if step.Ms.IsNull() {
			missing("ms")
		}
	case "assert_text", "if_text":
		if step.Text.IsNull() || step.Text.ValueString() == "" {
			missing("text")
		}
	case "press":
		if step.Key.IsNull() || step.Key.ValueString() == "" {
			missing("key")
		}
	case "wait_for_response":
		if step.URLIncludes.IsNull() || step.URLIncludes.ValueString() == "" {
			missing("url_includes")
		}
	case "http_auth":
		if step.Username.IsNull() || step.Username.ValueString() == "" {
			missing("username")
		}
		if step.Password.IsNull() {
			missing("password")
		}
	case "assert_json_path":
		if step.Path.IsNull() || step.Path.ValueString() == "" {
			missing("path")
		}
		if step.JSONEquals.IsNull() || step.JSONEquals.ValueString() == "" {
			missing("json_equals")
		}
	case "assert_header":
		if step.Name.IsNull() || step.Name.ValueString() == "" {
			missing("name")
		}
		if step.Includes.IsNull() || step.Includes.ValueString() == "" {
			missing("includes")
		}
	}
	if diags.HasError() {
		return client.ScenarioStep{}, false
	}
	return client.ScenarioStep{
		Op:          op,
		URL:         strPtr(step.URL),
		Selector:    strPtr(step.Selector),
		Value:       strPtr(step.Value),
		Text:        strPtr(step.Text),
		Ms:          int64Ptr(step.Ms),
		TimeoutMs:   int64Ptr(step.TimeoutMs),
		Includes:    strPtr(step.Includes),
		File:        strPtr(step.File),
		NewTab:      boolPtr(step.NewTab),
		Href:        strPtr(step.Href),
		URLIncludes: strPtr(step.URLIncludes),
		IgnoreCase:  boolPtr(step.IgnoreCase),
		Regex:       boolPtr(step.Regex),
		URLRegex:    strPtr(step.URLRegex),
		Path:        strPtr(step.Path),
		IgnoreHash:  boolPtr(step.IgnoreHash),
		IgnoreQuery: boolPtr(step.IgnoreQuery),
		Key:         strPtr(step.Key),
		Status:      int64Ptr(step.Status),
		RetryTimes:  int64Ptr(step.RetryTimes),
		Currency:    strPtr(step.Currency),
		Min:         float64Ptr(step.Min),
		Max:         float64Ptr(step.Max),
		Equals:      float64Ptr(step.Equals),
		Name:        strPtr(step.Name),
		Username:    strPtr(step.Username),
		Password:    strPtr(step.Password),
		JSONEquals:  strPtr(step.JSONEquals),
	}, true
}

type hopModel struct {
	Name            types.String `tfsdk:"name"`
	Method          types.String `tfsdk:"method"`
	URL             types.String `tfsdk:"url"`
	Headers         types.Map    `tfsdk:"headers"`
	Body            types.String `tfsdk:"body"`
	WaitMs          types.Int64  `tfsdk:"wait_ms"`
	AssertStatus    types.Int64  `tfsdk:"assert_status"`
	ExpectText      types.String `tfsdk:"expect_text"`
	ExtractJSONPath types.String `tfsdk:"extract_json_path"`
	ExtractJSONAs   types.String `tfsdk:"extract_json_as"`
	ExtractCookie   types.String `tfsdk:"extract_cookie"`
}

func hopsFrom(ctx context.Context, m scenarioModel, diags *diag.Diagnostics) []client.HttpChainHop {
	if m.HttpChain.IsNull() || m.HttpChain.IsUnknown() {
		return nil
	}
	var hops []hopModel
	diags.Append(m.HttpChain.ElementsAs(ctx, &hops, false)...)
	if diags.HasError() || len(hops) == 0 {
		return nil
	}
	out := make([]client.HttpChainHop, 0, len(hops))
	for _, hop := range hops {
		item := client.HttpChainHop{
			Name:          strPtr(hop.Name),
			Method:        hop.Method.ValueString(),
			URL:           hop.URL.ValueString(),
			Body:          strPtr(hop.Body),
			WaitMs:        int64Ptr(hop.WaitMs),
			AssertStatus:  int64Ptr(hop.AssertStatus),
			ExpectText:    strPtr(hop.ExpectText),
			ExtractCookie: strPtr(hop.ExtractCookie),
		}
		if !hop.Headers.IsNull() && !hop.Headers.IsUnknown() {
			var headers map[string]string
			diags.Append(hop.Headers.ElementsAs(ctx, &headers, false)...)
			if len(headers) > 0 {
				item.Headers = headers
			}
		}
		if path := hop.ExtractJSONPath.ValueString(); path != "" && hop.ExtractJSONAs.ValueString() != "" {
			item.ExtractJSON = &client.ExtractJSON{Path: path, As: hop.ExtractJSONAs.ValueString()}
		}
		out = append(out, item)
	}
	return out
}

func requireJourneyOrURL(m scenarioModel, diags *diag.Diagnostics) {
	browser := m.Type.ValueString() == "browser"
	hasSteps := !m.Steps.IsNull() && !m.Steps.IsUnknown()
	hasURL := !m.URL.IsNull() && !m.URL.IsUnknown() && m.URL.ValueString() != ""
	hasChain := !m.HttpChain.IsNull() && !m.HttpChain.IsUnknown()

	if browser && !hasSteps {
		diags.AddAttributeError(
			path.Root("steps"),
			"Journey required",
			"A Flow is a sequence of actions: set `steps`, starting with `goto`.",
		)
		return
	}
	if !browser && !hasURL && !hasChain {
		diags.AddAttributeError(
			path.Root("url"),
			"Address or Chain required",
			"A Ping needs `url`. A Chain needs `http_chain`.",
		)
	}
}

func firstStepMustBeGoto(ctx context.Context, m scenarioModel, diags *diag.Diagnostics) {
	if m.Steps.IsNull() || m.Steps.IsUnknown() {
		return
	}
	var steps []stepModel
	diags.Append(m.Steps.ElementsAs(ctx, &steps, false)...)
	if diags.HasError() || len(steps) == 0 {
		return
	}
	if steps[0].Op.ValueString() != "goto" {
		diags.AddAttributeError(
			path.Root("steps").AtListIndex(0).AtName("op"),
			"Journey must start with goto",
			"The worker opens a page before it can click or fill anything. The first step has to be `goto`.",
		)
	}
}
