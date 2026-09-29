package policy

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	toolv1 "github.com/garm-ai/contracts/garm/tool/v1"
)

// DeclaredCompartments unions the file-level compartment declarations across a
// set of files.
//
// A compartment is declared once, at file scope, and referred to by name
// everywhere else; this is what turns those declarations into the list
// [NewRegistry] wants. The union is by name and keeps the first declaration of
// each, so two files declaring the same compartment is not an error — the
// linter is what decides whether it should be.
//
// # Why this is here
//
// It was in the command line tool's internal packages, and being internal to
// the tool meant nobody else could call it. What everybody else did instead
// was worse in two different ways: the daemon duplicated the walk, and this
// module's own test fixture hand-wrote the OUTPUT of the tool's code
// generator — thirty-two lines hardcoding three strings, carrying a promise to
// stay byte-identical to a generator in another repository, with no way to
// check it from here.
//
// Reading a declaration is what this package is for. So it reads it.
func DeclaredCompartments(fds ...protoreflect.FileDescriptor) []*toolv1.Decl {
	return declaredAtFileScope(fds, toolv1.E_Compartments)
}

// DeclaredSets unions the file-level tool-set declarations across a set of
// files. The counterpart to [DeclaredCompartments], and the same rules.
func DeclaredSets(fds ...protoreflect.FileDescriptor) []*toolv1.Decl {
	return declaredAtFileScope(fds, toolv1.E_ToolSets)
}

// declaredAtFileScope walks the files in the order given and returns the first
// declaration of each name.
//
// Order is the caller's, not sorted: a registry assigns bit positions in the
// order it is handed, so sorting here would silently renumber a compartment
// set and make one build's bitset mean something else in the next. If a caller
// wants a stable order across differently ordered inputs, it sorts before
// calling and owns that decision.
func declaredAtFileScope(fds []protoreflect.FileDescriptor, xt protoreflect.ExtensionType) []*toolv1.Decl {
	seen := map[string]bool{}
	var out []*toolv1.Decl
	for _, fd := range fds {
		if fd == nil {
			continue
		}
		opts, ok := fd.Options().(*descriptorpb.FileOptions)
		if !ok || !proto.HasExtension(opts, xt) {
			continue
		}
		ds, _ := proto.GetExtension(opts, xt).(*toolv1.DeclSet)
		for _, d := range ds.GetDeclared() {
			if d.GetName() == "" || seen[d.GetName()] {
				continue
			}
			seen[d.GetName()] = true
			out = append(out, d)
		}
	}
	return out
}
