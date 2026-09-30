# Known gaps

What this module does not check, and what it does on purpose that will
surprise you. Not an inventory of what works — the code says that, and a file
that repeats it goes stale in a way the code cannot.

## What nothing here checks

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
