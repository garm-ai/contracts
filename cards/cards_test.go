package cards

import (
	"reflect"
	"testing"

	toolv1 "github.com/garm-ai/contracts/garm/tool/v1"
)

// F20: a synthesised card is reachable by exactly whoever can see the tool
// it describes, which means it carries the parent's sets. Before this fix
// cardPolicy declared no sets at all, which made every synthesised card
// unreachable by any scoped caller — garmd's `inScope` rule treats a tool in
// no set as reachable only by a caller in no set, and every one of the
// bank's roles is scoped. Verified against the running plane as a clean 404
// on a payments service's synthesised ApprovalCard.
//
// md may be nil here: cardPolicy only dereferences it when the parent names
// no title, and every parent below names one.
func TestCardPolicyCarriesParentSets(t *testing.T) {
	parent := &toolv1.ToolPolicy{
		Name:         "initiate_payment",
		Title:        "Initiate a payment",
		MinClearance: toolv1.Clearance_CLEARANCE_CONFIDENTIAL,
		Compartments: []string{"payments"},
		Sets:         []string{"payments"},
		Approval:     &toolv1.Approval{Mode: toolv1.Approval_MODE_GRANT},
	}

	got := cardPolicy(parent, "initiate_payment", Approval, nil)

	if !reflect.DeepEqual(got.GetSets(), []string{"payments"}) {
		t.Fatalf("card sets = %v, want [payments]", got.GetSets())
	}
	// Unaffected by this change, and checked here so a future edit that
	// breaks one does not pass by only watching the other.
	if got.GetMinClearance() != toolv1.Clearance_CLEARANCE_CONFIDENTIAL {
		t.Fatalf("card min_clearance = %v, want CONFIDENTIAL", got.GetMinClearance())
	}
	if !reflect.DeepEqual(got.GetCompartments(), []string{"payments"}) {
		t.Fatalf("card compartments = %v, want [payments]", got.GetCompartments())
	}
}

// The unscoped-tool case must keep working for an unscoped deployment: a
// parent declaring no sets at all (nil scope, in garmd's `inScope` sense)
// still synthesises a card reachable by every caller that reaches the
// parent, and gains no set of its own.
func TestCardPolicyOfAnUnscopedToolSynthesisesWithNoSets(t *testing.T) {
	parent := &toolv1.ToolPolicy{
		Name:     "create_task",
		Title:    "Open a task",
		Approval: &toolv1.Approval{Mode: toolv1.Approval_MODE_NONE},
	}

	got := cardPolicy(parent, "create_task", Input, nil)

	if len(got.GetSets()) != 0 {
		t.Fatalf("card sets = %v, want none", got.GetSets())
	}
}
