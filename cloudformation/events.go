package cloudformation

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// An Event is an event emitted when watching a deployment.
//
// The interface is closed, the type is always one of:
//   - StackEvent
//   - ResourceEvent
//   - ErrorEvent
type Event interface {
	isEvent()
}

// A StackEvent represents an event on the deployed stack.
type StackEvent struct {
	Operation StackOperation
	State     State
	Reason    string
}

func (StackEvent) isEvent() {}

// A ResourceEvent represents an event on a resource.
type ResourceEvent struct {
	LogicalID string
	Operation ResourceOperation
	State     State
	Reason    string
}

func (ResourceEvent) isEvent() {}

// An ErrorEvent contains an error that occurred when watching events for a
// deployment.
type ErrorEvent struct {
	Error error
}

func (ErrorEvent) isEvent() {}

func stackEvent(ev cloudformation.StackEvent) StackEvent {
	reason := ""
	if ev.ResourceStatusReason != nil {
		reason = *ev.ResourceStatusReason
	}
	return StackEvent{
		Operation: parseStackOp(cloudformation.StackStatus(ev.ResourceStatus)),
		State:     parseState(ev.ResourceStatus),
		Reason:    reason,
	}
}

func resourceEvent(ev cloudformation.StackEvent) ResourceEvent {
	reason := ""
	if ev.ResourceStatusReason != nil {
		reason = *ev.ResourceStatusReason
	}
	return ResourceEvent{
		LogicalID: *ev.LogicalResourceId,
		Operation: parseResourceOp(ev.ResourceStatus),
		State:     parseState(ev.ResourceStatus),
		Reason:    reason,
	}
}

// ResourceOperation is the operation that is being performed on a resource.
type ResourceOperation int

//go:generate go run golang.org/x/tools/cmd/stringer -type ResourceOperation -trimprefix Resource

const (
	ResourceCreate ResourceOperation = iota
	ResourceUpdate
	ResourceDelete
	ResourceImport
)

func parseResourceOp(status cloudformation.ResourceStatus) ResourceOperation {
	str := string(status)
	prefix := str[0:6]
	switch prefix {
	case "CREATE":
		return ResourceCreate
	case "UPDATE":
		return ResourceUpdate
	case "DELETE":
		return ResourceDelete
	case "IMPORT":
		return ResourceImport
	default:
		panic(fmt.Sprintf("Unknown resource status %q (prefix from %s)", prefix, str))
	}
}

// StackOperation is the operation that is being performed on a stack.
type StackOperation int

//go:generate go run golang.org/x/tools/cmd/stringer -type StackOperation -trimprefix Stack

const (
	StackCreate StackOperation = iota
	StackUpdate
	StackDelete
	StackImport
	StackCleanup
	StackReview
	StackRollback
)

func parseStackOp(status cloudformation.StackStatus) StackOperation {
	str := string(status)

	if strings.HasPrefix(str, "ROLLBACK") ||
		strings.HasPrefix(str, "UPDATE_ROLLBACK") ||
		strings.HasPrefix(str, "IMPORT_ROLLBACK") {
		return StackRollback
	}
	switch string(status)[0:6] {
	case "CREATE":
		return StackCreate
	case "UPDATE":
		return StackUpdate
	case "DELETE":
		return StackDelete
	case "IMPORT":
		return StackImport
	case "REVIEW":
		return StackReview
	}
	panic(fmt.Sprintf("Unknown stack status %q", str))
}

// State describes the state of a resource that is being deployed.
type State int

//go:generate go run golang.org/x/tools/cmd/stringer -type State -trimprefix State

const (
	StateInProgress State = iota
	StateFailed
	StateComplete
	StateSkipped
)

func parseState(status cloudformation.ResourceStatus) State {
	str := string(status)
	switch {
	case strings.HasSuffix(str, "IN_PROGRESS"):
		return StateInProgress
	case strings.HasSuffix(str, "FAILED"):
		return StateFailed
	case strings.HasSuffix(str, "COMPLETE"):
		return StateComplete
	case strings.HasSuffix(str, "SKIPPED"):
		return StateSkipped
	default:
		panic(fmt.Sprintf("Unknown state in %q", str))
	}
}
