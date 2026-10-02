package toolv1_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	toolv1 "github.com/garm-ai/contracts/garm/tool/v1"
)

// InvocationContext is outside descriptorHash (garm/tasks/v1/wireshape_test.go):
// it is a field of no message, so TasksService's walk never reaches it. That
// left the one object crossing the hop between a garmd build and a tool build
// with nothing pinning its shape -- a drift neither side would notice, because
// a field the receiver does not know about simply does not arrive.
//
// Anchored on the message itself rather than on a service, because it belongs
// to no service. The same field/number/cardinality/kind walk as
// descriptorHash, and NEVER an option -- deliberately the same idea of "a
// message's shape", so this repository has one of those rather than two.
const envelopeShapeValue = "5a84b346fb4ee877c0cfc79b0a0b1672cfdba4eeef0c5c473da9d16fd2866b55"

func TestTheEnvelopeShapeIsPinned(t *testing.T) {
	if got := envelopeShape(t); got != envelopeShapeValue {
		t.Errorf("InvocationContext's shape moved:\n got  %s\n want %s\n\n"+
			"Every garmd and every tool build must agree about this message. If "+
			"the change is intended, update the constant AND say in the commit "+
			"which field moved it -- a tool built against the old shape will not "+
			"see the new field, and nothing else will tell you.", got, envelopeShapeValue)
	}
}

// envelopeShape walks InvocationContext and every message it reaches
// (CallContext, InvocationPrincipal, Act, and whatever well-known types they
// carry) and hashes field numbers, names, cardinalities and kinds -- never an
// option. This is garm/tasks/v1/wireshape_test.go's descriptorHash walk,
// reused rather than reinvented, starting from the message itself instead of
// from a service's methods.
func envelopeShape(t *testing.T) string {
	t.Helper()

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

	walk((&toolv1.InvocationContext{}).ProtoReflect().Descriptor())
	return hex.EncodeToString(h.Sum(nil))
}
