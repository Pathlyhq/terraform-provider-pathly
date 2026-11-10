package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

/*
Spaces, status pages and account — the surface the 0.4.1 API opened.

What can go wrong here and costs for real: a destroy that deletes what the API
only archives or unpublishes, a status page whose address changes without the
plan saying so, and a Space-scoped key that files a scenario in its Space while
the plan announces it outside.
*/

// ---------------------------------------------------------------------------
// pathly_space
// ---------------------------------------------------------------------------

const spaceJSON = `{"id":"sp-1","name":"Client A","slug":"client-a","kind":"client",
  "statusPublic":false,"createdAt":"2026-11-02T10:00:00.000Z","updatedAt":"2026-11-02T10:00:00.000Z"}`

func TestSpaceCreateStoresWhatTheAPIReturns(t *testing.T) {
	ctx := context.Background()
	r := NewSpaceResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(spaceJSON))
	})

	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name": str("Client A"),
		"kind": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	var state spaceModel
	resp.State.Get(ctx, &state)
	if state.ID.ValueString() != "sp-1" || state.Slug.ValueString() != "client-a" || state.Kind.ValueString() != "client" {
		t.Errorf("state = %+v", state)
	}
	if h.bodies[0]["name"] != "Client A" {
		t.Errorf("body = %v", h.bodies[0])
	}
	// An unknown kind is left to the API, which derives it from the profile.
	if _, sent := h.bodies[0]["kind"]; sent {
		t.Errorf("body = %v: an omitted kind must not be sent", h.bodies[0])
	}
}

func TestSpaceCreateReportsRefusal(t *testing.T) {
	r := NewSpaceResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"Un Espace porte déjà ce nom."}`))
	})
	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(context.Background(), resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{"name": str("Client A")})}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused creation must surface")
	}
}

func TestSpaceReadRefreshesAndDropsArchived(t *testing.T) {
	ctx := context.Background()
	r := NewSpaceResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"sp-1","name":"Client A2","slug":"client-a","kind":"client",
		  "statusPublic":true,"sites":3,"scenarioIds":["a","b","c"]}`))
	})
	resp := &resource.ReadResponse{State: h.state(map[string]tftypes.Value{"id": str("sp-1"), "name": str("Client A")})}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, resp)
	var state spaceModel
	resp.State.Get(ctx, &state)
	if state.Name.ValueString() != "Client A2" || !state.StatusPublic.ValueBool() {
		t.Errorf("a rename from the console must show in the next plan: %+v", state)
	}

	gone := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Espace introuvable"}`))
	})
	resp = &resource.ReadResponse{State: gone.state(map[string]tftypes.Value{"id": str("sp-1")})}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, resp)
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Errorf("an archived Space must leave the state: %v", resp.Diagnostics)
	}
}

func TestSpaceUpdateRenames(t *testing.T) {
	ctx := context.Background()
	r := NewSpaceResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"sp-1","name":"Client B","slug":"client-a","kind":"site","statusPublic":false}`))
	})
	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  h.plan(map[string]tftypes.Value{"id": str("sp-1"), "name": str("Client B"), "kind": str("site")}),
		State: h.state(map[string]tftypes.Value{"id": str("sp-1"), "name": str("Client A")}),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if h.requests[0] != "PATCH /v1/spaces/sp-1" || h.bodies[0]["name"] != "Client B" || h.bodies[0]["kind"] != "site" {
		t.Errorf("requests = %v, bodies = %v", h.requests, h.bodies)
	}
	var state spaceModel
	resp.State.Get(ctx, &state)
	if state.Name.ValueString() != "Client B" {
		t.Errorf("state = %+v", state)
	}
}

func TestSpaceUpdateReportsVanishedAndRefused(t *testing.T) {
	for status, want := range map[int]string{
		http.StatusNotFound:  "Space vanished",
		http.StatusForbidden: "Space update refused",
	} {
		r := NewSpaceResource()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"no"}`))
		})
		resp := &resource.UpdateResponse{State: h.emptyState()}
		r.Update(context.Background(), resource.UpdateRequest{
			Plan:  h.plan(map[string]tftypes.Value{"id": str("sp-1"), "name": str("Client B")}),
			State: h.state(map[string]tftypes.Value{"id": str("sp-1"), "name": str("Client A")}),
		}, resp)
		if !resp.Diagnostics.HasError() || resp.Diagnostics.Errors()[0].Summary() != want {
			t.Errorf("HTTP %d: diagnostics = %v, want %q", status, resp.Diagnostics, want)
		}
	}
}

