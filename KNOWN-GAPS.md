# Known gaps

What this module does not check, and what it does on purpose that will
surprise you. Not an inventory of what works — the code says that, and a file
that repeats it goes stale in a way the code cannot.

## What nothing here checks

**`policy/testdata/testdatagarm` cannot be verified here, and the check is
owed by `garm`.** It is hand-written to be byte-identical to what the tool
plugin emits, and the plugin is `cmd/protoc-gen-garm-go`, which stayed with
the command line tool. Nothing in this repository can tell you the file is
still what the plugin would produce. The fix is not here: it is a check in
`garm`'s CI that runs the plugin and compares. Listed here because this is
where the stale file would be, not because this is where the work is.

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
