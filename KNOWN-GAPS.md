# Known gaps

What this module does not check, and what it does on purpose that will
surprise you. Not an inventory of what works — the code says that, and a file
that repeats it goes stale in a way the code cannot.

## What nothing here checks

**`get_task_grant` is declared and its per-task gate cannot be implemented
yet.** This is the biggest thing to know about `garm.tasks.v1` today, and it is
a gap in the PLATFORM that this contract records rather than one it can close.

The method exists because the decided event carries a reference and never the
grant (R5): a bearer on a stream is readable by anything that can consume the
subject for that stream's retention, and a replay would replay a credential. So
a woken run fetches the approval as a governed call. The ruling that shaped the
method is `decisions/2026-10-01-tasksd-is-the-only-task-service.md`, amended the
same day — audience RUNNER, in `escalation`, not a field on `Task`, and **gated
on the run and not only on the audience**, because an audience is a listing rule
and would leave any runner able to read any task's grant.

**There is nothing attested to gate on.** `garmd` builds the invocation's
attribution with a tenant and a correlation id and nothing else
(`internal/toolplane/core.go`, `withInvocationContext`), so
`CallContext.run_id` — field 3, declared in this module — reaches a tool service
EMPTY on every call through the daemon, whatever the runner put on its own
request. A gate comparing it to the task's `run_id` therefore refuses
everything. `tasksd` implements the gate that way on purpose, fail-closed, so
the method is unreachable rather than wrongly reachable; it is not a working
authorization check and must not be read as one.

The answer being designed is an opaque single-task capability `create_task`
returns and the runner presents — possession rather than an unattested
assertion. `GetTaskGrantRequest` is a message of its own, not the shared
`garm.card.v1.TaskRef`, so that capability lands as field 2 of a message only
this method takes. **It will move the wire shape a second time**, and declaring
it now to save that rebuild would mean guessing a contract that is still being
decided.

**The same emptiness used to break `create_task`, and this contract is where the
fix is.** `tasksd`'s `Create` requires a run to have something to signal and read
it off the invocation, so every call through the daemon was refused — which is why
nothing in the platform had ever opened a task. `CreateTaskRequest.run_id`
(field 11) now carries it, because the runner is the only party that knows which
run it is executing and **the daemon must not learn**: knowing nothing about agents
or runs is one of garmd's invariants, and `CallContext.run_id` being declared is
not permission to make the daemon fill one in.

It is an **assertion** — nothing attests it — used for **routing** and never for
authorization. It keys `garm.tasks.v1.decided.<tenant>.<run_id>`, and routing
decides who hears a decision, never who may act on one. The worst a runner can do
by naming another run is wake it spuriously; that runner is then refused the
approval, holding no capability for a task that was never its. Noise, not
privilege — and it is why the grant gate above cannot be built on the same value.

**`escalation` holds `create_task` and `get_task_grant`, and the runner's policy
gains one verb.** Opening an escalation and collecting its answer are two halves
of one act, which is what naming a set for the act rather than the caller buys.

The amendment said the set choice meant no claims policy changed. Verified
against all four axes of garmd's visibility predicate rather than the set alone —
it is an AND over verb, clearance, compartments and sets, with **no implication
between verbs** — that turned out to be true only for a method declaring
`VERB_WRITE`, because the runner held `verbs: [WRITE]`. **That was the wrong trade
and it was reversed.** A verb describes the act before it is a policy axis;
`get_task_grant` returns a value and moves nothing, which is also what its own
`effects.idempotent` says, so a `VERB_WRITE` declaration would have put a write in
the catalogue an auditor reads where nothing is written — and would have set the
precedent that a policy too narrow is fixed by relabelling the tool. This verb
vocabulary is garm's own invention with no upstream to appeal to, so it is worth
exactly what each declaration keeps it worth.

So the method declares `VERB_READ`, `CLEARANCE_PUBLIC`, no compartment and
`escalation`, and the runner principal is granted `READ` beside `WRITE` in
`sts/deploy/claims.yaml` and `examples/bank/auth/claims.yaml`. **The set is what
bounds that widening, not the verb:** scoped to `escalation`, which holds
`create_task` and `get_task_grant` and nothing else — and no other contract in the
platform declares a set by that name — the added verb reaches one more tool rather
than every `CLEARANCE_PUBLIC`, uncompartmented, `VERB_READ` tool in the composed
catalogue. `garm/tasks/v1/scoping_test.go` pins all four axes and the membership by
name; `tasksd` pins the identical set on its own copy.

**The wire shape MOVED TWICE in one round, and that is the standing cost of a new
method and a new field.** It is
`8939ac00b2a441346826759b77d72dc568a9bb0fa32d50732cc72b3c166b5a63` now, from
`6d981dae…`: `GetTaskGrantRequest` and `TaskGrant` are both new messages, which
took it to `b80da142…3df2`, and then `run_id` on `CreateTaskRequest` took it to the
value above. The two were batched deliberately — each move costs a catalogue
rebuild and a command line tool release, so two in one release is one turn of that
treadmill rather than two. `get_task_grant` going from `VERB_WRITE` to `VERB_READ`
in the same round moved it not at all, a verb being an option, confirmed by running
the golden test either side rather than assumed.

