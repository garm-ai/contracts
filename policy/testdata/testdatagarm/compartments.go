// Package testdatagarm is a compatibility shim, and the only thing left in it
// is scheduled to go.
//
// It used to hand-write the OUTPUT of a code generator: thirty-two lines
// hardcoding the three compartment names that
// [github.com/garm-ai/contracts/policy/testdata]'s fixture.proto declares, with
// a doc comment promising they stayed byte-identical to what
// cmd/protoc-gen-garm-go emits. That promise could not be checked from this
// repository, because the plugin is in the command line tool's, and a promise
// nobody can check is not one.
//
// The declaration is a file-level option on the fixture and reading it is what
// [github.com/garm-ai/contracts/policy] is for, so
// [policy.DeclaredCompartments] reads it. **New code calls that.** This
// variable is derived from the same call, so the two cannot disagree and there
// is nothing left to verify.
//
// It survives only because garmd's toolplane and grants tests import it, and
// they cannot call [policy.DeclaredCompartments] until garmd moves off
// github.com/garm-ai/garm and onto this module. When it does, those four test
// files change one line each and this package is deleted.
package testdatagarm

import (
	toolv1 "github.com/garm-ai/contracts/garm/tool/v1"
	"github.com/garm-ai/contracts/policy"
	"github.com/garm-ai/contracts/policy/testdata"
)

// Compartments is the taxonomy declared by testdata/fixture.proto.
//
// Deprecated: call [policy.DeclaredCompartments] with the fixture's file
// descriptor instead. It is the same call this variable makes.
var Compartments = policy.DeclaredCompartments(
	(&testdata.Profile{}).ProtoReflect().Descriptor().ParentFile())

// Compile-time proof that the type has not moved under the one consumer left.
var _ []*toolv1.Decl = Compartments
