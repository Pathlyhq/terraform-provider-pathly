package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

/*
Schema invariants.

The defect that motivated this file showed up neither at build time, nor in a
unit test, nor on the first `terraform plan`: `type` was optional and computed,
forced replacement, and had no modifier to keep the known value. As long as
nothing changed, the plan stayed empty. On the first interval change, Terraform
declared `type` unknown, the replacement triggered, and the scenario was
destroyed then recreated — losing its run history, its incidents and its
availability measurements.

These tests therefore check the shape of the schema, not a behavior: this is the
only place where the faulty combination is visible before a customer suffers it.
*/

// attributeInfo gathers what can be queried without knowing the type.
type attributeInfo struct {
	optional    bool
	computed    bool
	modifiers   []string
	description string
}

// inspect returns the modifiers as type names, the only common denominator
// between the five typed variants of planmodifier.
func inspect(a fwschema.Attribute) attributeInfo {
	info := attributeInfo{
		optional:    a.IsOptional(),
		computed:    a.IsComputed(),
		description: a.GetDescription(),
	}
	switch typed := a.(type) {
	case fwschema.StringAttribute:
		info.modifiers = names(typed.PlanModifiers)
	case fwschema.BoolAttribute:
		info.modifiers = names(typed.PlanModifiers)
	case fwschema.Int64Attribute:
		info.modifiers = names(typed.PlanModifiers)
	case fwschema.Float64Attribute:
		info.modifiers = names(typed.PlanModifiers)
	case fwschema.ListAttribute:
		info.modifiers = names(typed.PlanModifiers)
	}
	return info
}

func names[T any](modifiers []T) []string {
	out := make([]string, 0, len(modifiers))
	for _, m := range modifiers {
		var description string
		switch typed := any(m).(type) {
		case planmodifier.String:
			description = typed.Description(context.Background())
		case planmodifier.Bool:
			description = typed.Description(context.Background())
		case planmodifier.Int64:
			description = typed.Description(context.Background())
		case planmodifier.Float64:
			description = typed.Description(context.Background())
		case planmodifier.List:
			description = typed.Description(context.Background())
		}
		out = append(out, description)
	}
	return out
}

func contains(values []string, pattern string) bool {
	for _, v := range values {
		if strings.Contains(v, pattern) {
			return true
		}
	}
	return false
}

func schemas(t *testing.T) map[string]fwschema.Schema {
	t.Helper()
	ctx := context.Background()
	out := map[string]fwschema.Schema{}
	for name, factory := range map[string]func() resource.Resource{
		"pathly_scenario":           NewScenarioResource,
		"pathly_maintenance_window": NewMaintenanceWindowResource,
		"pathly_webhook":            NewWebhookResource,
		"pathly_sla_target":         NewSlaTargetResource,
	} {
		resp := &resource.SchemaResponse{}
		factory().Schema(ctx, resource.SchemaRequest{}, resp)
		out[name] = resp.Schema
	}
	return out
}

// The deadly combination: optional, computed, and replacement forced, without
// keeping the known value.
func TestNoComputedAttributeTriggersAPhantomReplacement(t *testing.T) {
	for resourceName, s := range schemas(t) {
		for name, attribute := range s.Attributes {
			info := inspect(attribute)
			if !info.computed {
				continue
			}
			if !contains(info.modifiers, "Once set, the value of this attribute in state will not change") &&
				contains(info.modifiers, "Requires replacement") {
				t.Errorf(
					"%s.%s: computed and forcing replacement without UseStateForUnknown. "+
						"On the first change of another attribute, the resource would be destroyed then recreated.",
					resourceName, name,
				)
			}
		}
	}
}

// A computed attribute the API does not make evolve must keep its value,
// otherwise every plan displays imaginary moves and the user stops reading
// plans.
func TestStableComputedAttributesKeepTheirValue(t *testing.T) {
	// Only legitimate exceptions: what the server changes on its own.
	volatile := map[string]bool{
		"pathly_scenario.last_status": true,
		"pathly_scenario.muted_until": true,
	}

	for resourceName, s := range schemas(t) {
		for name, attribute := range s.Attributes {
			info := inspect(attribute)
			if !info.computed || volatile[resourceName+"."+name] {
				continue
			}
			if !contains(info.modifiers, "Once set, the value of this attribute in state will not change") {
				t.Errorf(
					"%s.%s: computed without UseStateForUnknown, while its value does not move on the server side",
					resourceName, name,
				)
			}
		}
	}
}

// An attribute without a description appears nowhere in the registry
// documentation, and the user has to read the Go code to understand what they
// are setting.
func TestEveryAttributeIsDescribed(t *testing.T) {
	for resourceName, s := range schemas(t) {
		for name, attribute := range s.Attributes {
			if strings.TrimSpace(inspect(attribute).description) == "" {
				t.Errorf("%s.%s: without a description", resourceName, name)
			}
		}
	}
}
