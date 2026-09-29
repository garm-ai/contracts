# Known gaps

What is not done, and what the split left half finished. Kept honest by the
changes that close the entries.

## The split is not finished

**No consumer has moved.** Ten Go modules across nine repositories still
require `github.com/garm-ai/garm` for these packages, at versions spanning
v0.4.0 to v0.17.0:

| | |
|---|---|
| v0.17.0 | `garmd`, `tasksd` |
| v0.16.0 | `agentd`, `examples`, `stack` |
| v0.14.2 | `tools/web`, `tools/taxonomy` |
| v0.13.0 | `sts` |
| v0.8.0 | `sink` |
| v0.4.0 | `tool-go` |

Nothing imports `github.com/garm-ai/contracts` yet. Until they move, the
dependency win this module exists for is not collected anywhere: `sink` is
still pulling cobra through `garm`.

**`garm` has not been cut down.** `garm/contracts/`, `garm/policy/` and
`garm/embed.go` are still there, and apart from the import paths the files are
the same ones. The same code exists in two places, and a change made in one of
them is silently absent from the other, in both directions, until the CLI's
copy is deleted and it takes a require on this module.

**This module starts at v0.1.0, and that is not the version of the contract
it holds.** The nineteen tags that came over with the history were garm's, and
every one of them was deleted before this repository had a remote: at the tag
named v0.17.0 this tree has no `go.mod` at all and the Go sits under
`contracts/`, so `go get …@latest` would have resolved to something no
consumer could build. Leaving them to look like history would have been worse
than useless.

Cutting the first tag as v0.17.0 to match the garm release this was extracted
from was considered and rejected. It would claim sixteen prior releases of a
module that has none, and the correspondence it buys is worth one migration
and then becomes a lie the first time either side releases alone. The mapping
is written down instead: **contracts v0.1.0 is the contract as of garm
v0.17.0**, and after that the two numbers mean different things on purpose.

## Rebuilding consumers

**The catalogue digest moves; the descriptor hash does not.** A catalogue
embeds a `FileDescriptorSet` that includes every dependency, and a
`FileDescriptorProto` carries its file's options — so the new `go_package`
changes the bytes and therefore the catalogue's sha256, for every catalogue
rebuilt against the vendored annotations. It does **not** change the per-package
`DescriptorHash` that a tool service advertises and the daemon reconciles at
mount: that is computed over message full names, field numbers, names,
cardinality and kind only, and deliberately never touches options. So a
service and a catalogue do not have to be rebuilt in lockstep for routing to
keep working — but anything pinning a catalogue digest, or comparing two
catalogues, sees a change with no wire change behind it.

**Every vendored annotation file is one line out of date.** `garm init`
writes these bytes verbatim into a consumer's `third_party/`, and the only
difference between the old copy and this one is the `go_package` line. Since
they are written verbatim so that a later drift check is a byte comparison
rather than a parse, every tree that ran `garm init` before the split now
reports drift, and re-vendoring is a catalogue rebuild for the reason above.

## What nothing here checks

**No breaking-change check runs.** `buf.yaml` declares `breaking: use: [FILE]`
and nothing invokes it. This is the module every consumer compiles against and
the one place a breaking change is most expensive, and today the only thing
standing between a renamed field and nine repositories is review. It wants a
`buf breaking --against` in CI once there is a tag to compare against — which
is blocked on the tag gap above.

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

## Left as it is

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

**`policy` has no package doc comment.** The most consequential package in the
module, and what it is and who may import it is documented only in
`boundary_test.go` and in `CLAUDE.md`.
