package policy_test

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	toolv1 "github.com/garm-ai/contracts/garm/tool/v1"
	"github.com/garm-ai/contracts/policy"
	"github.com/garm-ai/contracts/policy/testdata"
	"github.com/garm-ai/contracts/policy/testdata/testdatagarm"
)

// names is for diagnostics: a failure should print the compartments it found,
// not a slice of pointers.
func names(ds []*toolv1.Decl) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.GetName())
	}
	return out
}

func fixtureFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	return (&testdata.Profile{}).ProtoReflect().Descriptor().ParentFile()
}

// The fixture declares three compartments at file scope. Reading them is what
// replaced a hand-written file that hardcoded the same three names and claimed
// to be byte-identical to the output of a generator in another repository.
//
// The names are written out here on purpose. This is the one place the
// taxonomy is asserted rather than derived, so a change to fixture.proto that
// drops or renames a compartment fails here by name instead of quietly
// changing what every other test in this package is planning against.
func TestDeclaredCompartmentsReadsTheFileScopeDeclaration(t *testing.T) {
	got := policy.DeclaredCompartments(fixtureFile(t))

	want := []string{"financial", "pii-contact", "support"}
	if len(got) != len(want) {
		t.Fatalf("got %d compartments, want %d: %v", len(got), len(want), names(got))
	}
	for i, w := range want {
		if got[i].GetName() != w {
			t.Errorf("compartment %d = %q, want %q (declaration order is the "+
				"registry's bit order and must not drift)", i, got[i].GetName(), w)
		}
	}
}

// Declaration order is the registry's bit order, so it is part of the
// contract rather than an implementation detail: the same taxonomy read twice
// has to assign the same bit to the same compartment.
func TestTheRegistryBuiltFromTheDeclarationIsStable(t *testing.T) {
	a, err := policy.NewRegistry(policy.DeclaredCompartments(fixtureFile(t)))
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	b, err := policy.NewRegistry(policy.DeclaredCompartments(fixtureFile(t)))
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}

	for _, name := range []string{"financial", "pii-contact", "support"} {
		sa, err := a.Set([]string{name})
		if err != nil {
			t.Fatalf("Set(%q): %v", name, err)
		}
		sb, err := b.Set([]string{name})
		if err != nil {
			t.Fatalf("Set(%q): %v", name, err)
		}
		if sa != sb {
			t.Errorf("%q got different bits in two registries built from the "+
				"same declaration", name)
		}
	}
}

// A file with no declaration is not an error and not a nil-pointer walk: most
// files in any tree declare no taxonomy at all, and a caller unions across
// every file it loaded.
func TestAFileDeclaringNothingContributesNothing(t *testing.T) {
	// garm.tool.v1 itself declares no compartments — it defines the option
	// rather than using it.
	toolFile := fixtureFile(t).Imports().Get(0).FileDescriptor

	if got := policy.DeclaredCompartments(toolFile); len(got) != 0 {
		t.Errorf("got %v from a file that declares no compartments", names(got))
	}
	if got := policy.DeclaredCompartments(); got != nil {
		t.Errorf("got %v from no files at all", names(got))
	}
	if got := policy.DeclaredCompartments(nil); got != nil {
		t.Errorf("got %v from a nil file; a caller unioning over a slice it "+
			"built should not have to filter it first", names(got))
	}
}

// The union is by name across files, and the first declaration of a name wins.
// Passing the same file twice is the cheapest way to assert that without a
// second fixture, and it is also a real case: a caller unioning a file and its
// transitive imports can reach one file by two paths.
func TestTheUnionIsByNameAcrossFiles(t *testing.T) {
	fd := fixtureFile(t)
	once := policy.DeclaredCompartments(fd)
	twice := policy.DeclaredCompartments(fd, fd)

	if len(twice) != len(once) {
		t.Errorf("the same file counted twice gave %d compartments, want %d: %v",
			len(twice), len(once), names(twice))
	}
}

func TestDeclaredSetsReadsTheToolSetDeclaration(t *testing.T) {
	got := policy.DeclaredSets(fixtureFile(t))
	if len(got) == 0 {
		t.Skip("the fixture declares no tool sets; nothing to assert here")
	}
	for _, d := range got {
		if d.GetName() == "" {
			t.Error("a declared tool set has no name")
		}
	}
}

// The deprecated shim garmd still imports is derived from the same call, so
// this asserts they cannot disagree.
//
// It lives HERE rather than beside the shim because the go tool skips
// directories named testdata: a test file in there is never compiled by
// `go test ./...` and never runs in CI, which is how a package four test files
// in another repository depend on came to have no coverage of its own at all.
func TestTheDeprecatedShimAgreesWithTheReader(t *testing.T) {
	want := policy.DeclaredCompartments(fixtureFile(t))
	got := testdatagarm.Compartments

	if len(got) != len(want) {
		t.Fatalf("shim has %d compartments, reader found %d: %v vs %v",
			len(got), len(want), names(got), names(want))
	}
	for i := range want {
		if got[i].GetName() != want[i].GetName() {
			t.Errorf("[%d] shim %q, reader %q", i, got[i].GetName(), want[i].GetName())
		}
	}
}