**Every catalogue carrying `garm.tasks.v1` has to be rebuilt by a command line tool
linking this version**, and until it is, `garmd` quarantines `tasksd` on the
mismatch. `garm/tasks/v1/wireshape_test.go` now pins the value here as well as in
`tasksd`, so each of the two copies that hold this package during the switchover
guards its own — before this, an edit here moved the shape and nothing in this
repository said so.

**contracts v0.10.0 moved no descriptor hash, and that is a structural fact
about the walk, not an oversight.** That release landed three shapes at
once — `grants.Claims.Caveats`, `InvocationContext.grant_jti` (field 11), and
`garm.agent.v1.Consent`/`Limits`/`TriggerKind` with `AgentPolicy.consent`
(field 11) — and a reader who has just read the paragraph above, about
`garm.tasks.v1` moving twice in one round, will reasonably expect a third move
here. It did not happen. `descriptorHash`
(`garm/tasks/v1/wireshape_test.go`) starts from `TasksService`'s methods and
walks their inputs and outputs transitively through message-kind fields,
reading field numbers, names, cardinalities and kinds — never an option.
`InvocationContext` is a field of no message: it crosses the NATS hop beside
the request rather than living inside any RPC's request or response, so it is
unreachable from `TasksService`'s signature and outside the walk entirely —
`grant_jti` living on it moves nothing. `Consent` reaches `AgentPolicy` only
through the `agent` extension on `google.protobuf.ServiceOptions`, which is an
option, and the walk excludes every option by design. So that release is
strictly additive against the hash: no catalogue rebuild is forced by it, and
a consumer adopts v0.10.0 for the new types rather than because anything
broke.

**v0.12.0 checked the same question a second time, deliberately, and the
answer is the same for a sharper reason.** It adds `ToolPolicy.stateful_caveats`
and `InvocationContext.agent`, and `garm.tasks.v1`'s own tools carry
`(garm.tool.v1.tool)` annotations — `ToolPolicy` reached as an OPTION on
`TasksService`'s methods, which is precisely the case the walk is built to
exclude. Confirmed by running `TestTheWireShapeIsTheValueTasksdAlsoPins` both
before and after rather than assumed: the value stayed
`8939ac00b2a441346826759b77d72dc568a9bb0fa32d50732cc72b3c166b5a63`. This is
the second release running where an addition to an option message shares a
file with option annotations on the service the hash walks, and both times
the hash held — because the walk starts at a method's input and output
message types and a `MethodOptions` extension is never one of those, however
deep the message it carries.

**`InvocationContext` now has a shape guard.**
`garm/tool/v1/envelope_shape_test.go`'s `TestTheEnvelopeShapeIsPinned` walks
`InvocationContext` and the messages it reaches (`CallContext`,
`InvocationPrincipal`, `Act`) with the same field/number/cardinality/kind walk
`wireshape_test.go` uses for `garm.tasks.v1`, anchored on the message itself
rather than on a service, because `InvocationContext` belongs to no service —
it crosses the NATS hop between a garmd build and a tool build, not any RPC's
input or output, so `descriptorHash` never reaches it. A garmd that adds,
removes or renumbers a field here, built against a tool compiled on an older
copy, now fails a test on either side instead of drifting silently. Watched
fail once, with a real value, before the constant was filled in — the same
discipline `wireshape_test.go` holds — and once more with a field added to
`InvocationContext` locally and reverted, to confirm the guard is not hashing
a constant that could never move. Also worth someone checking, and not
established either way by this change: `contract_version` (field 10) has
existed on `InvocationContext` since before v0.10.0, and nothing confirms
anything downstream actually reads it — an unread version field is not a
guard, whatever its name suggests.

**`Provenance.inputs` is declared and nothing writes it yet.** It records what
a catalogue was composed from, and stays empty until `garm catalogue build`
composes from a manifest — which the contract defines as "this builder did not
say", never "composed from nothing". The same message shows how long that state
can last: `built_at` has been declared since the first release and no builder
has ever stamped it. So read the field, and do not require it.

**`policy/testdata/testdatagarm` is a shim with one consumer left, and a
date.** It exists only because garmd's toolplane and grants tests import it,
and they cannot call [`policy.DeclaredCompartments`] until garmd moves off
`github.com/garm-ai/garm` and onto this module. When it does, those four test
files change one line each and the package is deleted.

It no longer hand-writes a generator's output or promise byte-identity with a
plugin in another repository, and no check is owed by `garm` any more: the
variable is derived from the same call new code makes, and a test in `policy`
asserts the two agree.

**A package under `testdata/` is never built by `go test ./...`.** The go tool
skips directories with that name, so `policy/testdata` and everything below it
is outside `go vet ./...`, outside the boundary test's discovery, and outside
any assertion this repository makes about itself — while four test files in
garmd compile against it. That is the reason the shim could carry an unchecked
promise for as long as it did. Nothing about it is fixed by the shim's
deprecation: the next fixture to grow an exported API will have the same hole.
Promoting the fixture out of `testdata/` into a named package would close it,
at the cost of changing garmd's import path, and is the obvious thing to do
when garmd migrates.

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
