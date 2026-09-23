package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwdsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pathlyhq/terraform-provider-pathly/internal/client"
)

/*
The data sources.

A data source that fails loudly costs a plan. A data source that returns less
than it should costs resources: a `for_each` built on a truncated listing
destroys everything that disappeared from it. So what is checked here is mostly
that nothing comes back quietly empty — filters that really filter, pages that
are really walked, refusals that really surface.
*/

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

// dataHarness drives a data source the way Terraform does: schema, Configure,
// then a read against a stub API.
type dataHarness struct {
	t       *testing.T
	d       datasource.DataSource
	schema  fwdsschema.Schema
	objType tftypes.Object
}

func newDataHarness(t *testing.T, d datasource.DataSource, handler http.HandlerFunc) *dataHarness {
	t.Helper()
	ctx := context.Background()

	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	configureResp := &datasource.ConfigureResponse{}
	d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{
		ProviderData: client.New(srv.URL, "sp_test"),
	}, configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("Configure: %v", configureResp.Diagnostics)
	}

	return &dataHarness{
		t:       t,
		d:       d,
		schema:  schemaResp.Schema,
		objType: schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object),
	}
}

func (h *dataHarness) read(set map[string]tftypes.Value) *datasource.ReadResponse {
	ctx := context.Background()
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: h.schema, Raw: tftypes.NewValue(h.objType, nil)}}
	h.d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Schema: h.schema,
		Raw:    dataObject(h.objType, set),
	}}, resp)
	return resp
}

// mustRead reads and decodes into the model, failing the test on a diagnostic.
func (h *dataHarness) mustRead(set map[string]tftypes.Value, into any) {
	h.t.Helper()
	resp := h.read(set)
	if resp.Diagnostics.HasError() {
		h.t.Fatalf("Read: %v", resp.Diagnostics)
	}
	resp.State.Get(context.Background(), into)
}

// readDataSources are the twelve factories, used by the tests that have the
// same thing to say about all of them.
func readDataSources() map[string]func() datasource.DataSource {
	return map[string]func() datasource.DataSource{
		"scenarios":           NewScenariosDataSource,
		"settings":            NewSettingsDataSource,
		"usage":               NewUsageDataSource,
		"incidents":           NewIncidentsDataSource,
		"members":             NewMembersDataSource,
		"runs":                NewRunsDataSource,
		"run":                 NewRunDataSource,
		"sla":                 NewSlaDataSource,
		"sla_targets":         NewSlaTargetsDataSource,
		"webhooks":            NewWebhooksDataSource,
		"maintenance_windows": NewMaintenanceWindowsDataSource,
		"status_page":         NewStatusPageDataSource,
	}
}

// ---------------------------------------------------------------------------
// What every data source must do
// ---------------------------------------------------------------------------

