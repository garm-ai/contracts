// Package contracts is the root of garm's generated tool contracts.
//
// It is a separate Go module, and now a separate repository, on purpose. A
// tool service depends on this module and on garmtool, and on nothing else:
// the boundary is what stops a tool importing toolplane and attempting the
// governance chain locally — a second, unreviewed implementation of a policy
// that already ran before the request crossed NATS.
//
// The second reason is weight. This module requires three things. The command
// line tool it was extracted from requires eleven, including an S3 client and
// a CLI framework, and every consumer of a message type used to inherit them.
//
// It is not published to a registry. Consume it at a git tag:
//
//	go get github.com/garm-ai/contracts@v0.1.0
package contracts
