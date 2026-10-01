# Known gaps

What this module does not check, and what it does on purpose that will
surprise you. Not an inventory of what works — the code says that, and a file
that repeats it goes stale in a way the code cannot.

## What nothing here checks

**`create_task` declares `escalation` now, and no set reaches the read back that
pairs with it.** This entry used to say the question was not this contract's
call alone, on the reading that a set on `create_task` would force **every
agent's manifest** to carry it — because the runner's token was taken to be
narrowed by the calling agent's manifest. That reading was wrong about who
calls it. A runner opens a task as its OWN service principal, not as a
delegated agent token, so the set lives on that one principal's claims and no
agent's ceiling changes.

The ruling is therefore made: `escalation` holds `create_task` alone. A caller
granted it can open a task and do nothing else — not list the queue, not read a
task back, not claim, decide or triage. The alternatives were both worse: with no
set at all `create_task` is reachable only by an UNSCOPED caller, which hands the
platform's one runner every `CLEARANCE_PUBLIC`, uncompartmented, `VERB_WRITE`
tool in the composed catalogue, and whatever that principal holds every run it
executes can reach; and granting it `triage` instead leaves `create_task`
unreachable while handing it the whole queue. `garm/tasks/v1/scoping_test.go`
pins the membership by name, and `tasksd` pins the identical membership on its
own copy of this package.

**It did not move the wire shape.** A tool set is an option, and the descriptor
hash both `garm catalogue build` and a tool service compute walks only each
method's input and output message fields. So no deployed catalogue needs
rebuilding for this, and the two copies of `garm.tasks.v1` still hash alike —
which is what keeps a deployment out of quarantine while two modules hold the
package. Verified by running `tasksd`'s golden constant either side of the
change, not reasoned about.

**What is still open is the read back, and it is `tasksd`'s entry to close.** The
decided event carries a task id and an outcome and never the grant, so a runner
that wakes reads the task through `get_task` — which is in `triage`. A runner
holding only `escalation` cannot do that, and granting it `triage` would hand
every run the queue. Whatever closes it is a change to this contract rather than
a deployment's workaround.

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
