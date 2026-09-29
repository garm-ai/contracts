# Known gaps

What this module does not check, and what it does on purpose that will
surprise you. Not an inventory of what works — the code says that, and a file
that repeats it goes stale in a way the code cannot.

## What nothing here checks

**No breaking-change check runs.** `buf.yaml` declares `breaking: use: [FILE]`
and nothing invokes it. This is the module every consumer compiles against and
the one place a breaking change is most expensive, and today the only thing
standing between a renamed field and nine repositories is review. It wants a
`buf breaking --against` in CI, which v0.1.0 now gives it something to compare
against.

**`embed.go` is untested here.** Every test that exercises `AnnotationsProto`,
the vendored paths and `AnnotationSchemaVersion` — `cmd/garm/init_test.go`,
`catalogue_test.go`, `catalogue_source_test.go` — stayed in `garm`. Nothing in
this repository would notice if the embed directives stopped matching the
files, or if `AnnotationSchemaVersion` were bumped when it should not have
been. The version is declared here and defended one repository away.

**`policy/testdata/testdatagarm` cannot be regenerated or verified here.** It
is hand-written to be byte-identical to what the tool plugin emits, and the
plugin is `cmd/protoc-gen-garm-go`, which stayed in `garm`. Its own doc
comment still says it supplies the taxonomy "to toolpolicy's tests" — a
package name that has not existed since the rename to `policy`.

**The boundary test's package list is written out by hand.** A new top-level
package in this module is outside the assertion until somebody adds it, and
nothing fails to point that out. `./...` is not an option: it would include
`policy`, which is the thing being kept out.

## Deliberate, and worth knowing

**`policy/testdata` is a published package with a cross-repository consumer.**
`garmd`'s toolplane and grants tests import `policy/testdata` and
`policy/testdata/testdatagarm`. They are test fixtures with an exported API
that another repository depends on, so a change to the fixture is a change to
somebody else's build.

**The annotations are not published as a buf module.** `buf.yaml` declares no
`name`, so nothing here can be pushed to a registry. Vendoring through
`garm init` is the deliberate answer — a registry is one more host an
enterprise build has to be allowed to reach — but it means a non-Go consumer
has no way to fetch the annotations except by copying the files.
