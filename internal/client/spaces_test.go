package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

/*
Spaces, status pages and account: the routes the 0.4.1 API added to `/v1`.

Same two checks as for the older routes — the right method on the right path,
and every refusal surfacing — plus what is particular to them: the Space filter
of the scenario listing, a `false` that must reach the API, and quotas that may
come back without a limit.
*/

func TestModuleOperationsTargetTheRightRoute(t *testing.T) {
	ctx := context.Background()
	name := "Shop"
	cases := []struct {
		name   string
		call   func(c *Client) error
		method string
		path   string
	}{
		{"create space", func(c *Client) error {
			_, err := c.CreateSpace(ctx, SpaceInput{Name: &name})
			return err
		}, http.MethodPost, "/v1/spaces"},
		{"read space", func(c *Client) error {
			_, err := c.GetSpace(ctx, "sp-1")
			return err
		}, http.MethodGet, "/v1/spaces/sp-1"},
		{"update space", func(c *Client) error {
			_, err := c.UpdateSpace(ctx, "sp-1", SpaceInput{Name: &name})
			return err
		}, http.MethodPatch, "/v1/spaces/sp-1"},
		{"archive space", func(c *Client) error { return c.ArchiveSpace(ctx, "sp-1") }, http.MethodDelete, "/v1/spaces/sp-1"},
		{"list spaces", func(c *Client) error {
			_, err := c.ListSpaces(ctx)
			return err
		}, http.MethodGet, "/v1/spaces"},
		{"create status page", func(c *Client) error {
			_, err := c.CreateStatusPageEntry(ctx, StatusPageInput{}, "idem-status")
			return err
		}, http.MethodPost, "/v1/status-pages"},
		{"read status page", func(c *Client) error {
			_, err := c.GetStatusPageEntry(ctx, "org-1")
			return err
		}, http.MethodGet, "/v1/status-pages/org-1"},
		{"update status page", func(c *Client) error {
			_, err := c.UpdateStatusPageEntry(ctx, "org-1", StatusPageInput{})
			return err
		}, http.MethodPatch, "/v1/status-pages/org-1"},
		{"account", func(c *Client) error {
			_, err := c.GetAccount(ctx)
			return err
		}, http.MethodGet, "/v1/account"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				_, _ = w.Write([]byte(`{"id":"x"}`))
			})
			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotMethod != tc.method || gotPath != tc.path {
				t.Errorf("%s: %s %s, want %s %s", tc.name, gotMethod, gotPath, tc.method, tc.path)
			}
		})
	}
}

func TestModuleOperationsPropagateAPIErrors(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"missing scope"}`))
	})

	cases := map[string]func() error{
		"CreateSpace": func() error {
			_, err := c.CreateSpace(ctx, SpaceInput{})
			return err
		},
		"GetSpace": func() error {
			_, err := c.GetSpace(ctx, "sp-1")
			return err
		},
		"UpdateSpace": func() error {
			_, err := c.UpdateSpace(ctx, "sp-1", SpaceInput{})
			return err
		},
		"ArchiveSpace": func() error { return c.ArchiveSpace(ctx, "sp-1") },
		"ListSpaces": func() error {
			_, err := c.ListSpaces(ctx)
			return err
		},
		"ListScenariosInSpace": func() error {
			_, err := c.ListScenariosInSpace(ctx, "sp-1")
			return err
		},
		"CreateStatusPageEntry": func() error {
			_, err := c.CreateStatusPageEntry(ctx, StatusPageInput{}, "idem-status")
			return err
		},
		"GetStatusPageEntry": func() error {
			_, err := c.GetStatusPageEntry(ctx, "org-1")
			return err
		},
		"UpdateStatusPageEntry": func() error {
			_, err := c.UpdateStatusPageEntry(ctx, "org-1", StatusPageInput{})
			return err
		},
		"GetAccount": func() error {
			_, err := c.GetAccount(ctx)
			return err
		},
	}

	for name, call := range cases {
		err := call()
		if err == nil {
			t.Errorf("%s: the API refusal must surface", name)
			continue
		}
		if !strings.Contains(err.Error(), "missing scope") {
			t.Errorf("%s: message lost (%v)", name, err)
		}
	}
}

func TestScenariosOfASpaceAreFilteredByTheAPI(t *testing.T) {
	var query string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query().Get("space_id")
		_, _ = w.Write([]byte(`{"items":[{"id":"mon-1","name":"Shop","type":"http","spaceId":"sp-1"}],"nextCursor":null}`))
	})

	got, err := c.ListScenariosInSpace(context.Background(), "sp-1")
	if err != nil {
		t.Fatal(err)
	}
	if query != "sp-1" {
		t.Errorf("space_id = %q: the filter must go to the API", query)
	}
	if len(got) != 1 || got[0].SpaceID == nil || *got[0].SpaceID != "sp-1" {
		t.Errorf("scenarios = %+v", got)
	}
}

func TestUnpublishingSendsFalse(t *testing.T) {
	// `omitempty` on a plain bool would drop false, and a destroy meant to
	// unpublish would send an empty body the API refuses.
	var body map[string]any
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"id":"org-1","public":false}`))
	})

	private := false
	got, err := c.UpdateStatusPageEntry(context.Background(), "org-1", StatusPageInput{Public: &private})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := body["public"]; !ok || v != false {
		t.Errorf("body = %v: public=false must be sent", body)
	}
	if _, ok := body["slug"]; ok {
		t.Errorf("body = %v: an absent slug must not be sent, it would move the page", body)
	}
	if got.Public {
		t.Error("the response must be decoded")
	}
}

func TestAccountQuotasKeepUnlimitedApart(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "organization":{"id":"org-1","name":"Agency"},
		  "space":null,
		  "plan":{"id":"pro","name":"Pro","priceMonthlyEur":69},
		  "quotas":{
		    "ping":{"used":3,"limit":50,"remaining":47},
		    "flow":{"used":2,"limit":null,"remaining":null},
		    "runs":{"used":"12.5","limit":800,"remaining":787,"period":"month","packRunsRemaining":0},
		    "spaces":null
		  },
		  "billingUrl":"https://app.example.com/billing"
		}`))
	})

	got, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Quotas.Flow.Limit != nil || got.Quotas.Flow.Remaining != nil {
		t.Errorf("an unlimited quota must stay null, not 0: %+v", got.Quotas.Flow)
	}
	if *got.Quotas.Runs.Used.Float() != 12.5 || *got.Quotas.Runs.Period != "month" {
		t.Errorf("runs = %+v", got.Quotas.Runs)
	}
	if got.Space != nil || got.Quotas.Spaces != nil {
		t.Errorf("absent Space and Spaces quota must stay nil: %+v", got)
	}
}