func TestSpaceDeleteArchivesAndToleratesAlreadyGone(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		r := NewSpaceResource()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"ok":true,"archived":"sp-1"}`))
		})
		resp := &resource.DeleteResponse{State: h.emptyState()}
		r.Delete(context.Background(), resource.DeleteRequest{State: h.state(map[string]tftypes.Value{"id": str("sp-1")})}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("HTTP %d: %v", status, resp.Diagnostics)
		}
		if h.requests[0] != "DELETE /v1/spaces/sp-1" {
			t.Errorf("requests = %v", h.requests)
		}
	}
}

// ---------------------------------------------------------------------------
// pathly_status_page
// ---------------------------------------------------------------------------

const spacePageJSON = `{"id":"sp-1","kind":"space","spaceId":"sp-1","name":"Client A",
  "slug":"agency/client-a","public":true,"url":"https://app.example.com/status/agency/client-a",
  "badgeUrl":"https://api.example.com/public/status/agency/client-a/badge.svg"}`

const orgPageJSON = `{"id":"org-1","kind":"organization","spaceId":null,"name":"Agency",
  "slug":"agency","public":true,"url":"https://app.example.com/status/agency",
  "badgeUrl":"https://api.example.com/public/status/agency/badge.svg"}`

func TestStatusPageCreatePublishesASpacePage(t *testing.T) {
	ctx := context.Background()
	r := NewStatusPageResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Idempotency-Key") == "" {
			t.Error("the creation must carry an idempotency key")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(spacePageJSON))
	})
	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"space_id": str("sp-1"),
		"public":   boolean(true),
		"slug":     tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if h.bodies[0]["spaceId"] != "sp-1" || h.bodies[0]["public"] != true {
		t.Errorf("body = %v", h.bodies[0])
	}
	if _, sent := h.bodies[0]["slug"]; sent {
		t.Errorf("body = %v: a Space page has no slug of its own", h.bodies[0])
	}
	var state statusPageResourceModel
	resp.State.Get(ctx, &state)
	if state.Kind.ValueString() != "space" || !strings.HasSuffix(state.BadgeURL.ValueString(), "/badge.svg") {
		t.Errorf("state = %+v", state)
	}
}

func TestStatusPageCreatePointsAtImportOnConflict(t *testing.T) {
	r := NewStatusPageResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"La page de statut existe déjà à l'adresse agency."}`))
	})
	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(context.Background(), resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{"slug": str("other")})}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a refused creation must surface")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "organization_id") {
		t.Errorf("detail = %q: it must say how to adopt the page", resp.Diagnostics.Errors()[0].Detail())
	}
}

func TestStatusPageReadRefreshesAndDropsMissing(t *testing.T) {
	ctx := context.Background()
	r := NewStatusPageResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(orgPageJSON))
	})
	resp := &resource.ReadResponse{State: h.state(map[string]tftypes.Value{"id": str("org-1")})}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, resp)
	var state statusPageResourceModel
	resp.State.Get(ctx, &state)
	if state.Slug.ValueString() != "agency" || !state.SpaceID.IsNull() || !state.Public.ValueBool() {
		t.Errorf("state = %+v", state)
	}

	gone := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Page de statut introuvable"}`))
	})
	resp = &resource.ReadResponse{State: gone.state(map[string]tftypes.Value{"id": str("sp-1")})}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, resp)
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Errorf("a page that no longer exists must leave the state: %v", resp.Diagnostics)
	}
}

