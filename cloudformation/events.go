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
}

func (StackEvent) isEvent() {}

// A ResourceEvent represents an event on a resource.
type ResourceEvent struct {
	LogicalID string
	Operation ResourceOperation
	State     State
}

func (ResourceEvent) isEvent() {}

// An ErrorEvent contains an error that occurred when watching events for a
// deployment.
type ErrorEvent struct {
	Error error
}

func (ErrorEvent) isEvent() {}

func stackEvent(ev cloudformation.StackEvent) StackEvent {
	return StackEvent{
		Operation: parseStackOp(cloudformation.StackStatus(ev.ResourceStatus)),
		State:     parseState(string(ev.ResourceStatus)),
	}
}

func resourceEvent(ev cloudformation.StackEvent) ResourceEvent {
	return ResourceEvent{
		LogicalID: *ev.LogicalResourceId,
		Operation: parseResourceOp(ev.ResourceStatus),
		State:     parseState(string(ev.ResourceStatus)),
	}
}

// ResourceOperation is the operation that is being performed on a resource.
type ResourceOperation int

//go:generate stringer -type ResourceOperation -trimprefix Resource

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

//go:generate stringer -type StackOperation -trimprefix Stack

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
	if strings.Contains(str, "ROLLBACK") {
		// Rollback, Update rollback, Import rollback
		return StackRollback
	}
	panic(fmt.Sprintf("Unknown stack status %q", str))
}

// State describes the state of a resource that is being deployed.
type State int

//go:generate stringer -type State -trimprefix State

const (
	StateInProgress State = iota
	StateFailed
	StateComplete
	StateSkipped
)

func parseState(status string) State {
	switch {
	case strings.HasSuffix(status, "IN_PROGRESS"):
		return StateInProgress
	case strings.HasSuffix(status, "FAILED"):
		return StateFailed
	case strings.HasSuffix(status, "COMPLETE"):
		return StateComplete
	case strings.HasSuffix(status, "SKIPPED"):
		return StateSkipped
	default:
		panic(fmt.Sprintf("Unknown state in %q", status))
	}
}