// TestEveryDataSourceReportsRefusals is the one that matters most.
//
// A 403 on a missing scope must stop the plan. Were it swallowed, the listing
// would come back empty, and a `for_each` over it would plan the destruction of
// everything it no longer sees.
func TestEveryDataSourceReportsRefusals(t *testing.T) {
	for name, factory := range readDataSources() {
		t.Run(name, func(t *testing.T) {
			h := newDataHarness(t, factory(), func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"This key does not carry the required scope."}`))
			})
			if !h.read(map[string]tftypes.Value{"id": str("run-1")}).Diagnostics.HasError() {
				t.Error("a refused read must surface, not return an empty result")
			}
		})
	}
}

// A data source read before the provider is configured has no client. It must
// come back silently: the framework calls the read again once the provider is
// up, and an error here would break a legitimate plan.
func TestEveryDataSourceWaitsForItsClient(t *testing.T) {
	ctx := context.Background()
	for name, factory := range readDataSources() {
		t.Run(name, func(t *testing.T) {
			d := factory()
			schemaResp := &datasource.SchemaResponse{}
			d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
			objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

			configureResp := &datasource.ConfigureResponse{}
			d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{}, configureResp)
			if configureResp.Diagnostics.HasError() {
				t.Fatalf("Configure without data must stay silent: %v", configureResp.Diagnostics)
			}

			resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
			d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
				Schema: schemaResp.Schema,
				Raw:    dataObject(objType, nil),
			}}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("%s: read without a client must stay silent: %v", name, resp.Diagnostics)
			}
		})
	}
}

// An unreadable configuration must stop before the call. It happens when a
// state written by an older version of the provider no longer matches the
// schema, and going ahead would read with filters nobody asked for.
func TestUnreadableFilterStopsTheRead(t *testing.T) {
	ctx := context.Background()
	// Only those that take filters: the others never look at their
	// configuration, so there is nothing to fail to read.
	filtered := map[string]func() datasource.DataSource{
		"scenarios":           NewScenariosDataSource,
		"incidents":           NewIncidentsDataSource,
		"members":             NewMembersDataSource,
		"runs":                NewRunsDataSource,
		"run":                 NewRunDataSource,
		"sla":                 NewSlaDataSource,
		"maintenance_windows": NewMaintenanceWindowsDataSource,
	}

	for name, factory := range filtered {
		t.Run(name, func(t *testing.T) {
			d := factory()
			schemaResp := &datasource.SchemaResponse{}
			d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				t.Errorf("%s: no call must go out from an unreadable configuration", name)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer srv.Close()
			d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{
				ProviderData: client.New(srv.URL, "sp_test"),
			}, &datasource.ConfigureResponse{})

			// An object that does not match the schema: whatever the filters of
			// the data source, it cannot be decoded into its model.
			badType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"filter_status": tftypes.Bool}}
			bad := tftypes.NewValue(badType, map[string]tftypes.Value{
				"filter_status": tftypes.NewValue(tftypes.Bool, true),
			})

			objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
			resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
			d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: bad}}, resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("%s: an unreadable configuration must stop the read", name)
			}
		})
	}
}

// Every attribute of every data source has to be documented: the registry page
// is generated from these descriptions, and an undocumented attribute is one
// nobody uses.
func TestEveryDataSourceAttributeIsDocumented(t *testing.T) {
	ctx := context.Background()
	for name, factory := range readDataSources() {
		t.Run(name, func(t *testing.T) {
			schemaResp := &datasource.SchemaResponse{}
			factory().Schema(ctx, datasource.SchemaRequest{}, schemaResp)
			if schemaResp.Schema.Description == "" {
				t.Errorf("%s: data source without a description", name)
			}
			for attrName, attribute := range schemaResp.Schema.Attributes {
				if attribute.GetDescription() == "" {
					t.Errorf("%s.%s without a description", name, attrName)
				}
				nested, ok := attribute.(fwdsschema.ListNestedAttribute)
				if !ok {
					continue
				}
				for nestedName, nestedAttr := range nested.NestedObject.Attributes {
					if nestedAttr.GetDescription() == "" {
						t.Errorf("%s.%s.%s without a description", name, attrName, nestedName)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Settings and consumption
// ---------------------------------------------------------------------------

func TestSettingsDataSourceReadsTheOrganization(t *testing.T) {
	h := newDataHarness(t, NewSettingsDataSource(), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("a data source must not write: %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"settings":{"name":"Acme","planId":"pro","timezone":"Europe/Paris",
		  "statusPublic":true,"sslWarnDays":[30,7],"triageLatencyFactor":"2.5","hasPagerduty":true}}`))
	})

	var state settingsModel
	h.mustRead(nil, &state)

	if state.ID.ValueString() != settingsID {
		t.Errorf("id = %q, the singleton must always carry the same one", state.ID.ValueString())
	}
	if state.Name.ValueString() != "Acme" || state.PlanID.ValueString() != "pro" {
		t.Errorf("organization = %+v", state)
	}
	if state.TriageLatencyFactor.ValueFloat64() != 2.5 {
		t.Errorf("triage latency factor = %v", state.TriageLatencyFactor)
	}
	if !state.HasPagerduty.ValueBool() {
		t.Error("the PagerDuty flag must be reported")
	}
	if len(state.SSLWarnDays.Elements()) != 2 {
		t.Errorf("ssl warn days = %v", state.SSLWarnDays)
	}
}