func TestStatusPageUpdateOnlyMovesTheAddressWhenAsked(t *testing.T) {
	ctx := context.Background()
	current := map[string]tftypes.Value{"id": str("org-1"), "slug": str("agency"), "public": boolean(true)}

	r := NewStatusPageResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Replace(orgPageJSON, `"public":true`, `"public":false`, 1)))
	})
	resp := &resource.UpdateResponse{State: h.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  h.plan(map[string]tftypes.Value{"id": str("org-1"), "slug": str("agency"), "public": boolean(false)}),
		State: h.state(current),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if _, sent := h.bodies[0]["slug"]; sent || h.bodies[0]["public"] != false {
		t.Errorf("body = %v: an unchanged slug must stay out, it would move the page", h.bodies[0])
	}

	moved := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.ReplaceAll(orgPageJSON, "agency", "studio")))
	})
	resp = &resource.UpdateResponse{State: moved.emptyState()}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  moved.plan(map[string]tftypes.Value{"id": str("org-1"), "slug": str("studio"), "public": boolean(true)}),
		State: moved.state(current),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if moved.bodies[0]["slug"] != "studio" {
		t.Errorf("body = %v: a changed slug must be sent", moved.bodies[0])
	}
	var state statusPageResourceModel
	resp.State.Get(ctx, &state)
	if !strings.Contains(state.URL.ValueString(), "/status/studio") {
		t.Errorf("url = %q", state.URL.ValueString())
	}
}

