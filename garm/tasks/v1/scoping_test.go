package tasksv1_test

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	tasksv1 "github.com/garm-ai/contracts/garm/tasks/v1"
	toolv1 "github.com/garm-ai/contracts/garm/tool/v1"
)

// unscopedWithAReason are the methods of this contract that deliberately
// declare no tool set, each with why.
//
// An entry is a claim that the method is reachable anyway. Adding one without
// a reason is how the bug this test exists for got in.
var unscopedWithAReason = map[string]string{
	"CreateTask": "the caller is a runner, and giving it a set would mean every " +
		"agent that can park a call for approval must carry that set in its " +
		"manifest — a change to every agent's ceiling, which is not this " +
		"contract's decision to make alone. Open question; see KNOWN-GAPS.",
}

// Every method of this service must declare a tool set, and the reason is a
// rule about the daemon rather than a style preference.
//
// garmd's scope check refuses a caller any tool that shares none of its sets,
// and a tool in NO set therefore shares none with anybody who names one. So a
// method with no set is reachable only by a caller with no set at all — in the
// bank example, only the customer's role, because every staff role names one.
//
// This went wrong once already and the cause is worth pinning. decide_task and
// approval_card declared `audience: [AUDIENCE_PERSON]` and no set, written as
// though the audience were the gate. It is not: an audience decides who is
// OFFERED a tool in a listing, and the `act` refusal is what keeps a delegated
// caller from deciding. So the two methods a human approver most needs were
// unreachable by every role that had been scoped, and the symptom was a queue
// nobody could act on rather than an error anybody could see.
func TestEveryMethodDeclaresAToolSet(t *testing.T) {
	svc := tasksv1.File_garm_tasks_v1_tasks_proto.Services().Get(0)

	for i := 0; i < svc.Methods().Len(); i++ {
		m := svc.Methods().Get(i)
		name := string(m.Name())

		pol, _ := proto.GetExtension(m.Options(), toolv1.E_Tool).(*toolv1.ToolPolicy)
		if pol == nil {
			t.Errorf("%s carries no tool annotation at all", name)
			continue
		}

		if len(pol.GetSets()) > 0 {
			if reason, ok := unscopedWithAReason[name]; ok {
				t.Errorf("%s declares sets %v and is also listed as deliberately "+
					"unscoped (%q). Remove the exemption.", name, pol.GetSets(), reason)
			}
			continue
		}

		if _, ok := unscopedWithAReason[name]; ok {
			continue
		}
		t.Errorf("%s declares no tool set, so garmd refuses it to every caller "+
			"that names one — which is every role in the bank example but the "+
			"customer's. Declare a set, or add %s to unscopedWithAReason with "+
			"the reason it is reachable without one.", name, name)
	}
}

// The pair that must not drift apart: an approver who may decide and may not
// read the card they decide from has been granted nothing usable.
func TestDecidingAndReadingItsCardShareASet(t *testing.T) {
	svc := tasksv1.File_garm_tasks_v1_tasks_proto.Services().Get(0)

	setsOf := func(name protoreflect.Name) []string {
		m := svc.Methods().ByName(name)
		if m == nil {
			t.Fatalf("no method %s", name)
		}
		pol, _ := proto.GetExtension(m.Options(), toolv1.E_Tool).(*toolv1.ToolPolicy)
		return pol.GetSets()
	}

	decide, card := setsOf("DecideTask"), setsOf("ApprovalCard")
	if len(decide) == 0 || len(card) == 0 {
		t.Fatalf("DecideTask sets %v, ApprovalCard sets %v; both must be scoped", decide, card)
	}
	shared := false
	for _, d := range decide {
		for _, c := range card {
			if d == c {
				shared = true
			}
		}
	}
	if !shared {
		t.Errorf("DecideTask %v and ApprovalCard %v share no set, so a role can "+
			"reach one without the other", decide, card)
	}
}

// Claiming and deciding are two halves of one act — the decision requires the
// claim — so a caller granted one and not the other reaches a queue and can do
// nothing with it.
func TestClaimingAndDecidingShareASet(t *testing.T) {
	svc := tasksv1.File_garm_tasks_v1_tasks_proto.Services().Get(0)
	get := func(n protoreflect.Name) []string {
		pol, _ := proto.GetExtension(svc.Methods().ByName(n).Options(), toolv1.E_Tool).(*toolv1.ToolPolicy)
		return pol.GetSets()
	}
	claim, decide := get("ClaimTask"), get("DecideTask")
	shared := false
	for _, a := range claim {
		for _, b := range decide {
			if a == b {
				shared = true
			}
		}
	}
	if !shared {
		t.Errorf("ClaimTask %v and DecideTask %v share no set", claim, decide)
	}
}
