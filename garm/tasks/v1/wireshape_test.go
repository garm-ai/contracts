package tasksv1_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	tasksv1 "github.com/garm-ai/contracts/garm/tasks/v1"
)

// wireShape is the value this package's copy of garm.tasks.v1 advertises,
// written down.
//
// It is here because the package exists in TWO modules during the switchover —
// this one and `github.com/garm-ai/tasksd`, which serves it — and a deployment
// stays out of quarantine for exactly one reason: both copies hash to the same
// number. `garm catalogue build` stamps this value into a catalogue from
// whichever copy the command line tool links, and garmd refuses to route to a
// service whose own value differs. So the two copies agreeing is not a tidiness
// claim, it is what keeps the queue reachable, and `tasksd` pins the identical
// constant in `internal/tasks/contract_test.go` on its own copy.
//
// Until this test existed only tasksd guarded it, which left the risk one-sided:
// an edit HERE could move the shape and nothing in this repository would say so
// until somebody ran another repository's suite. Each copy now guards itself.
//
// IT MOVED ON 2026-10-01, and twice in the one round. That is a real answer rather
// than a broken test, and the two changes were batched BECAUSE each move costs a
// catalogue rebuild and a command line tool release:
//
//  1. `TasksService` gained `GetTaskGrant`, whose request and answer —
//     `GetTaskGrantRequest` and `TaskGrant` — are both new messages, so the walk
//     below covers two messages and four fields it did not before. That took the
//     value to `b80da142…3df2`.
//  2. `CreateTaskRequest` gained `run_id` (field 11), because the run has to come
//     from the request: garmd never puts one on an invocation and must not learn
//     how, knowing nothing about runs being one of its invariants. One field, and
//     the value below.
//
// What did NOT move it, in the same round, was `get_task_grant` going from
// VERB_WRITE to VERB_READ: a verb is an option, and this walk emits none.
// Confirmed by running this test either side of that change rather than assumed.
//
// Every catalogue carrying garm.tasks.v1 has to be rebuilt by a command line tool
// linking this version, and until it is, garmd quarantines tasksd on the mismatch.
// If a change moves it again: update this constant, update tasksd's to the same
// value, and expect the rebuild.
const wireShape = "8939ac00b2a441346826759b77d72dc568a9bb0fa32d50732cc72b3c166b5a63"

// The wire shape is what a daemon compares with the catalogue before it routes
// anything to the service answering this contract.
func TestTheWireShapeIsTheValueTasksdAlsoPins(t *testing.T) {
	got := descriptorHash(t)
	if len(got) != 64 {
		t.Fatalf("the wire shape is %q, want 64 hex characters", got)
	}
	if got != wireShape {
		t.Errorf("the wire shape of this copy is %q, want %q.\n"+
			"Read the constant's comment before changing it: tasksd pins the same "+
			"value on its own copy of garm.tasks.v1, and a deployment stays out of "+
			"quarantine only while the two agree.", got, wireShape)
	}
}

// descriptorHash is the encoding `garm catalogue build` stamps into a catalogue
// (garm's internal/compiler, emit_micro.go, func descriptorHash), and the one
// `tasksd`'s internal/tasks computes at startup to advertise.
//
// Written out here rather than imported, because the producer's copy is
// unexported in another repository's internal/ package and tasksd's is in its
// own internal/ too. That is a known duplication with a stated home: a hash a
// producer and a consumer both compute belongs in THIS module, exported, and
// these three copies collapse into one import when it lands. Until then the
// golden constant above is what keeps them from drifting apart unnoticed, and a
// drift that did get through is loud rather than silent — garmd refuses to route
// to a service whose shape it cannot confirm, and says so.
//
// It reads field numbers, names, cardinalities and kinds, and never an option.
// That is why `go_package` and the tool annotations differ between the two
// copies of this package without moving the value.
func descriptorHash(t *testing.T) string {
	t.Helper()
	fd := tasksv1.File_garm_tasks_v1_tasks_proto
	sd := fd.Services().ByName("TasksService")
	if sd == nil {
		t.Fatalf("%s declares no TasksService", fd.Path())
	}

	h := sha256.New()
	visited := map[protoreflect.FullName]bool{}

	var walk func(md protoreflect.MessageDescriptor)
	walk = func(md protoreflect.MessageDescriptor) {
		if visited[md.FullName()] {
			return
		}
		visited[md.FullName()] = true

		fields := md.Fields()
		ordered := make([]protoreflect.FieldDescriptor, fields.Len())
		for i := range ordered {
			ordered[i] = fields.Get(i)
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].Number() < ordered[j].Number() })

		fmt.Fprintf(h, "message %s\n", md.FullName())
		for _, f := range ordered {
			fmt.Fprintf(h, "field %d %s %s %s\n", f.Number(), f.Name(), f.Cardinality(), f.Kind())
		}
		// After this message's own field list, so one message's shape is never
		// interleaved with another's.
		for _, f := range ordered {
			if f.Kind() == protoreflect.MessageKind || f.Kind() == protoreflect.GroupKind {
				walk(f.Message())
			}
		}
	}

	for i := 0; i < sd.Methods().Len(); i++ {
		walk(sd.Methods().Get(i).Input())
		walk(sd.Methods().Get(i).Output())
	}
	return hex.EncodeToString(h.Sum(nil))
}
