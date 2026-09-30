package contracts_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"google.golang.org/protobuf/proto"

	cataloguev1 "github.com/garm-ai/contracts/garm/catalogue/v1"
)

// provenanceBeforeInputs is the `provenance` submessage of a real catalogue,
// lifted byte for byte out of an artefact the current builder wrote: garm
// v0.19.0, for the bank example, before Provenance.inputs existed.
//
// It carries fields 1, 2 and 4 and nothing else — note that `built_at` is
// field 3 and the builder does not set it, which is why reading the wire
// rather than the .proto makes 3 look free when it is not.
//
// A hand-written literal rather than a generated fixture on purpose. What
// needs asserting is that bytes ALREADY PUBLISHED still mean what they meant,
// and bytes this module produces from its own new descriptor cannot answer
// that question: they would agree with themselves whatever the change was.
const provenanceBeforeInputs = "0a0c6761726d2f76302e31392e30" +
	"121470726f746f636f6d70696c652f76302e31342e31" +
	"22156761726d2d61692f6578616d706c65732f62616e6b"

// Every catalogue in existence was built before `inputs` existed, and garmd
// must keep loading them. A new field is additive on the wire by protobuf's
// rules, but "by the rules" is not the same as checked, and the cost of being
// wrong is a daemon that will not boot against an artefact a bank already
// published.
//
// The round trip is the load-bearing half. If field 5 had collided with
// anything those bytes carry, the collision would show up as a field parsed
// into the wrong place or as unknown-field bytes re-emitted in a different
// position — so re-marshalling to the same bytes says both that nothing was
// reinterpreted and that nothing was lost.
func TestProvenanceWrittenBeforeInputsStillReads(t *testing.T) {
	raw, err := hex.DecodeString(provenanceBeforeInputs)
	if err != nil {
		t.Fatalf("decoding the fixture: %v", err)
	}

	var p cataloguev1.Provenance
	if err := proto.Unmarshal(raw, &p); err != nil {
		t.Fatalf("a catalogue the current builder wrote no longer unmarshals: %v", err)
	}

	if got, want := p.GetProducer(), "garm/v0.19.0"; got != want {
		t.Errorf("producer = %q, want %q", got, want)
	}
	if got, want := p.GetCompiler(), "protocompile/v0.14.1"; got != want {
		t.Errorf("compiler = %q, want %q", got, want)
	}
	if got, want := p.GetSource(), "garm-ai/examples/bank"; got != want {
		t.Errorf("source = %q, want %q", got, want)
	}

	// Empty means "this builder did not say", never "composed from nothing":
	// the contract's comment on the field carries the reasoning, and this is
	// the case it is about.
	if n := len(p.GetInputs()); n != 0 {
		t.Errorf("inputs = %d entries, want 0: a catalogue built before the field existed "+
			"cannot claim to know what it was composed from", n)
	}

	out, err := proto.Marshal(&p)
	if err != nil {
		t.Fatalf("re-marshalling: %v", err)
	}
	if !bytes.Equal(raw, out) {
		t.Errorf("re-marshalled to different bytes\n got %x\nwant %x\n"+
			"Something in the new descriptor reinterprets or drops what an "+
			"already-published catalogue carries.", out, raw)
	}
}

// The two kinds of input are mutually exclusive by construction, which is the
// reason `inputs` is a repeated oneof rather than one message with fields that
// are sometimes empty. This asserts the property a consumer relies on: an
// entry is a pinned module or a local tree, never both and never neither, so
// "was this composed from something reproducible" is a switch and not an
// inference from which fields happen to be set.
func TestAnInputIsAPinnedModuleOrALocalTreeAndNeverBoth(t *testing.T) {
	p := &cataloguev1.Provenance{
		Inputs: []*cataloguev1.Input{
			{
				Of:            &cataloguev1.Input_Local{Local: &cataloguev1.LocalInput{Path: "proto", Source: "example.com/acme@abc123"}},
				ProtoPackages: []string{"acme.accounts.v1"},
			},
			{
				Of:            &cataloguev1.Input_Module{Module: &cataloguev1.ModuleInput{Path: "example.com/tools/web", Version: "v0.3.0"}},
				ProtoPackages: []string{"web.v1"},
			},
		},
	}

	raw, err := proto.Marshal(p)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	var got cataloguev1.Provenance
	if err := proto.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshalling: %v", err)
	}
	if n := len(got.GetInputs()); n != 2 {
		t.Fatalf("inputs = %d entries, want 2", n)
	}

	for i, in := range got.GetInputs() {
		switch of := in.GetOf().(type) {
		case *cataloguev1.Input_Local:
			if in.GetModule() != nil {
				t.Errorf("input %d reports both a local tree and a module", i)
			}
			if of.Local.GetPath() != "proto" || of.Local.GetSource() != "example.com/acme@abc123" {
				t.Errorf("input %d local = %v", i, of.Local)
			}
		case *cataloguev1.Input_Module:
			if in.GetLocal() != nil {
				t.Errorf("input %d reports both a module and a local tree", i)
			}
			if of.Module.GetPath() != "example.com/tools/web" || of.Module.GetVersion() != "v0.3.0" {
				t.Errorf("input %d module = %v", i, of.Module)
			}
		default:
			t.Errorf("input %d is neither kind: %T. A new arm of the oneof needs a case "+
				"here and in every consumer that decides whether an input is reproducible.", i, of)
		}
	}
}
