package client

import (
	"context"
	"net/http"
	"net/url"
)

// ---------------------------------------------------------------------------
// Spaces
// ---------------------------------------------------------------------------

// Space groups the scenarios of one client, one brand or one application.
//
// The API answers 404 on every `/v1/spaces` route while Spaces are not open
// for the organization: the provider then reports the resource as missing,
// exactly as it would for a Space archived from the console.
type Space struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Slug         *string `json:"slug"`
	Kind         *string `json:"kind"`
	StatusPublic *bool   `json:"statusPublic"`
	// Sites is the number of scenarios filed in the Space. The update response
	// leaves it out, which is why the resource does not expose it.
	Sites       *int64   `json:"sites"`
	ScenarioIDs []string `json:"scenarioIds"`
	CreatedAt   *string  `json:"createdAt"`
}

// SpaceInput serves both the create and the update. A nil field is not sent,
// and the API keeps the existing value.
type SpaceInput struct {
	Name *string `json:"name,omitempty"`
	Kind *string `json:"kind,omitempty"`
}

// SpaceList is the listing, with the limit of the plan. A nil MaxSpaces means
// the plan has no limit.
type SpaceList struct {
	Items     []Space `json:"items"`
	MaxSpaces *int64  `json:"maxSpaces"`
}

// CreateSpace carries no idempotency key: the API does not replay this route.
// A retry after a lost response is still safe, because two active Spaces of an
// organization cannot share a name — the second attempt is refused with a 409
// instead of producing a duplicate.
func (c *Client) CreateSpace(ctx context.Context, in SpaceInput) (*Space, error) {
	out := &Space{}
	err := c.do(ctx, request{method: http.MethodPost, path: "/v1/spaces", body: in, out: out})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetSpace(ctx context.Context, id string) (*Space, error) {
	out := &Space{}
	err := c.do(ctx, request{method: http.MethodGet, path: "/v1/spaces/" + url.PathEscape(id), out: out})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UpdateSpace(ctx context.Context, id string, in SpaceInput) (*Space, error) {
	out := &Space{}
	err := c.do(ctx, request{
		method: http.MethodPatch,
		path:   "/v1/spaces/" + url.PathEscape(id),
		body:   in,
		out:    out,
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ArchiveSpace is what DELETE does on this route: the Space is archived, its
// scenarios leave it and keep running. Nothing is deleted.
func (c *Client) ArchiveSpace(ctx context.Context, id string) error {
	return c.do(ctx, request{method: http.MethodDelete, path: "/v1/spaces/" + url.PathEscape(id)})
}

// ListSpaces returns the active Spaces. The API sends the whole list at once.
func (c *Client) ListSpaces(ctx context.Context) (*SpaceList, error) {
	out := &SpaceList{}
	if err := c.do(ctx, request{method: http.MethodGet, path: "/v1/spaces", out: out}); err != nil {
		return nil, err
	}
	return out, nil
}

// ListScenariosInSpace walks the scenarios of one Space, filtered by the API.
func (c *Client) ListScenariosInSpace(ctx context.Context, spaceID string) ([]Scenario, error) {
	return listPaged[Scenario](ctx, c, "/v1/scenarios", url.Values{"space_id": []string{spaceID}})
}

// ---------------------------------------------------------------------------
// Status pages
// ---------------------------------------------------------------------------

// StatusPageEntry is a status page as `/v1/status-pages` describes it: the
// page of the organization (`kind: organization`, its identifier is the
// organization's) or the page of a Space (`kind: space`, its identifier is the
// Space's).
//
// Not to be confused with StatusPage, which is what the public page displays.
type StatusPageEntry struct {
	ID       string  `json:"id"`
	Kind     string  `json:"kind"`
	SpaceID  *string `json:"spaceId"`
	Name     string  `json:"name"`
	Slug     string  `json:"slug"`
	Public   bool    `json:"public"`
	URL      string  `json:"url"`
	BadgeURL string  `json:"badgeUrl"`
}

// StatusPageInput serves the create (all three fields) and the update
// (`public` and `slug`). A pointer to false is sent: unpublishing is a value.
type StatusPageInput struct {
	SpaceID *string `json:"spaceId,omitempty"`
	Slug    *string `json:"slug,omitempty"`
	Public  *bool   `json:"public,omitempty"`
}

// CreateStatusPageEntry creates the page if it does not exist, and returns it
// as it is otherwise. `public` is applied in both cases.
func (c *Client) CreateStatusPageEntry(ctx context.Context, in StatusPageInput, idempotencyKey string) (*StatusPageEntry, error) {
	out := &StatusPageEntry{}
	err := c.do(ctx, request{
		method:         http.MethodPost,
		path:           "/v1/status-pages",
		body:           in,
		idempotencyKey: idempotencyKey,
		out:            out,
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetStatusPageEntry(ctx context.Context, id string) (*StatusPageEntry, error) {
	out := &StatusPageEntry{}
	err := c.do(ctx, request{method: http.MethodGet, path: "/v1/status-pages/" + url.PathEscape(id), out: out})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateStatusPageEntry publishes, unpublishes, or moves the page of the
// organization to another address. The API refuses a body with neither field.
func (c *Client) UpdateStatusPageEntry(ctx context.Context, id string, in StatusPageInput) (*StatusPageEntry, error) {
	out := &StatusPageEntry{}
	err := c.do(ctx, request{
		method: http.MethodPatch,
		path:   "/v1/status-pages/" + url.PathEscape(id),
		body:   in,
		out:    out,
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Account
// ---------------------------------------------------------------------------

// Quota is one counter of the plan. A nil Limit means the plan sets no quota,
// and Remaining is then nil too.
type Quota struct {
	Used      *Number `json:"used"`
	Limit     *Number `json:"limit"`
	Remaining *Number `json:"remaining"`
	// Period is set on the counters that reset: `month` or `day`.
	Period *string `json:"period"`
	// PackRunsRemaining is only set on the runs counter.
	PackRunsRemaining *Number `json:"packRunsRemaining"`
}

// NamedRef is an identifier with its display name.
type NamedRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AccountPlan is the active plan, as the API names and prices it.
type AccountPlan struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	PriceMonthlyEur *Number `json:"priceMonthlyEur"`
}

// AccountQuotas groups the counters. Spaces is nil for a key limited to a
// Space, and while Spaces are not open for the organization.
type AccountQuotas struct {
	Ping            Quota  `json:"ping"`
	Chain           Quota  `json:"chain"`
	Flow            Quota  `json:"flow"`
	Runs            Quota  `json:"runs"`
	AssistantTrials Quota  `json:"assistantTrials"`
	Spaces          *Quota `json:"spaces"`
}

// Account is what `/v1/account` returns. For a key limited to a Space, Space
// is set, and `used` only counts that Space while `limit` and `remaining` stay
// those of the organization, whose quotas the Space shares.
type Account struct {
	Organization NamedRef      `json:"organization"`
	Space        *NamedRef     `json:"space"`
	Plan         AccountPlan   `json:"plan"`
	Quotas       AccountQuotas `json:"quotas"`
	BillingURL   string        `json:"billingUrl"`
}

func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	out := &Account{}
	if err := c.do(ctx, request{method: http.MethodGet, path: "/v1/account", out: out}); err != nil {
		return nil, err
	}
	return out, nil
}
