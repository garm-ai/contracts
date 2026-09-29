# contracts — the contract, and nothing that enforces it

The generated Go, the protos it comes from, and the small hand-written
packages that both sides of a governed call have to agree on. It is a library,
not a process, so the name takes no `d` — see `../spec/CLAUDE.md`, "How they
are named".

Extracted from `garm` at v0.17.0 with `git filter-repo`, keeping the 29
commits that touched these files. `garm` is the CLI; it depends on this.

## The invariant that defines this repository

**`boundary_test.go`.** Read it — the comment carries the history and there is
no point restating it here. What it asserts:

- Whatever a tool author imports for message types must not reach the plan
  compiler, a broker, a store or the CLI.
- `policy` is in this module and is **named in the forbidden list on purpose**.
  It is the daemon's to import, not an author's, and nothing outside it may
  pull it in.
- It runs `go list` with `GOWORK=off` on the subprocess, so it tests what a
  consumer of the published module sees rather than what the `go.work` above
  this checkout resolves.

Two things that follow:

- The package list in that test is written out by hand — `.`, `./audit/...`,
  `./callctx/...`, `./cards/...`, `./garm/...`, `./grant/...`, `./grants/...`,
  `./ledger/...`, `./wire/...`. It is not `./...`, because `./...` would
  include `policy` and the assertion would then be about the thing it exists
  to keep out. **A new top-level package is not covered until it is added to
  that list.**
- If an import you just added makes the test fail, that is the boundary
  working. Move the code; do not add a require for `github.com/garm-ai/garm`.

## The generated files are regenerated, never hand-edited

Every `.pb.go` under `garm/**/v1/` embeds its own file descriptor as a
length-prefixed byte string. Editing an import path in place leaves the prefix
saying the old length, and the package panics at `init` when protobuf tries to
parse the descriptor — not at build, not at the call site, at process start.

That mistake was made during this extraction, and `boundary_test.go` is what
caught it. The fix is always `mise run gen`, never an editor.

`mise run gen-check` fails if the committed output differs from what the
generators produce, and CI runs it.

## What moved, and what did not

Proto **package names did not change**. `garm.tool.v1` is still
`garm.tool.v1`, so no subject, queue group, service name or wire identity
moved in the split. What changed is the Go import path and the `go_package`
option:

```
github.com/garm-ai/garm/contracts/garm/tool/v1  →  github.com/garm-ai/contracts/garm/tool/v1
github.com/garm-ai/garm/policy                  →  github.com/garm-ai/contracts/policy
```

The generated Go was at `contracts/` inside garm and is flattened to the
module root here, so the import path stops stuttering.

`cmd/garm`, `cmd/protoc-gen-garm-go`, `internal/` and `conformance/` stayed in
`garm`. Conformance imports `internal/compile` and `internal/compiler`, so it
tests the CLI's compiler and belongs with it.

## Layout

```
proto/garm/{tool,agent,card,tasks,catalogue,ledger,meta}/v1/   the source
garm/**/v1/            the generated Go. Regenerated, never edited
policy/                the plan compiler — the daemon's, not an author's
policy/redact/         the redactions a plan names
policy/testdata/       the fixture, and its generated connect-go mirror
cards/                 the three card endpoints every tool serves
callctx/ wire/         the NATS hop: the envelope, and the names
ledger/ audit/         the record, and its fail-closed half
grant/ grants/         the digest a person approved, and verifying it
embed.go               the annotation bytes `garm init` vendors
boundary_test.go       the invariant above
```

## Working here

```
mise install     the toolchain
mise run gen     both buf templates — the second one is easy to forget
mise run test    go test ./... -race
mise run lint    buf lint + go vet
mise run ci      what CI runs
```

Every Go task sets `GOWORK=off`. There is a `go.work` above this checkout, and
left to the ambient environment `go list` here resolves through `go.work.sum`
instead of this module's own `go.sum` — which masks exactly the class of gap
(a missing sum entry, an unbuildable standalone module) that this repository
being a module at all is meant to expose.

Task names match every other garm-ai repository on purpose.

## The design record is not in this repository

It lives in **[`garm-ai/spec`](https://github.com/garm-ai/spec)** (private),
checked out beside this one at `../spec/docs/superpowers/`. The ones that
govern this repository:

- `specs/2026-09-25-public-split-design.md` — why these repositories exist
- `decisions/2026-09-25-garm-is-the-cli.md` — why `garm` is the CLI and this
  is not
- `specs/2026-09-24-tool-service-shell-design.md` — the contract artifacts,
  §1.2 the module boundary
- `specs/2026-09-24-call-stack-design.md` — what the annotations govern
- `specs/2026-09-29-cards-and-tasks-as-tools-design.md` — the card vocabulary
  and `garm.tasks.v1`

**Do not create `docs/superpowers/` here.**

Documentation is updated in the same commit as the fact it states: this file,
`README.md`, `KNOWN-GAPS.md`.

## This repository is public

No customer, deployment, tenant or internal hostname appears anywhere in it.
Fixtures use `example.com` shapes and generic subjects.