func TestStatusPageUpdateReportsVanishedAndRefused(t *testing.T) {
	for status, want := range map[int]string{
		http.StatusNotFound:   "Status page vanished",
		http.StatusBadRequest: "Status page update refused",
	} {
		r := NewStatusPageResource()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"no"}`))
		})
		resp := &resource.UpdateResponse{State: h.emptyState()}
		r.Update(context.Background(), resource.UpdateRequest{
			Plan:  h.plan(map[string]tftypes.Value{"id": str("org-1"), "public": boolean(true)}),
			State: h.state(map[string]tftypes.Value{"id": str("org-1"), "public": boolean(false)}),
		}, resp)
		if !resp.Diagnostics.HasError() || resp.Diagnostics.Errors()[0].Summary() != want {
			t.Errorf("HTTP %d: diagnostics = %v, want %q", status, resp.Diagnostics, want)
		}
	}
}

func TestStatusPageDestroyUnpublishesAndSaysSo(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		r := NewStatusPageResource()
		h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(orgPageJSON))
		})
		resp := &resource.DeleteResponse{State: h.emptyState()}
		r.Delete(context.Background(), resource.DeleteRequest{State: h.state(map[string]tftypes.Value{"id": str("org-1")})}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("HTTP %d: %v", status, resp.Diagnostics)
		}
		if h.requests[0] != "PATCH /v1/status-pages/org-1" || h.bodies[0]["public"] != false {
			t.Errorf("requests = %v, bodies = %v: a destroy must unpublish", h.requests, h.bodies)
		}
		if resp.Diagnostics.WarningsCount() != 1 {
			t.Errorf("HTTP %d: the destroy must say the page still exists: %v", status, resp.Diagnostics)
		}
	}
}

func TestStatusPageAddressesFollowTheSlug(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	NewStatusPageResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
	s := schemaResp.Schema
	objType := s.Type().TerraformType(ctx).(tftypes.Object)
	unknown := types.StringUnknown()
	known := types.StringValue("https://app.example.com/status/agency")
	stateSlug := map[string]tftypes.Value{"slug": str("agency")}

	run := func(configSlug tftypes.Value, state map[string]tftypes.Value, stateValue, planValue types.String) types.String {
		t.Helper()
		resp := &planmodifier.StringResponse{PlanValue: planValue}
		keepUnlessSlugChanges{}.PlanModifyString(ctx, planmodifier.StringRequest{
			Path:       path.Root("url"),
			Config:     tfsdk.Config{Schema: s, Raw: object(objType, map[string]tftypes.Value{"slug": configSlug})},
			State:      tfsdk.State{Schema: s, Raw: object(objType, state)},
			StateValue: stateValue,
			PlanValue:  planValue,
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("modifier: %v", resp.Diagnostics)
		}
		return resp.PlanValue
	}

	nullSlug := tftypes.NewValue(tftypes.String, nil)
	if got := run(nullSlug, stateSlug, known, unknown); !got.Equal(known) {
		t.Errorf("slug left out: url = %v, want the known one", got)
	}
	if got := run(str("agency"), stateSlug, known, unknown); !got.Equal(known) {
		t.Errorf("slug unchanged: url = %v, want the known one", got)
	}
	if got := run(str("studio"), stateSlug, known, unknown); !got.IsUnknown() {
		t.Errorf("slug changed: url = %v, it must be announced unknown", got)
	}
	if got := run(str("agency"), nil, types.StringNull(), unknown); !got.IsUnknown() {
		t.Errorf("creation: url = %v, it must stay unknown", got)
	}
	if got := run(str("agency"), stateSlug, known, known); !got.Equal(known) {
		t.Errorf("known plan: url = %v, it must not be touched", got)
	}

	// A configuration the schema cannot decode stops the plan rather than
	// guessing.
	badType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"slug": tftypes.Bool}}
	resp := &planmodifier.StringResponse{PlanValue: unknown}
	keepUnlessSlugChanges{}.PlanModifyString(ctx, planmodifier.StringRequest{
		Config:     tfsdk.Config{Schema: s, Raw: tftypes.NewValue(badType, map[string]tftypes.Value{"slug": tftypes.NewValue(tftypes.Bool, true)})},
		State:      tfsdk.State{Schema: s, Raw: object(objType, stateSlug)},
		StateValue: known,
		PlanValue:  unknown,
	}, resp)
	if !resp.Diagnostics.HasError() || !resp.PlanValue.IsUnknown() {
		t.Errorf("an unreadable configuration must fail: %v", resp.Diagnostics)
	}

	m := keepUnlessSlugChanges{}
	if m.MarkdownDescription(ctx) != m.Description(ctx) {
		t.Error("both descriptions must say the same thing")
	}
}

// ---------------------------------------------------------------------------
// space_id on scenarios
// ---------------------------------------------------------------------------

func TestScenarioCarriesItsSpace(t *testing.T) {
	ctx := context.Background()
	r := NewScenarioResource()
	h := newHarness(t, r, func(_ *harness, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(strings.Replace(scenarioJSON, `"lastStatus"`, `"spaceId":"sp-1","lastStatus"`, 1)))
	})
	resp := &resource.CreateResponse{State: h.emptyState()}
	r.Create(ctx, resource.CreateRequest{Plan: h.plan(map[string]tftypes.Value{
		"name":         str("Checkout"),
		"url":          str("https://shop.example.com/cart"),
		"interval_sec": num(300),
		"space_id":     str("sp-1"),
	})}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	if h.bodies[0]["spaceId"] != "sp-1" {
		t.Errorf("body = %v", h.bodies[0])
	}
	var state scenarioModel
	resp.State.Get(ctx, &state)
	if state.SpaceID.ValueString() != "sp-1" {
		t.Errorf("space_id = %v", state.SpaceID)
	}
}

func TestScenariosDataSourceFiltersBySpaceOnTheServer(t *testing.T) {
	var query string
	h := newDataHarness(t, NewScenariosDataSource(), func(w http.ResponseWriter, req *http.Request) {
		query = req.URL.Query().Get("space_id")
		_, _ = w.Write([]byte(`{"items":[{"id":"mon-1","name":"Shop","type":"http","spaceId":"sp-1"}],"nextCursor":null}`))
	})
	var state scenariosDataSourceModel
	h.mustRead(map[string]tftypes.Value{"filter_space_id": str("sp-1")}, &state)
	if query != "sp-1" {
		t.Errorf("space_id = %q: the filter must go to the API", query)
	}
	if len(state.Scenarios) != 1 || state.Scenarios[0].SpaceID.ValueString() != "sp-1" {
		t.Errorf("scenarios = %+v", state.Scenarios)
	}
}

// ---------------------------------------------------------------------------
// pathly_spaces and pathly_account
// ---------------------------------------------------------------------------

func TestSpacesDataSourceReadsTheListAndTheLimit(t *testing.T) {
	h := newDataHarness(t, NewSpacesDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[` + strings.Replace(spaceJSON, `"statusPublic":false`, `"statusPublic":false,"sites":4`, 1) + `],"maxSpaces":25}`))
	})
	var state spacesDataSourceModel
	h.mustRead(nil, &state)
	if state.MaxSpaces.ValueInt64() != 25 || len(state.Spaces) != 1 {
		t.Fatalf("state = %+v", state)
	}
	if state.Spaces[0].ScenarioCount.ValueInt64() != 4 || state.Spaces[0].Slug.ValueString() != "client-a" {
		t.Errorf("space = %+v", state.Spaces[0])
	}

	unlimited := newDataHarness(t, NewSpacesDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[],"maxSpaces":null}`))
	})
	unlimited.mustRead(nil, &state)
	if !state.MaxSpaces.IsNull() {
		t.Errorf("an unlimited plan must read as null, not 0: %v", state.MaxSpaces)
	}
}

func TestSpacesDataSourceSaysWhenSpacesAreClosed(t *testing.T) {
	h := newDataHarness(t, NewSpacesDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Ressource introuvable"}`))
	})
	resp := h.read(nil)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "not open") {
		t.Errorf("closed Spaces must be said, not read as an empty list: %v", resp.Diagnostics)
	}
}

