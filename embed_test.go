package contracts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/garm-ai/contracts"
)

// vendored is one annotation file this module publishes: the bytes `garm init`
// writes into a consumer's tree, and where it writes them.
type vendored struct {
	name  string
	bytes []byte
	path  string
	src   string
}

func all() []vendored {
	return []vendored{
		{"tool", contracts.AnnotationsProto, contracts.VendoredAnnotationsPath,
			"proto/garm/tool/v1/tool.proto"},
		{"agent", contracts.AgentAnnotationsProto, contracts.VendoredAgentAnnotationsPath,
			"proto/garm/agent/v1/agent.proto"},
		{"card", contracts.CardAnnotationsProto, contracts.VendoredCardAnnotationsPath,
			"proto/garm/card/v1/card.proto"},
		{"meta", contracts.MetaAnnotationsProto, contracts.VendoredMetaAnnotationsPath,
			"proto/garm/meta/v1/meta.proto"},
	}
}

// The embeds are declared here and, until now, exercised only by the CLI's
// tests in another repository. Nothing here would have noticed an //go:embed
// directive pointing at a file that had moved, or bytes that had stopped
// matching the source they claim to be.
//
// "Verbatim and with no added header" is the documented promise, and a later
// drift check being a byte comparison rather than a parse is the reason it is
// made, so a byte comparison is what this asserts.
func TestEachEmbeddedAnnotationIsItsSourceFileVerbatim(t *testing.T) {
	for _, v := range all() {
		t.Run(v.name, func(t *testing.T) {
			onDisk, err := os.ReadFile(v.src)
			if err != nil {
				t.Fatalf("reading %s: %v", v.src, err)
			}
			if len(v.bytes) == 0 {
				t.Fatalf("the embedded %s annotations are empty; the //go:embed "+
					"directive found a file but read nothing from it", v.name)
			}
			if string(v.bytes) != string(onDisk) {
				t.Errorf("the embedded %s annotations are not %s verbatim "+
					"(embedded %d bytes, file %d). `garm init` writes these into a "+
					"consumer's tree and a drift check compares them byte for byte, "+
					"so they cannot differ by so much as a header.",
					v.name, v.src, len(v.bytes), len(onDisk))
			}
		})
	}
}

// The tail of a vendored path is part of the contract: an import of
// "garm/tool/v1/tool.proto" inside a consumer's proto has to resolve to the
// file we put there. Only the root, third_party/, is a choice.
func TestAVendoredPathEndsWithTheImportItHasToSatisfy(t *testing.T) {
	for _, v := range all() {
		t.Run(v.name, func(t *testing.T) {
			// The import a consumer writes is the source path with proto/ off
			// the front: proto/garm/tool/v1/tool.proto is imported as
			// garm/tool/v1/tool.proto.
			imp := strings.TrimPrefix(v.src, "proto/")
			if !strings.HasSuffix(v.path, imp) {
				t.Errorf("vendored path %q does not end with %q, so a consumer "+
					"importing %q would not resolve it", v.path, imp, imp)
			}
			if !strings.HasPrefix(v.path, "third_party/") {
				t.Errorf("vendored path %q is not under third_party/. Annotations "+
					"under a consumer's own proto/ become inputs to their buf "+
					"workspace, which generates Go for them — and two packages "+
					"registering one proto file panic at init.", v.path)
			}
			if filepath.Base(v.path) != filepath.Base(v.src) {
				t.Errorf("vendored path %q and source %q disagree on the file name",
					v.path, v.src)
			}
		})
	}
}

// Four annotation vocabularies are published and each needs an embed. A fifth
// added to proto/ without one is an annotation `garm init` never vendors, and
// the symptom is an author's import that will not resolve — a long way from
// the omission.
//
// garm.catalogue.v1, garm.ledger.v1 and garm.tasks.v1 are deliberately not
// vendored: they are message vocabularies a consumer reaches through the
// generated Go, not annotations written in a consumer's own .proto.
func TestEveryAnnotationVocabularyIsEmbedded(t *testing.T) {
	notVendored := map[string]string{
		"garm/catalogue/v1": "a catalogue is read through the generated Go, never written by an author",
		"garm/ledger/v1":    "a ledger row is read, never declared",
		"garm/tasks/v1":     "a service contract, reached through the generated Go",
	}

	embedded := map[string]bool{}
	for _, v := range all() {
		embedded[filepath.Dir(strings.TrimPrefix(v.src, "proto/"))] = true
	}

	entries, err := os.ReadDir("proto/garm")
	if err != nil {
		t.Fatalf("reading proto/garm: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		vers, err := os.ReadDir(filepath.Join("proto/garm", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, ver := range vers {
			if !ver.IsDir() {
				continue
			}
			pkg := filepath.Join("garm", e.Name(), ver.Name())
			if embedded[pkg] || notVendored[pkg] != "" {
				continue
			}
			t.Errorf("proto/%s has no embed in embed.go and no entry saying why "+
				"it is not vendored. If an author writes this vocabulary in their "+
				"own .proto it needs an embed and a vendored path; if they only "+
				"read it through the generated Go, say so in notVendored here.", pkg)
		}
	}
}

// The schema version is what a daemon range-checks at boot to decide whether
// it can read a catalogue, and it is declared here. Its own doc says it is
// garm.tool.v1 only and bumped only when an older daemon would MISREAD a
// declaration — so the thing worth holding is that it is a plain positive
// integer somebody has to change deliberately, not a value derived from
// anything that moves.
func TestTheAnnotationSchemaVersionIsAPositiveInteger(t *testing.T) {
	if contracts.AnnotationSchemaVersion < 1 {
		t.Errorf("AnnotationSchemaVersion = %d. A daemon reads the current "+
			"version and the two previous and refuses anything outside that "+
			"window at boot; zero or negative is not a window.",
			contracts.AnnotationSchemaVersion)
	}
}
