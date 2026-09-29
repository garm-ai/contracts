// Package policy turns a declaration into a plan.
//
// It reads the annotations on a method and its messages and works out what a
// given caller may send and may be shown: which fields are in the request a
// tool is allowed to receive, which fields of the answer survive the viewer's
// clearance and compartments, and how the ones that do not are removed. The
// plan is the answer; applying it is the caller's.
//
// # Who may import this
//
// A daemon, and the command line tool. **Not a tool service, and not an
// author's generated code.** A tool that can compile a plan is a tool that
// will eventually enforce one locally: a second implementation of a policy
// that already ran, in a process nobody reviews as an enforcement point, on
// the far side of the hop that was supposed to have settled it.
//
// That is a rule about the dependency graph rather than about intent, so it
// is enforced as one. `boundary_test.go` in this module's root lists this
// package as forbidden and fails if anything a tool author imports can reach
// it. The generated message types in garm/ do not, and must not start.
//
// The name is `policy` because the thing it produces is the policy's shape
// for one call. It was `toolpolicy` until the enforcing half moved to the
// daemon's repository and the qualifier stopped distinguishing it from
// anything.
package policy
