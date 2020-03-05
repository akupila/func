package cloudformation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
)

// A ChangeSet describes a set of changes on a CloudFormation stack.
type ChangeSet struct {
	ID          string
	Name        string
	Description string
	Changes     []Change
	Stack       *Stack
	Template    *Template
}

// A Change describes a change within a ChangeSet to be performed on a stack.
type Change struct {
	Operation ResourceOperation
	LogicalID string
}

// change waits until a change set has been created and returns the changes in
// it.
//
// Cancelling the context will stop polling.
func (c *ChangeSet) loadChanges(ctx context.Context, api cloudformationiface.ClientAPI, pollTime time.Duration) error {
	var changes []cloudformation.Change
	var nextToken *string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := api.DescribeChangeSetRequest(&cloudformation.DescribeChangeSetInput{
			ChangeSetName: aws.String(c.ID),
			StackName:     aws.String(c.Stack.Name),
			NextToken:     nextToken,
		}).Send(ctx)
		if err != nil {
			return fmt.Errorf("describe change set: %w", err)
		}

		switch resp.Status {
		case cloudformation.ChangeSetStatusFailed:
			reason := *resp.StatusReason
			if strings.Contains(reason, "The submitted information didn't contain changes") {
				return nil
			}
			return fmt.Errorf("list changes failed: %s", strings.ToLower(reason))
		case cloudformation.ChangeSetStatusCreateComplete:
			changes = append(changes, resp.DescribeChangeSetOutput.Changes...)
			nextToken = resp.DescribeChangeSetOutput.NextToken
			if nextToken == nil {
				cc, err := convertChanges(changes)
				if err != nil {
					return err
				}
				c.Changes = cc
				// Done
				return nil
			}
			// Get next page without sleep
		case cloudformation.ChangeSetStatusCreatePending, cloudformation.ChangeSetStatusCreateInProgress:
			time.Sleep(pollTime)
		default:
			return fmt.Errorf("unknown status %q", resp.Status)
		}
	}
}

func convertChanges(changes []cloudformation.Change) ([]Change, error) {
	out := make([]Change, len(changes))
	for i, c := range changes {
		var op ResourceOperation
		switch c.ResourceChange.Action {
		case cloudformation.ChangeActionAdd:
			op = ResourceCreate
		case cloudformation.ChangeActionModify:
			op = ResourceUpdate
		case cloudformation.ChangeActionRemove:
			op = ResourceDelete
		case cloudformation.ChangeActionImport:
			op = ResourceImport
		default:
			return nil, fmt.Errorf("unknown action %q", c.ResourceChange.Action)
		}
		out[i] = Change{
			Operation: op,
			LogicalID: *c.ResourceChange.LogicalResourceId,
		}
	}
	return out, nil
}
