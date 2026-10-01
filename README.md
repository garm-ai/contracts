# contracts

**The contract garm's tools are written against, as a Go module of its own.**

A tool service, a daemon, a runner and a token exchange all have to agree on
what a governed call looks like: the annotations that declare it, the message
types that carry it, the subject it travels on, the row it leaves behind, and
the token that says a person approved it. That agreement is this repository.

It is not the command line tool. That is
[`garm`](https://github.com/garm-ai/garm), which depends on this. Nothing here
depends on anything of garm's.

## Contents

- [Why it is a module of its own](#why-it-is-a-module-of-its-own)
- [Who imports it](#who-imports-it)
- [What is in here](#what-is-in-here)
- [Using it](#using-it)
- [Regenerating](#regenerating)
- [The relationship to garm](#the-relationship-to-garm)
- [Working here](#working-here)
- [Licence](#licence)

## Why it is a module of its own

`go.mod` has three requires and no indirects:

| | |
|---|---|
| `google.golang.org/protobuf` | every generated type |
| `buf.build/gen/go/bufbuild/protovalidate/...` | the request constraints `garm.tasks.v1` declares, read by `cards` |
| `github.com/go-jose/go-jose/v4` | verifying an approval grant's signature, in `grants` |

Before the split those types lived in the CLI's module. A service that
imported one message type inherited the AWS SDK, cobra, pflag, protocompile,
cel-go, an ANTLR runtime and yaml — from a binary it never ran. `sink` drains
a ledger to Parquet and was pulling cobra.

`boundary_test.go` is what keeps the list short, and it is a package boundary
as well as a module one: nothing an author imports may reach the plan
compiler. See `CLAUDE.md`.

## Who imports it

| | |
|---|---|
| [`garmd`](https://github.com/garm-ai/garmd) | the daemon — `policy`, `policy/redact`, `audit`, `callctx`, `grant`, `ledger`, `wire`, and the catalogue, ledger and tool types |
| [`agentd`](https://github.com/garm-ai/agentd) | the agent runner — the agent, card, catalogue, ledger, meta and tool types, and `policy` |
| [`tasksd`](https://github.com/garm-ai/tasksd) | the task service — `garm.tasks.v1`, `garm.card.v1`, `callctx`, `grant`, `grants`, `wire` |
| [`tool-go`](https://github.com/garm-ai/tool-go) | what a Go tool service imports — `callctx`, `wire`, the tool types |
| [`sts`](https://github.com/garm-ai/sts) | the token exchange — `grant`, to compute the digest it signs over |
| [`sink`](https://github.com/garm-ai/sink) | the ledger drain — the ledger types and `wire` |
| `stack`, `examples`, `tools` | the development stack and the tool trees written against the annotations |

Ten Go modules across nine repositories. They reach these packages through
`github.com/garm-ai/garm` today, at versions from v0.4.0 to v0.17.0; moving
them to this path is the next step, and `KNOWN-GAPS.md` says what that costs.

## What is in here

The generated Go sits at the module root, so an import reads
`github.com/garm-ai/contracts/garm/tool/v1` rather than stuttering through a
`contracts/` directory.

| | |
|---|---|
| `proto/garm/tool/v1/` | The tool annotations, the invocation envelope, the error and attribution types. The only namespace a daemon's version check covers |
| `proto/garm/agent/v1/` | The agent manifest — mode, principal, model, bounds, prompts, the tool allowlist |
| `proto/garm/card/v1/` | The card vocabulary a renderer draws, and the `result_card` / `task_card` templates an author declares |
| `proto/garm/tasks/v1/` | The task queue as governed tools: the contract this repository publishes and `tasksd` serves. `get_task_grant` is the runner's read back of a decided approval, `VERB_READ` in the `escalation` set — the decided event carries a reference and never the grant; `KNOWN-GAPS.md` has what its gate cannot check yet |
| `proto/garm/catalogue/v1/` | The artefact `garm catalogue build` writes and a daemon loads, and the provenance that records every input it was composed from — each local tree, and each Go module at the version `go.mod` resolved it to |
| `proto/garm/ledger/v1/` | One event shape for both planes |
| `proto/garm/meta/v1/` | Ownership — the `owner` a service or method carries |
| `garm/**/v1/` | The generated Go for each of the above. Regenerated, never edited |
| `policy/` | The plan compiler: field annotations become a redaction plan, resolved against a caller and applied per call. `garmd` imports this rather than reimplementing it — two implementations of plan compilation would be a governance bug |
| `policy/redact/` | The redactions a plan can name |
| `cards/` | The three card endpoints every tool serves: the names, and the defaults built from the descriptor |
| `callctx/` | Encoding `InvocationContext` for the NATS hop, behind one function so the envelope can move without touching every call site |
| `wire/` | A route becomes a subject, a queue group and an endpoint name. Both sides of the hop were deriving these separately |
| `ledger/` | The shape of a governed call's record, and the idempotency key for one event |
| `audit/` | The fail-closed half of the record. `ledger.Recorder` must never fail a call; a tool declaring `audit: { fail_closed: true }` says the opposite, so they cannot be one interface |
| `grant/` | The digest over the values a person approved. A minter and a verifier in two repositories have to agree on it byte for byte |
| `grants/` | Verifying a grant — the part that is contract logic rather than daemon policy |
| `embed.go` | The annotation bytes `garm init` vendors, verbatim, and `AnnotationSchemaVersion` |

## Using it

```
go get github.com/garm-ai/contracts@vX.Y.Z
```

```go
import (
    toolv1 "github.com/garm-ai/contracts/garm/tool/v1"
    "github.com/garm-ai/contracts/callctx"
    "github.com/garm-ai/contracts/wire"
)
```

It is not published to a registry and is consumed at a git tag. This module's
own releases start at `v0.1.0`; the nineteen tags that came over from garm with
the history were deleted, because each of them describes the command line tool
and points at a tree with no `go.mod` — so `@latest` would have resolved, on
the day this repository got a remote, to something no consumer could build.

Proto package names did not move in the split. `garm.tool.v1` is still
`garm.tool.v1`, so no subject, service name or wire identity changed — only
the Go import path and the `go_package` option did.

## Regenerating

Two templates, and **both are needed**:

```
buf generate                                  # the contract, into garm/**/v1
buf generate --template buf.gen.testdata.yaml # the policy fixture and its connect-go mirror
```

Or `mise run gen`, which runs both. `mise run gen-check` fails if the
committed output differs from what they produce — that check is the one that
matters most here, because stale generated code in this module is a consumer's
build failure rather than ours.

The testdata template is easy to forget and deliberately does **not** generate
a tool registry: that binds to the enforcing package, which lives in the
daemon's repository.

**Never hand-edit a `.pb.go`.** A generated file embeds its own
`FileDescriptorProto` as a length-prefixed byte string. Editing a path inside
it leaves the prefix describing a length that is no longer there, and the
result is a binary that panics in `filedesc` at init with a slice bound out of
range, at the first import, with nothing in the message about what you
changed. Change the `.proto` and regenerate.

**What a regeneration moves, and what it does not.** Changing a file option —
`go_package`, say — changes the **catalogue digest**, because a catalogue
embeds a `FileDescriptorSet` and a file descriptor carries its options. It
does **not** change a **descriptor hash**: that is computed from message full
names, field numbers, names, cardinality and kind, and never reads an option.
So a service and the catalogue do not have to be rebuilt in lockstep for
routing to keep working; only something pinning a catalogue digest sees it.

## The relationship to garm

`garm` is the command line tool: it scaffolds a proto tree, lints governed
declarations, runs the generators and builds the catalogue. It depends on this
module. **This module depends on nothing of garm's**, and `boundary_test.go`
refuses a build that changes that.

The split is one direction only. `garm` keeps `cmd/garm`,
`cmd/protoc-gen-garm-go`, `internal/` and `conformance/` — conformance tests
the CLI's own compiler, so it belongs beside it.

## Working here

```
mise install    the toolchain
mise run gen    both generators
mise run test   go test ./... -race
mise run ci     what CI runs
```

`GOWORK=off` matters here. There is a `go.work` above this checkout, and the
boundary test forces the workspace off on its own subprocess so that it sees
what a consumer sees rather than what a developer inside the workspace does.
Every task in `mise.toml` does the same, for the same reason.

`KNOWN-GAPS.md` says what is not done.

## Licence

MIT. See [LICENSE](LICENSE).
