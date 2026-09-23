package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Conversions between the Terraform model and the client structures.
//
// Single rule, applied everywhere: a value that is null or unknown on the
// Terraform side does not go into the request. Sending a zero instead would
// erase a setting made from the console, and an `apply` must never modify what
// the configuration does not mention.

func strPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := v.ValueInt64()
	return &n
}

func boolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func float64Ptr(v types.Float64) *float64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	f := v.ValueFloat64()
	return &f
}

func stringFrom(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

func int64From(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

func boolFrom(p *bool) types.Bool {
	if p == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*p)
}

func float64From(p *float64) types.Float64 {
	if p == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*p)
}

// stringsFrom renders a Terraform list from a Go slice.
//
// An empty slice becomes an empty list and not a null one: the API returns `[]`
// for "no tag", and converting that into null would make a difference show up
// on every plan.
func stringsFrom(values []string) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}
	elements := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elements = append(elements, types.StringValue(v))
	}
	// Must is risk-free here: the type of the elements is known at build time
	// and equals that of the list.
	return types.ListValueMust(types.StringType, elements)
}

func stringsTo(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []string
	diags.Append(list.ElementsAs(ctx, &out, false)...)
	return out
}

// newIdempotencyKey produces the idempotency key of a creation.
//
// Generated once per Create call and reused by the internal retries of the
// client: if the first request went through but the response got lost, the
// retry returns the resource already created instead of creating a second one,
// that Terraform would not know about and would never delete.
func newIdempotencyKey(prefix string) string {
	buf := make([]byte, 16)
	// rand.Read never returns an error: since Go 1.24 it terminates the program
	// if entropy is missing, rather than returning weak randomness that would
	// give guessable keys from one call to the next.
	_, _ = rand.Read(buf)
	return "tf-" + prefix + "-" + hex.EncodeToString(buf)
}