func TestUsageDataSourceReadsTheCounters(t *testing.T) {
	h := newDataHarness(t, NewUsageDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"planId":"pro","browserRunsUsed":"12.5","packRunsUsedThisPeriod":40,"packRunsRemaining":960}`))
	})

	var state usageModel
	h.mustRead(nil, &state)

	if state.PlanID.ValueString() != "pro" {
		t.Errorf("plan = %q", state.PlanID.ValueString())
	}
	if state.BrowserRunsUsed.ValueFloat64() != 12.5 {
		t.Errorf("browser runs = %v", state.BrowserRunsUsed)
	}
	if state.PackRunsRemaining.ValueFloat64() != 960 {
		t.Errorf("remaining runs = %v", state.PackRunsRemaining)
	}
}

// ---------------------------------------------------------------------------
// Incidents
// ---------------------------------------------------------------------------

func TestIncidentsDataSourceFilters(t *testing.T) {
	body := `{"items":[
	  {"id":"inc-1","monitor_id":"mon-1","monitor_name":"Checkout","status":"open","title":"Down",
	   "opened_at":"2026-09-23T08:00:00.000Z","public_postmortem":true,"assignee_email":"ops@example.com",
	   "ado_work_item_id":4242,"ado_work_item_url":"https://dev.azure.com/x/_workitems/edit/4242"},
	  {"id":"inc-2","monitor_id":"mon-2","status":"resolved","resolved_at":"2026-09-23T09:00:00.000Z"},
	  {"id":"inc-3","monitor_id":"mon-1","status":"resolved"}
	],"nextCursor":null}`

	cases := []struct {
		name   string
		filter map[string]tftypes.Value
		want   []string
	}{
		{name: "no filter", want: []string{"inc-1", "inc-2", "inc-3"}},
		{
			name:   "by status",
			filter: map[string]tftypes.Value{"filter_status": str("resolved")},
			want:   []string{"inc-2", "inc-3"},
		},
		{
			name:   "by scenario",
			filter: map[string]tftypes.Value{"filter_scenario_id": str("mon-1")},
			want:   []string{"inc-1", "inc-3"},
		},
		{
			name: "both at once",
			filter: map[string]tftypes.Value{
				"filter_status":      str("resolved"),
				"filter_scenario_id": str("mon-1"),
			},
			want: []string{"inc-3"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newDataHarness(t, NewIncidentsDataSource(), func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			})
			var state incidentsDataSourceModel
			h.mustRead(tc.filter, &state)

			got := make([]string, 0, len(state.Incidents))
			for _, in := range state.Incidents {
				got = append(got, in.ID.ValueString())
			}
			if len(got) != len(tc.want) {
				t.Fatalf("incidents = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("incidents = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestIncidentsDataSourceCarriesTheWorkItemLink(t *testing.T) {
	h := newDataHarness(t, NewIncidentsDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"inc-1","status":"open","title":"Down",
		  "ado_work_item_id":4242,"ado_work_item_url":"https://dev.azure.com/x/_workitems/edit/4242",
		  "assignee_email":"ops@example.com","assignee_name":"Ops","postmortem":"Root cause.",
		  "opened_run_id":"run-9","resolved_run_id":"run-12"}],"nextCursor":null}`))
	})

	var state incidentsDataSourceModel
	h.mustRead(nil, &state)
	if len(state.Incidents) != 1 {
		t.Fatalf("incidents = %+v", state.Incidents)
	}
	got := state.Incidents[0]
	// The work item is what lets a run book jump straight to the ticket.
	if got.WorkItemID.ValueInt64() != 4242 || got.WorkItemURL.ValueString() == "" {
		t.Errorf("work item = %v / %q", got.WorkItemID, got.WorkItemURL.ValueString())
	}
	if got.AssigneeEmail.ValueString() != "ops@example.com" || got.Postmortem.ValueString() != "Root cause." {
		t.Errorf("incident = %+v", got)
	}
	if got.OpenedRunID.ValueString() != "run-9" || got.ResolvedRunID.ValueString() != "run-12" {
		t.Errorf("runs of the incident = %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Members
// ---------------------------------------------------------------------------

func TestMembersDataSourceFiltersByRole(t *testing.T) {
	body := `{"items":[
	  {"id":"u-1","email":"a@example.com","name":"A","role":"admin"},
	  {"id":"u-2","email":"b@example.com","name":"B","role":"viewer"},
	  {"id":"u-3","email":"c@example.com"}
	]}`

	h := newDataHarness(t, NewMembersDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	var all membersDataSourceModel
	h.mustRead(nil, &all)
	if len(all.Members) != 3 {
		t.Fatalf("members = %+v", all.Members)
	}

	h = newDataHarness(t, NewMembersDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	var admins membersDataSourceModel
	h.mustRead(map[string]tftypes.Value{"filter_role": str("admin")}, &admins)
	// A member with no role must not pass an explicit filter: an access review
	// that counted them as an administrator would be wrong.
	if len(admins.Members) != 1 || admins.Members[0].ID.ValueString() != "u-1" {
		t.Errorf("administrators = %+v", admins.Members)
	}

	h = newDataHarness(t, NewMembersDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	var unknown membersDataSourceModel
	// The role is not validated against a list on purpose: a role added later
	// on the Pathly side must be usable without waiting for a release.
	h.mustRead(map[string]tftypes.Value{"filter_role": str("billing")}, &unknown)
	if len(unknown.Members) != 0 {
		t.Errorf("unknown role = %+v", unknown.Members)
	}
}

// ---------------------------------------------------------------------------
// Runs
// ---------------------------------------------------------------------------

func TestRunsDataSourceFiltersLocallyAndServerSide(t *testing.T) {
	var query string
	h := newDataHarness(t, NewRunsDataSource(), func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"items":[
		  {"id":"run-1","monitor_id":"mon-1","status":"ok","latency_ms":120},
		  {"id":"run-2","monitor_id":"mon-1","status":"fail","latency_ms":9000,"message":"timeout",
		   "triage_verdict":"real","browser_seconds":"3.25","screenshot_path":"s/1.png","video_path":"v/1.webm"},
		  {"id":"run-3","monitor_id":"mon-1"}
		],"nextCursor":null}`))
	})

	var state runsDataSourceModel
	h.mustRead(map[string]tftypes.Value{
		"filter_scenario_id": str("mon-1"),
		"filter_status":      str("fail"),
	}, &state)

	// The scenario goes to the API, the status is applied here: only the first
	// one lowers the number of pages read.
	if !strings.Contains(query, "scenarioId=mon-1") {
		t.Errorf("query = %q", query)
	}
	if len(state.Runs) != 1 || state.Runs[0].ID.ValueString() != "run-2" {
		t.Fatalf("runs = %+v", state.Runs)
	}
	failed := state.Runs[0]
	if failed.LatencyMs.ValueInt64() != 9000 || failed.Message.ValueString() != "timeout" {
		t.Errorf("failed run = %+v", failed)
	}
	if failed.BrowserSeconds.ValueFloat64() != 3.25 {
		t.Errorf("browser seconds = %v", failed.BrowserSeconds)
	}
	if failed.ScreenshotPath.ValueString() == "" || failed.VideoPath.ValueString() == "" {
		t.Error("the capture and the recording are what make a failure readable")
	}
}

func TestSingleRunDataSourceReadsByIdentifier(t *testing.T) {
	var path string
	h := newDataHarness(t, NewRunDataSource(), func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"run-1","monitor_id":"mon-1","monitor_name":"Checkout","status":"ok",
		  "latency_ms":320,"probe_region":"eu-west","in_maintenance":true,
		  "compared_to_run_id":"run-0","confirms_run_id":"run-2","triage_reason":"latency within range",
		  "created_at":"2026-09-23T08:00:00.000Z"}`))
	})

	var state runEntry
	h.mustRead(map[string]tftypes.Value{"id": str("run-1")}, &state)

	if path != "/v1/runs/run-1" {
		t.Errorf("path = %q", path)
	}
	if state.LatencyMs.ValueInt64() != 320 || state.ProbeRegion.ValueString() != "eu-west" {
		t.Errorf("run = %+v", state)
	}
	if !state.InMaintenance.ValueBool() {
		t.Error("a run inside a window must say so: it does not count against the SLA")
	}
	if state.ComparedToRunID.ValueString() != "run-0" || state.ConfirmsRunID.ValueString() != "run-2" {
		t.Errorf("triage chain = %+v", state)
	}
	// Absent from this projection: it must stay null rather than become an empty
	// string that reads like a type.
	if !state.ScenarioType.IsNull() {
		t.Errorf("scenario type = %v", state.ScenarioType)
	}
}

// ---------------------------------------------------------------------------
// Availability objectives
// ---------------------------------------------------------------------------

func TestSlaDataSourceFlattensTheMeasurement(t *testing.T) {
	body := `{"items":[
	  {"id":"sla-1","monitorId":"mon-1","monitorName":"Checkout","name":"Checkout 99.9",
	   "objectivePct":99.9,"windowDays":30,"excludeMaintenance":true,"warnAtBudgetRatio":0.8,"enabled":true,
	   "status":{"uptimePct":"99.95","eligibleRuns":8640,"okRuns":8636,"failedRuns":4,"excludedRuns":12,
	             "errorBudgetRuns":9,"errorBudgetUsedRatio":"0.444","estimatedDowntimeMinutes":20,"state":"ok"}},
	  {"id":"sla-2","monitorId":"mon-2","name":"API 99.5","objectivePct":99.5,
	   "status":{"state":"breached","uptimePct":"98.1"}},
	  {"id":"sla-3","monitorId":"mon-3","name":"Never measured"}
	]}`

	h := newDataHarness(t, NewSlaDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	var state slaDataSourceModel
	h.mustRead(nil, &state)
	if len(state.Targets) != 3 {
		t.Fatalf("targets = %+v", state.Targets)
	}

	measured := state.Targets[0]
	if measured.UptimePct.ValueFloat64() != 99.95 || measured.State.ValueString() != "ok" {
		t.Errorf("measurement = %+v", measured)
	}
	if measured.OkRuns.ValueInt64() != 8636 || measured.ExcludedRuns.ValueInt64() != 12 {
		t.Errorf("run counts = %+v", measured)
	}
	if measured.ErrorBudgetUsedRatio.ValueFloat64() != 0.444 {
		t.Errorf("budget used = %v", measured.ErrorBudgetUsedRatio)
	}
	if measured.EstimatedDowntimeMinutes.ValueFloat64() != 20 {
		t.Errorf("estimated downtime = %v", measured.EstimatedDowntimeMinutes)
	}
	if measured.ScenarioName.ValueString() != "Checkout" || !measured.ExcludeMaintenance.ValueBool() {
		t.Errorf("definition = %+v", measured)
	}

	// A target never measured keeps its definition and leaves the measurement
	// null. A zero there would read as a total outage, and an alert built on it
	// would fire on the day the target was created.
	fresh := state.Targets[2]
	if fresh.Name.ValueString() != "Never measured" {
		t.Errorf("definition lost: %+v", fresh)
	}
	for name, value := range map[string]bool{
		"state":         fresh.State.IsNull(),
		"uptime_pct":    fresh.UptimePct.IsNull(),
		"eligible_runs": fresh.EligibleRuns.IsNull(),
		"failed_runs":   fresh.FailedRuns.IsNull(),
	} {
		if !value {
			t.Errorf("%s must stay null as long as nothing has been measured", name)
		}
	}
}

func TestSlaDataSourceFiltersByScenarioAndState(t *testing.T) {
	body := `{"items":[
	  {"id":"sla-1","monitorId":"mon-1","status":{"state":"ok"}},
	  {"id":"sla-2","monitorId":"mon-2","status":{"state":"breached"}},
	  {"id":"sla-3","monitorId":"mon-1","status":{"state":"breached"}},
	  {"id":"sla-4"}
	]}`

	cases := []struct {
		name   string
		filter map[string]tftypes.Value
		want   []string
	}{
		{
			name:   "by scenario",
			filter: map[string]tftypes.Value{"filter_scenario_id": str("mon-1")},
			want:   []string{"sla-1", "sla-3"},
		},
		{
			name:   "by state",
			filter: map[string]tftypes.Value{"filter_state": str("breached")},
			want:   []string{"sla-2", "sla-3"},
		},
		{
			name: "both at once",
			filter: map[string]tftypes.Value{
				"filter_scenario_id": str("mon-1"),
				"filter_state":       str("breached"),
			},
			want: []string{"sla-3"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newDataHarness(t, NewSlaDataSource(), func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			})
			var state slaDataSourceModel
			h.mustRead(tc.filter, &state)

			got := make([]string, 0, len(state.Targets))
			for _, target := range state.Targets {
				got = append(got, target.ID.ValueString())
			}
			if len(got) != len(tc.want) {
				t.Fatalf("targets = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("targets = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestSlaTargetsDataSourceReadsTheDefinitions(t *testing.T) {
	h := newDataHarness(t, NewSlaTargetsDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[
		  {"id":"sla-1","monitorId":"mon-1","name":"Checkout 99.9","objectivePct":99.9,"windowDays":30,
		   "excludeMaintenance":true,"warnAtBudgetRatio":0.8,"enabled":true}
		]}`))
	})

	var state slaTargetsDataSourceModel
	h.mustRead(nil, &state)
	if len(state.Targets) != 1 {
		t.Fatalf("targets = %+v", state.Targets)
	}
	got := state.Targets[0]
	if got.ObjectivePct.ValueFloat64() != 99.9 || got.WindowDays.ValueInt64() != 30 {
		t.Errorf("objective = %+v", got)
	}
	if got.WarnAtBudgetRatio.ValueFloat64() != 0.8 || !got.Enabled.ValueBool() {
		t.Errorf("target = %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Webhooks, maintenance, status page
// ---------------------------------------------------------------------------

func TestWebhooksDataSourceNeverReturnsTheDestination(t *testing.T) {
	h := newDataHarness(t, NewWebhooksDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		// The API answers a fingerprint, never the URL. Even if it did one day,
		// the data source must not carry it into the state.
		_, _ = w.Write([]byte(`{"items":[{"id":"h-1","events":["run.failed","incident.opened"],
		  "enabled":true,"hasSecret":true,"urlFingerprint":"sha256:abcd",
		  "url":"https://hooks.example.com/should-not-appear",
		  "createdAt":"2026-09-23T08:00:00.000Z"}]}`))
	})

	var state webhooksDataSourceModel
	h.mustRead(nil, &state)
	if len(state.Webhooks) != 1 {
		t.Fatalf("webhooks = %+v", state.Webhooks)
	}
	got := state.Webhooks[0]
	if got.URLFingerprint.ValueString() != "sha256:abcd" {
		t.Errorf("fingerprint = %q", got.URLFingerprint.ValueString())
	}
	if !got.HasSecret.ValueBool() || !got.Enabled.ValueBool() {
		t.Errorf("webhook = %+v", got)
	}
	if len(got.Events.Elements()) != 2 {
		t.Errorf("events = %v", got.Events)
	}
	// A state file is read by anyone who can read the backend: no destination
	// must be recoverable from it.
	for _, attribute := range []string{"url", "secret"} {
		schemaResp := &datasource.SchemaResponse{}
		NewWebhooksDataSource().Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
		nested := schemaResp.Schema.Attributes["webhooks"].(fwdsschema.ListNestedAttribute)
		if _, present := nested.NestedObject.Attributes[attribute]; present {
			t.Errorf("the listing must not expose %s", attribute)
		}
	}
}

func TestMaintenanceWindowsDataSourceFiltersByScenario(t *testing.T) {
	body := `{"items":[
	  {"id":"mw-1","monitorId":"mon-1","weekday":6,"startMinute":120,"durationMin":90},
	  {"id":"mw-2","monitorId":"mon-2","startsAt":"2026-10-01T02:00:00.000Z",
	   "endsAt":"2026-10-01T04:00:00.000Z","reason":"migration"},
	  {"id":"mw-3"}
	]}`

	h := newDataHarness(t, NewMaintenanceWindowsDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	var all maintenanceWindowsDataSourceModel
	h.mustRead(nil, &all)
	if len(all.Windows) != 3 {
		t.Fatalf("windows = %+v", all.Windows)
	}
	// Recurring and one-off windows are read through the same shape: the
	// attributes of the other kind stay null.
	if all.Windows[0].Weekday.ValueInt64() != 6 || !all.Windows[0].StartsAt.IsNull() {
		t.Errorf("recurring window = %+v", all.Windows[0])
	}
	if all.Windows[1].Reason.ValueString() != "migration" || !all.Windows[1].Weekday.IsNull() {
		t.Errorf("one-off window = %+v", all.Windows[1])
	}

	h = newDataHarness(t, NewMaintenanceWindowsDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	var filtered maintenanceWindowsDataSourceModel
	h.mustRead(map[string]tftypes.Value{"filter_scenario_id": str("mon-1")}, &filtered)
	if len(filtered.Windows) != 1 || filtered.Windows[0].ID.ValueString() != "mw-1" {
		t.Errorf("filtering = %+v", filtered.Windows)
	}
}

func TestStatusPageDataSourceReadsComponents(t *testing.T) {
	h := newDataHarness(t, NewStatusPageDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"components":[
		  {"name":"Checkout","uptime30d":99.98,"uptime90d":"99.9"},
		  {"name":"API","uptime30d":100}
		],"uptime30d":99.99}`))
	})

	var state statusPageModel
	h.mustRead(nil, &state)
	if len(state.Components) != 2 {
		t.Fatalf("components = %+v", state.Components)
	}
	if state.Uptime30d.ValueFloat64() != 99.99 {
		t.Errorf("overall uptime = %v", state.Uptime30d)
	}
	if state.Components[0].Uptime90d.ValueFloat64() != 99.9 {
		t.Errorf("90-day uptime = %v", state.Components[0].Uptime90d)
	}
	// Absent rather than zero: a component without 90 days of history has not
	// been down.
	if !state.Components[1].Uptime90d.IsNull() {
		t.Errorf("missing 90-day uptime = %v", state.Components[1].Uptime90d)
	}
}

func TestStatusPageDataSourceSaysWhenThereIsNone(t *testing.T) {
	h := newDataHarness(t, NewStatusPageDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Aucune status page publique"}`))
	})

	resp := h.read(nil)
	if !resp.Diagnostics.HasError() {
		t.Fatal("no status page must be said, not read as a page with no component")
	}
	// The message has to name the setting to change, otherwise the reader has to
	// go and look for why the page is missing.
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "status_public") {
		t.Errorf("detail = %q, it must point at the setting", resp.Diagnostics.Errors()[0].Detail())
	}
}
