package contracts_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestContractsDependencyGraphStaysThin asserts that importing the generated
// contracts does not drag anything else in.
//
// It is a MODULE boundary again, and this time for the reason the original
// one was abandoned for. The first version of this test guarded a nested
// go.mod under a `contracts/` directory inside the CLI's repository; that was
// dropped because two version numbers had to agree and the enforcing packages
// had already moved to garmd, out of reach either way.
//
// What brought it back is the other half of the graph. A consumer importing a
// message type inherited this module's whole require list, and that list
// belonged to a command line tool: the AWS SDK for publishing a catalogue,
// cobra and pflag for the command, protocompile for reading protos. `sink`
// drains a ledger to Parquet and was pulling cobra. So the CLI kept the name
// `garm` and this module became `garm-ai/contracts`, with three requires
// instead of eleven, and no version pair to reconcile because the tool now
// depends on the contract rather than containing it.
//
// The PACKAGE property is what this checks, and it is the one that survives
// every reshuffle: whatever a tool author imports to get message types must
// not reach the plan compiler or anything heavy. `policy` lives in this
// module and is named below on purpose — it is the daemon's to import, not an
// author's, and nothing outside it may pull it in.
func TestContractsDependencyGraphStaysThin(t *testing.T) {
	forbidden := []string{
		"github.com/nats-io/nats-server",
		"github.com/marcboeker/go-duckdb",
		"github.com/minio/minio-go",
		"github.com/garm-ai/contracts/policy",
		// The CLI. It depends on this module; nothing here may depend back.
		"github.com/garm-ai/garm",
		"github.com/spf13/cobra",
		// A4 compiles guards as CEL, which pulls in an ANTLR runtime. That
		// belongs to the linter, in internal/compiler. Named here as well as
		// covered by the internal/ ban above, because the ban would not catch
		// cel-go imported straight into a contracts package.
		"cel.dev/cel-go",
		"github.com/antlr4-go/antlr",
	}
	// GOWORK=off: this test exists to guarantee what a CONSUMER of the
	// published contracts module sees, not what a developer working
	// inside this repo's go.work sees. Left to the ambient environment,
	// `go list` here resolves through go.work and go.work.sum, which
	// substitute for this module's own go.sum — masking exactly the class
	// of gap (a missing go.sum entry, an unbuildable standalone module)
	// this test's own existence is supposed to catch. Forcing it off here,
	// on the subprocess itself, makes that true regardless of whether the
	// caller happens to run this from inside or outside the workspace.
	// Every package an AUTHOR or a TOOL SERVICE imports, which is this module
	// except the excluded subtrees. DISCOVERED, not written out: a list by
	// hand leaves a new top-level package outside the assertion until somebody
	// remembers to add it, and nothing fails to point that out. `./...` is not
	// the answer either — it would list the plan compiler's own dependencies
	// and the assertion below would be about the thing it is meant to keep
	// out. So: list what is here, subtract what is excluded, and fail if the
	// subtraction leaves nothing.
	pkgs := authorFacingPackages(t)
	cmd := exec.Command("go", append([]string{"list", "-deps"}, pkgs...)...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		t.Fatalf("go list -deps failed: %v\n%s\n"+
			"If you just added an import that names toolplane, toolpolicy or "+
			"meter: that is the boundary working. Those packages enforce, meter or "+
			"store, and belong to a daemon or to the CLI. Do NOT add a require for "+
			"github.com/garm-ai/garm to fix this — move the code instead.",
			err, stderr)
	}
	deps := string(out)
	for _, f := range forbidden {
		if strings.Contains(deps, f) {
			t.Fatalf("the contracts module depends on %q. Contracts carry messages and the "+
				"tool-side binding only; anything that enforces, meters or stores belongs "+
				"in the parent module (spec 1.2).", f)
		}
	}
}

// excludedFromAuthorSurface are the package path prefixes a tool author does
// not import, relative to the module root.
//
// `policy` is the plan compiler and belongs to a daemon and to the CLI; the
// forbidden list in the test above names it as an import for the same reason
// it is named here as a subtree. Add to this only with a reason: everything
// not excluded is asserted to stay thin, which is the direction the default
// should fail in.
var excludedFromAuthorSurface = []string{"policy"}

// authorFacingPackages is every package in this module that a tool author or
// a tool service may import: what `go list ./...` finds, minus the excluded
// subtrees.
//
// The point of discovering it is that adding a package to this module cannot
// silently put it outside the boundary. A new top-level directory is in the
// assertion the moment it exists, and a contributor who needed it excluded has
// to say so here, in a list short enough that the exception is visible.
func authorFacingPackages(t *testing.T) []string {
	t.Helper()

	cmd := exec.Command("go", "list", "./...")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		t.Fatalf("go list ./... failed: %v\n%s", err, stderr)
	}

	const mod = "github.com/garm-ai/contracts"
	var keep []string
	for _, p := range strings.Fields(string(out)) {
		rel := strings.TrimPrefix(strings.TrimPrefix(p, mod), "/")
		if excludedSubtree(rel) {
			continue
		}
		keep = append(keep, p)
	}
	if len(keep) == 0 {
		t.Fatal("no author-facing packages found; the exclusion list has eaten the module")
	}
	return keep
}

func excludedSubtree(rel string) bool {
	for _, ex := range excludedFromAuthorSurface {
		if rel == ex || strings.HasPrefix(rel, ex+"/") {
			return true
		}
	}
	return false
}

// The exclusion list is the one hand-maintained thing left, so it is checked
// too: an entry naming a package that does not exist is a rule guarding
// nothing, and the next person to read it would believe it.
func TestEveryExclusionNamesARealPackage(t *testing.T) {
	cmd := exec.Command("go", "list", "./...")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list ./...: %v", err)
	}
	const mod = "github.com/garm-ai/contracts"
	have := map[string]bool{}
	for _, p := range strings.Fields(string(out)) {
		have[strings.TrimPrefix(strings.TrimPrefix(p, mod), "/")] = true
	}
	for _, ex := range excludedFromAuthorSurface {
		found := false
		for rel := range have {
			if rel == ex || strings.HasPrefix(rel, ex+"/") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("excludedFromAuthorSurface names %q, which is not a package in this module. "+
				"Either it was renamed and the exclusion should follow, or it is gone and the "+
				"exclusion should be deleted.", ex)
		}
	}
}