func TestAccountDataSourceReadsAKeyOfASpace(t *testing.T) {
	h := newDataHarness(t, NewAccountDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "organization":{"id":"org-1","name":"Agency"},
		  "space":{"id":"sp-1","name":"Client A"},
		  "plan":{"id":"pro","name":"Pro","priceMonthlyEur":69},
		  "quotas":{
		    "ping":{"used":3,"limit":50,"remaining":40},
		    "chain":{"used":0,"limit":20,"remaining":20},
		    "flow":{"used":2,"limit":null,"remaining":null},
		    "runs":{"used":12,"limit":800,"remaining":700,"period":"month","packRunsRemaining":100},
		    "assistantTrials":{"used":1,"limit":15,"remaining":14,"period":"day"},
		    "spaces":null
		  },
		  "billingUrl":"https://app.example.com/billing"
		}`))
	})
	var state accountModel
	h.mustRead(nil, &state)
	if state.OrganizationID.ValueString() != "org-1" || state.SpaceID.ValueString() != "sp-1" || state.SpaceName.ValueString() != "Client A" {
		t.Errorf("identity = %+v", state)
	}
	if state.PlanPriceMonthlyEur.ValueFloat64() != 69 || state.BillingURL.IsNull() {
		t.Errorf("plan = %+v", state)
	}
	q := state.Quotas
	if q.Ping.Remaining.ValueFloat64() != 40 || !q.Flow.Limit.IsNull() || q.Runs.PackRunsRemaining.ValueFloat64() != 100 {
		t.Errorf("quotas = %+v", q)
	}
	if q.AssistantTrials.Period.ValueString() != "day" || q.Spaces != nil {
		t.Errorf("quotas = %+v", q)
	}
}

func TestAccountDataSourceReadsTheWholeOrganization(t *testing.T) {
	h := newDataHarness(t, NewAccountDataSource(), func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"organization":{"id":"org-1","name":"Agency"},"space":null,
		  "plan":{"id":"business","name":"Business","priceMonthlyEur":199},
		  "quotas":{"spaces":{"used":4,"limit":null,"remaining":null}},
		  "billingUrl":"https://app.example.com/billing"}`))
	})
	var state accountModel
	h.mustRead(nil, &state)
	if !state.SpaceID.IsNull() || !state.SpaceName.IsNull() {
		t.Errorf("a key of the whole organization has no Space: %+v", state)
	}
	if state.Quotas.Spaces == nil || state.Quotas.Spaces.Used.ValueFloat64() != 4 || !state.Quotas.Spaces.Limit.IsNull() {
		t.Errorf("spaces quota = %+v", state.Quotas.Spaces)
	}
}
