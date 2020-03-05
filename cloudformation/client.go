package cloudformation

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
)

// A Client is an AWS CloudFormation client.
type Client struct {
	api cloudformationiface.ClientAPI

	// Poller durations
	changeSetWaitTime time.Duration // Wait for changeset to be ready
	pollEvents        time.Duration // Read events for deployment
}

// NewClient creates a new CloudFormation client.
func NewClient(config aws.Config) *Client {
	return &Client{
		api:               cloudformation.New(config),
		changeSetWaitTime: 250 * time.Millisecond,
		pollEvents:        100 * time.Millisecond,
	}
}

// StackByName returns a stack with the given name.
func (c *Client) StackByName(ctx context.Context, name string) (*Stack, error) {
	resp, err := c.api.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	}).Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if strings.Contains(aerr.Message(), "does not exist") {
				return &Stack{
					Name: name,
					ID:   "",
				}, nil
			}
			return nil, err
		}
		return nil, fmt.Errorf("describe stacks: %w", err)
	}
	for _, stack := range resp.DescribeStacksOutput.Stacks {
		if *stack.StackName == name {
			return &Stack{
				Name: name,
				ID:   *stack.StackId,
			}, nil
		}
	}
	return nil, nil
}

// A ChangeSetOpt allows modifying how a change set is created.
type ChangeSetOpt func(input *cloudformation.CreateChangeSetInput)

// WithDescription sets the description on a change set. The description can be
// used to help the user identify the change set.
func WithDescription(desc string) ChangeSetOpt {
	return func(input *cloudformation.CreateChangeSetInput) {
		input.Description = aws.String(desc)
	}
}

// CreateChangeSet creates a new CloudFormation change set.
//
// Blocks until the change set has been created in CloudFormation.
func (c *Client) CreateChangeSet(ctx context.Context, stack *Stack, template *Template, opts ...ChangeSetOpt) (*ChangeSet, error) {
	body, err := json.Marshal(template)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	name := "func-" + time.Now().UTC().Format("20060102-150405")

	input := &cloudformation.CreateChangeSetInput{
		Capabilities:  []cloudformation.Capability{"CAPABILITY_NAMED_IAM"},
		ChangeSetName: aws.String(name),
		ChangeSetType: cloudformation.ChangeSetTypeUpdate,
		ClientToken:   aws.String(name),
		StackName:     aws.String(stack.Name),
		TemplateBody:  aws.String(string(body)),
	}

	for _, opt := range opts {
		opt(input)
	}

	if stack.ID == "" {
		input.ChangeSetType = cloudformation.ChangeSetTypeCreate
	}

	resp, err := c.api.CreateChangeSetRequest(input).Send(ctx)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	var desc string
	if input.Description != nil {
		desc = *input.Description
	}

	cs := &ChangeSet{
		ID:          *resp.CreateChangeSetOutput.Id,
		Name:        name,
		Description: desc,
		Stack:       stack,
		Template:    template,
	}

	if err := cs.loadChanges(ctx, c.api, c.changeSetWaitTime); err != nil {
		return nil, fmt.Errorf("collect changes: %w", err)
	}

	return cs, nil
}

// DeleteChangeSet deletes a change set.
func (c *Client) DeleteChangeSet(ctx context.Context, changeSet *ChangeSet) error {
	if _, err := c.api.DeleteChangeSetRequest(&cloudformation.DeleteChangeSetInput{
		ChangeSetName: aws.String(changeSet.ID),
		StackName:     aws.String(changeSet.Stack.Name),
	}).Send(ctx); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// ExecuteChangeSet executes the changes described in the change set.
func (c *Client) ExecuteChangeSet(ctx context.Context, changeSet *ChangeSet) (*Deployment, error) {
	if len(changeSet.Changes) == 0 {
		return nil, fmt.Errorf("no changes in change set")
	}

	input := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName:      aws.String(changeSet.Name),
		StackName:          aws.String(changeSet.Stack.Name),
		ClientRequestToken: aws.String(changeSet.Name),
	}
	if _, err := c.api.ExecuteChangeSetRequest(input).Send(ctx); err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	return &Deployment{
		ChangeSet: changeSet,
	}, nil
}

// Events watches all events occurring in a deployment. The returned channel is
// closed when the deployment has completed.
func (c *Client) Events(ctx context.Context, deployment *Deployment) <-chan Event {
	events := make(chan Event)

	var since time.Time
	go func() {
		defer func() {
			close(events)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var raw []cloudformation.StackEvent
			resp, err := c.api.DescribeStackEventsRequest(&cloudformation.DescribeStackEventsInput{
				StackName: aws.String(deployment.ChangeSet.Stack.Name),
			}).Send(ctx)
			if err != nil {
				events <- ErrorEvent{Error: err}
				return
			}
			for _, ev := range resp.StackEvents {
				if ev.ClientRequestToken != nil && *ev.ClientRequestToken == deployment.ChangeSet.Name {
					raw = append(raw, ev)
				}
			}

			var list []cloudformation.StackEvent
			last := since
			for _, e := range raw {
				if !e.Timestamp.After(since) {
					continue
				}
				if e.Timestamp.After(last) {
					last = *e.Timestamp
				}
				list = append(list, e)
			}
			since = last

			time.Sleep(c.pollEvents)

			if len(list) == 0 {
				continue
			}

			sort.Slice(list, func(i, j int) bool {
				return list[i].Timestamp.Before(*list[j].Timestamp)
			})

			for _, ev := range list {
				if *ev.ResourceType == "AWS::CloudFormation::Stack" {
					out := stackEvent(ev)
					events <- out
					if out.State == StateComplete {
						return
					}
					continue
				}
				events <- resourceEvent(ev)
			}
		}
	}()

	return events
}
