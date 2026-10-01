package tasksv1_test

import (
	"slices"
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
//
// IT IS EMPTY, and that is the finished state rather than a map waiting to be
// filled. `CreateTask` was the last entry, held open on the reading that a set
// on it would force every agent's manifest to carry that set — a change to
// every agent's ceiling. That reading was wrong about who calls it: a runner
// opens a task as its OWN service principal, not as a delegated agent token, so
// the set sits on that one principal's claims and no manifest is touched. It now
// declares `escalation`, a set holding create_task and nothing else, so the
// runner's reach is stated rather than inferred from clearance arithmetic. See
// KNOWN-GAPS.md.
var unscopedWithAReason = map[string]string{}

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

// `escalation` holds create_task and get_task_grant, pinned by name.
//
// This is the least-privilege half of the ruling and the half that would rot
// quietly: the principal that holds this set is the whole platform's runner, so
// a method added to the set later is reach handed to every run it executes. A
// caller granted `escalation` can open an escalation and collect its answer, and
// can do nothing else — it cannot list the queue, read a task back, claim,
// decide or triage.
//
// The pair is deliberate rather than a widening. Opening a task and collecting
// the grant its decision minted are two halves of one act, which is what naming
// a set for the ACT rather than for the caller buys: the membership grew by the
// method that completes the act, and no claims policy changed.
//
// `tasksd` pins the same membership in `internal/tasks/contract_test.go`, on its
// own copy of this package. Two modules hold `garm.tasks.v1` during the
// switchover and the copies must not diverge, so each one guards it.
func TestEscalationHoldsOpeningAnEscalationAndCollectingItsAnswer(t *testing.T) {
	svc := tasksv1.File_garm_tasks_v1_tasks_proto.Services().Get(0)

	var held []string
	for i := 0; i < svc.Methods().Len(); i++ {
		pol, _ := proto.GetExtension(svc.Methods().Get(i).Options(), toolv1.E_Tool).(*toolv1.ToolPolicy)
		for _, s := range pol.GetSets() {
			if s == "escalation" {
				held = append(held, pol.GetName())
			}
		}
	}
	slices.Sort(held)
	if want := []string{"create_task", "get_task_grant"}; !slices.Equal(held, want) {
		t.Errorf("`escalation` holds %v, want %v — holding this set must let a "+
			"runner park a run on a decision and pick that decision up, and "+
			"nothing else", held, want)
	}

	// Declared as well as used: a tool naming a set the file never declared
	// names something that matches no caller, which costs it every scoped
	// caller and says nothing.
	decl, _ := proto.GetExtension(svc.ParentFile().Options(), toolv1.E_ToolSets).(*toolv1.DeclSet)
	for _, want := range []string{"triage", "escalation"} {
		found := false
		for _, d := range decl.GetDeclared() {
			if d.GetName() == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the file declares no tool set named %q", want)
		}
	}
}

// get_task_grant is declared on the four axes garmd gates on, each pinned with
// the reason it is that value and not a wider one.
//
// The predicate is an AND over all four — `p.Verbs.Has(t.Verb) &&
// policy.Allows(p.Clearance, t.MinClearance) && p.Compartments.Covers(need) &&
// inScope(p.ToolSets, t.Sets)` — with no implication between verbs: a principal
// holding WRITE does not thereby hold READ, and a principal holding READ reaches
// every READ tool its clearance, compartments and SETS admit. That last clause is
// the whole of why this method may honestly declare READ. The runner principal is
// `clearance: PUBLIC`, no compartments, `tool_sets: [escalation]`, and it is
// granted READ for this method in `sts/deploy/claims.yaml` and
// `examples/bank/auth/claims.yaml`; `escalation` holds create_task and this
// method and nothing else, which the test above pins by name, so the verb reaches
// one more tool and not a class of them.
//
// An earlier revision declared VERB_WRITE here, to avoid touching a policy at
// all. That bought a method whose verb described the wrong act and contradicted
// its own `effects.idempotent`, and it set the precedent that a policy too narrow
// is fixed by relabelling the tool. The verb vocabulary is garm's own invention
// with no upstream to appeal to, so it is worth exactly what each declaration
// keeps it worth.
//
// It pins the audience and the audit for the same reason: AUDIENCE_RUNNER is why
// no person's client and no model ever lists a credential read, and
// record_response: false is why the bearer this method returns reaches no ledger
// row by value.
func TestTheGrantReadBackIsDeclaredWhereTheRunnerAlreadyReaches(t *testing.T) {
	svc := tasksv1.File_garm_tasks_v1_tasks_proto.Services().Get(0)
	m := svc.Methods().ByName("GetTaskGrant")
	if m == nil {
		t.Fatal("TasksService declares no GetTaskGrant; the decided event carries a " +
			"reference and never the grant, so a woken run has no way to read the " +
			"approval it was parked on")
	}
	pol, _ := proto.GetExtension(m.Options(), toolv1.E_Tool).(*toolv1.ToolPolicy)
	if pol == nil {
		t.Fatal("GetTaskGrant carries no tool annotation at all")
	}

	if pol.GetName() != "get_task_grant" {
		t.Errorf("name = %q, want get_task_grant", pol.GetName())
	}
	if pol.GetVerb() != toolv1.Verb_VERB_READ {
		t.Errorf("verb = %s, want VERB_READ. The call returns a value and moves "+
			"nothing, which is also what effects.idempotent says, and a verb is a "+
			"description of the act before it is a policy axis — this vocabulary is "+
			"garm's own, so it is worth what each declaration keeps it worth. The "+
			"runner is granted READ for this; the SET is what bounds that, not the "+
			"verb", pol.GetVerb())
	}
	if pol.GetMinClearance() != toolv1.Clearance_CLEARANCE_PUBLIC {
		t.Errorf("min_clearance = %s, want CLEARANCE_PUBLIC; the runner's principal "+
			"is PUBLIC", pol.GetMinClearance())
	}
	if len(pol.GetCompartments()) != 0 {
		t.Errorf("compartments = %v, want none; the runner's principal holds none",
			pol.GetCompartments())
	}
	if want := []string{"escalation"}; !slices.Equal(pol.GetSets(), want) {
		t.Errorf("sets = %v, want exactly %v. This is the bound on the runner's READ "+
			"rather than a label: that principal is scoped to `tool_sets: "+
			"[escalation]`, so a second set here would widen what one granted verb "+
			"reaches", pol.GetSets(), want)
	}
	if want := []toolv1.Audience{toolv1.Audience_AUDIENCE_RUNNER}; !slices.Equal(pol.GetAudience(), want) {
		t.Errorf("audience = %v, want %v; a person never calls this and no model is "+
			"ever offered it", pol.GetAudience(), want)
	}
	if pol.GetAudit().GetRecordResponse() {
		t.Error("audit.record_response is true: the response carries the bearer, so a " +
			"ledger row would hold the credential by value")
	}
	if pol.GetAudit().GetRecordRequest() {
		t.Error("audit.record_request is true: the request is where a single-task " +
			"capability lands, and a ledger row holding one would be the same leak")
	}
}

// No message a method ANSWERS WITH carries a grant, except the one message whose
// whole purpose is to.
//
// This is the ruling "not a field on `Task`" made mechanical. `Task` is read by
// Studio and by people, and a bearer in a widely-read projection is the thing
// the decided event's reference exists to prevent — gating it with a field
// policy would still put the credential in that message and make its omission
// depend on every reader evaluating policy correctly. So the check is on
// outputs, recursively, and `TaskGrant` is the single exemption: one method, one
// credential, reachable through one set.
//
// Requests are deliberately not walked. `DecideTaskRequest.grant` is the
// approval travelling INTO this service from the deciding person's client, which
// is the direction that has to work.
func TestNoProjectionAnswersWithAGrant(t *testing.T) {
	svc := tasksv1.File_garm_tasks_v1_tasks_proto.Services().Get(0)
	const allowed = "garm.tasks.v1.TaskGrant"

	seen := map[protoreflect.FullName]bool{}
	var walk func(md protoreflect.MessageDescriptor)
	walk = func(md protoreflect.MessageDescriptor) {
		if seen[md.FullName()] {
			return
		}
		seen[md.FullName()] = true
		for i := 0; i < md.Fields().Len(); i++ {
			f := md.Fields().Get(i)
			if f.Name() == "grant" && string(md.FullName()) != allowed {
				t.Errorf("%s.%s: a projection carries a bearer. The grant is read back "+
					"through get_task_grant, whose answer is %s and which nothing but "+
					"the runner's set reaches; a field here is read by Studio and by "+
					"people.", md.FullName(), f.Name(), allowed)
			}
			if f.Kind() == protoreflect.MessageKind || f.Kind() == protoreflect.GroupKind {
				walk(f.Message())
			}
		}
	}
	for i := 0; i < svc.Methods().Len(); i++ {
		walk(svc.Methods().Get(i).Output())
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
