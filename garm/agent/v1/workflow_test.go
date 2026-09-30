package agentv1_test

import (
	"testing"

	agentv1 "github.com/garm-ai/contracts/garm/agent/v1"
	"google.golang.org/protobuf/proto"
)

// The contract must carry a graph, and a round trip must preserve it: a field
// added to an option is easy to declare and easy to get wrong in a way only a
// marshal catches.
func TestAWorkflowPolicyRoundTrips(t *testing.T) {
	in := &agentv1.AgentPolicy{
		Mode:    agentv1.Mode_MODE_WORKFLOW,
		Initial: map[string]string{"customer_id": "input.customer_id"},
		Steps: []*agentv1.Step{{
			Id:   "customer",
			Tool: "accounts.v1.get_customer",
			With: map[string]string{"customer_id": "state.customer_id"},
			Set:  map[string]string{"display": "response.display_name"},
		}},
		Edges: []*agentv1.Edge{
			{From: "customer", To: "screen", When: "state.flagged"},
			{From: "customer", To: "done"},
		},
	}
	raw, err := proto.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out agentv1.AgentPolicy
	if err := proto.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.GetMode() != agentv1.Mode_MODE_WORKFLOW {
		t.Errorf("mode = %v, want MODE_WORKFLOW", out.GetMode())
	}
	if got := out.GetInitial()["customer_id"]; got != "input.customer_id" {
		t.Errorf("initial[customer_id] = %q", got)
	}
	if n := len(out.GetSteps()); n != 1 {
		t.Fatalf("steps = %d, want 1", n)
	}
	if got := out.GetSteps()[0].GetSet()["display"]; got != "response.display_name" {
		t.Errorf("set[display] = %q", got)
	}
	// The unconditional edge is how A6's totality is expressed, so an empty
	// `when` must survive rather than being indistinguishable from absent.
	if w := out.GetEdges()[1].GetWhen(); w != "" {
		t.Errorf("second edge when = %q, want empty", w)
	}
}

// MODE_REACT must keep its number. A renumbering would silently repoint every
// agent already published.
func TestModeNumbersAreStable(t *testing.T) {
	if agentv1.Mode_MODE_REACT != 1 {
		t.Errorf("MODE_REACT = %d, want 1", agentv1.Mode_MODE_REACT)
	}
	if agentv1.Mode_MODE_WORKFLOW != 2 {
		t.Errorf("MODE_WORKFLOW = %d, want 2", agentv1.Mode_MODE_WORKFLOW)
	}
}

// `derives` must be a FIELD option in this package's own block.
func TestDerivesIsAFieldOptionAt50102(t *testing.T) {
	if got := agentv1.E_Derives.TypeDescriptor().Number(); got != 50102 {
		t.Errorf("derives number = %d, want 50102", got)
	}
}
