package cloudformation

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestClient_StackByName(t *testing.T) {
	onDescribe := func(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
		if *input.StackName == "existing" {
			return &cloudformation.DescribeStacksOutput{
				Stacks: []cloudformation.Stack{
					{StackName: aws.String("existing"), StackId: aws.String("existing-stack")},
				},
			}, nil
		}
		msg := fmt.Sprintf("Stack with id %s does not exist", *input.StackName)
		return nil, awserr.New("ValidationError", msg, nil)
	}

	tests := []struct {
		name           string
		stackName      string
		describeStacks DescribeStacksHook
		want           *Stack
		wantErr        bool
	}{
		{
			name:           "Existing",
			stackName:      "existing",
			describeStacks: onDescribe,
			want:           &Stack{Name: "existing", ID: "existing-stack"},
		},
		{
			name:           "NonExisting",
			stackName:      "nonexisting",
			describeStacks: onDescribe,
			want:           &Stack{Name: "nonexisting", ID: ""},
		},
		{
			name:      "Error",
			stackName: "foo",
			describeStacks: func(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
				return nil, fmt.Errorf("err")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cli := &Client{
				api: &mockCF{
					DescribeStacks: tc.describeStacks,
				},
			}
			got, err := cli.StackByName(context.Background(), tc.stackName)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Error = %v, want err = %t", err, tc.wantErr)
			}
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

func TestClient_CreateChangeSet(t *testing.T) {
	noChanges := func(input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
		return &cloudformation.DescribeChangeSetOutput{
			Status:       cloudformation.ChangeSetStatusFailed,
			StatusReason: aws.String("The submitted information didn't contain changes"),
		}, nil
	}

	tests := []struct {
		name              string
		stack             *Stack
		template          *Template
		opts              []ChangeSetOpt
		createChangeSet   CreateChangeSetHook
		describeChangeSet DescribeChangeSetHook
		want              *ChangeSet
		wantErr           bool
	}{
		{
			name:     "Create",
			stack:    &Stack{Name: "teststack"},
			template: &Template{},
			createChangeSet: func(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
				return &cloudformation.CreateChangeSetOutput{Id: aws.String("test-id"), StackId: input.StackName}, nil
			},
			describeChangeSet: noChanges,
			want: &ChangeSet{
				ID:      "test-id",
				Name:    "func-00000000-000000",
				Changes: nil,
			},
		},
		{
			name:     "Update",
			stack:    &Stack{ID: "existing", Name: "teststack"}, // ID is set -> stack exists
			template: &Template{},
			createChangeSet: func(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
				if input.ChangeSetType != cloudformation.ChangeSetTypeUpdate {
					return nil, fmt.Errorf("type is %q, want %q", input.ChangeSetType, cloudformation.ChangeSetTypeUpdate)
				}
				return &cloudformation.CreateChangeSetOutput{Id: aws.String("test-id"), StackId: input.StackName}, nil
			},
			describeChangeSet: noChanges,
			want: &ChangeSet{
				ID:      "test-id",
				Name:    "func-00000000-000000",
				Changes: nil,
			},
		},
		{
			name:     "Changes",
			stack:    &Stack{Name: "teststack"},
			template: &Template{},
			createChangeSet: func(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
				return &cloudformation.CreateChangeSetOutput{Id: aws.String("test-id"), StackId: input.StackName}, nil
			},
			describeChangeSet: func(input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
				var nextToken string
				if input.NextToken != nil {
					nextToken = *input.NextToken
				}
				switch nextToken {
				case "":
					return &cloudformation.DescribeChangeSetOutput{
						Status: cloudformation.ChangeSetStatusCreateComplete,
						Changes: []cloudformation.Change{
							makeChange(cloudformation.ChangeActionAdd, "A"),
						},
						NextToken: aws.String("next"),
					}, nil
				case "next":
					return &cloudformation.DescribeChangeSetOutput{
						Status: cloudformation.ChangeSetStatusCreateComplete,
						Changes: []cloudformation.Change{
							makeChange(cloudformation.ChangeActionAdd, "B"),
						},
						NextToken: nil,
					}, nil
				default:
					panic("Invalid next token: " + nextToken)
				}
			},
			want: &ChangeSet{
				ID:   "test-id",
				Name: "func-00000000-000000",
				Changes: []Change{
					{Operation: ResourceCreate, LogicalID: "A"},
					{Operation: ResourceCreate, LogicalID: "B"},
				},
			},
		},
		{
			name:     "Description",
			stack:    &Stack{Name: "teststack"},
			template: &Template{},
			opts: []ChangeSetOpt{
				WithDescription("test description"),
			},
			createChangeSet: func(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
				wantDesc := "test description"
				if *input.Description != wantDesc {
					return nil, fmt.Errorf("description does not match; got %q, want %q", *input.Description, wantDesc)
				}
				return &cloudformation.CreateChangeSetOutput{Id: aws.String("id"), StackId: input.StackName}, nil
			},
			describeChangeSet: noChanges,
			want: &ChangeSet{
				ID:          "id",
				Name:        "func-00000000-000000",
				Description: "test description",
			},
		},
		{
			name:     "Error",
			stack:    &Stack{Name: "teststack"},
			template: &Template{},
			createChangeSet: func(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error) {
				return nil, awserr.New("TestError", "err", nil)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cli := &Client{
				api: &mockCF{
					CreateChangeSet:   tc.createChangeSet,
					DescribeChangeSet: tc.describeChangeSet,
				},
			}
			got, err := cli.CreateChangeSet(context.Background(), tc.stack, tc.template, tc.opts...)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Err = %v, want err = %t", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			tc.want.Stack = tc.stack
			tc.want.Template = tc.template
			opts := []cmp.Option{
				compareTemplate(),
				cmp.FilterPath(func(p cmp.Path) bool {
					return p.String() == "Name"
				}, cmp.Comparer(func(a, b string) bool {
					return stripNumbers(a) == stripNumbers(b)
				})),
			}
			if diff := cmp.Diff(got, tc.want, opts...); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

func makeChange(action cloudformation.ChangeAction, id string) cloudformation.Change {
	return cloudformation.Change{
		ResourceChange: &cloudformation.ResourceChange{
			Action:            action,
			LogicalResourceId: aws.String(id),
		},
	}
}

func TestClient_DeleteChangeSet(t *testing.T) {
	tests := []struct {
		name            string
		changeSet       *ChangeSet
		deleteChangeSet DeleteChangeSetHook
		wantErr         bool
	}{
		{
			name: "Delete",
			changeSet: &ChangeSet{
				ID:    "foo",
				Stack: &Stack{Name: "bar"},
			},
			deleteChangeSet: func(input *cloudformation.DeleteChangeSetInput) (*cloudformation.DeleteChangeSetOutput, error) {
				return &cloudformation.DeleteChangeSetOutput{}, nil
			},
		},
		{
			name: "Error",
			changeSet: &ChangeSet{
				ID:    "foo",
				Stack: &Stack{Name: "bar"},
			},
			deleteChangeSet: func(input *cloudformation.DeleteChangeSetInput) (*cloudformation.DeleteChangeSetOutput, error) {
				return nil, awserr.New("TestError", "err", nil)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cli := &Client{
				api: &mockCF{
					DeleteChangeSet: tc.deleteChangeSet,
				},
			}
			err := cli.DeleteChangeSet(context.Background(), tc.changeSet)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Err = %v, want err = %t", err, tc.wantErr)
			}
		})
	}
}

func TestClient_ExecuteChangeSet(t *testing.T) {
	tests := []struct {
		name             string
		changeSet        *ChangeSet
		executeChangeSet ExecuteChangeSetHook
		wantErr          bool
	}{
		{
			name: "Execute",
			changeSet: &ChangeSet{
				ID:    "foo",
				Stack: &Stack{Name: "bar"},
				Changes: []Change{
					{Operation: ResourceCreate, LogicalID: "baz"},
				},
			},
			executeChangeSet: func(input *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error) {
				return &cloudformation.ExecuteChangeSetOutput{}, nil
			},
		},
		{
			name: "NoChanges",
			changeSet: &ChangeSet{
				ID:      "foo",
				Stack:   &Stack{Name: "bar"},
				Changes: nil,
			},
			wantErr: true,
		},
		{
			name: "Error",
			changeSet: &ChangeSet{
				ID:    "foo",
				Stack: &Stack{Name: "bar"},
			},
			executeChangeSet: func(input *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error) {
				return nil, awserr.New("TestError", "err", nil)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cli := &Client{
				api: &mockCF{
					ExecuteChangeSet: tc.executeChangeSet,
				},
			}
			got, err := cli.ExecuteChangeSet(context.Background(), tc.changeSet)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Err = %v, want err = %t", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			want := &Deployment{
				ChangeSet: tc.changeSet,
			}
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

var numRe = regexp.MustCompile(`\d+`)

func stripNumbers(input string) string {
	return numRe.ReplaceAllString(input, "")
}

func TestClient_Events(t *testing.T) {
	var (
		deploy = &Deployment{
			ChangeSet: &ChangeSet{Name: "changeset-name", Stack: &Stack{Name: "stack-name"}},
		}
		deployOtherCS = &Deployment{
			ChangeSet: &ChangeSet{Name: "other-changeset-name", Stack: &Stack{Name: "stack-name"}},
		}
	)

	tests := []struct {
		name   string
		events DescribeStackEventsHook
		want   []Event
	}{
		{
			name: "LifeCycle",
			events: mockEvents{
				makeStackEvent(deploy, cloudformation.ResourceStatusUpdateInProgress),
				makeResourceEvent(deploy, "Test", cloudformation.ResourceStatusUpdateInProgress),
				makeResourceEvent(deploy, "Test", cloudformation.ResourceStatusUpdateComplete),
				makeStackEvent(deploy, cloudformation.ResourceStatusUpdateComplete),
			}.Paginate(1),
			want: []Event{
				StackEvent{Operation: StackUpdate, State: StateInProgress},
				ResourceEvent{Operation: ResourceUpdate, LogicalID: "Test", State: StateInProgress},
				ResourceEvent{Operation: ResourceUpdate, LogicalID: "Test", State: StateComplete},
				StackEvent{Operation: StackUpdate, State: StateComplete},
			},
		},
		{
			name: "OnlyCurrentChangeSet",
			events: mockEvents{
				makeStackEvent(deploy, cloudformation.ResourceStatusUpdateInProgress),
				makeResourceEvent(deploy, "Foo", cloudformation.ResourceStatusCreateInProgress),
				makeResourceEvent(deploy, "Bar", cloudformation.ResourceStatusDeleteInProgress),
				makeResourceEvent(deployOtherCS, "Xxx", cloudformation.ResourceStatusUpdateInProgress), // Not
				makeResourceEvent(deployOtherCS, "Yyy", cloudformation.ResourceStatusUpdateInProgress), // in
				makeResourceEvent(deployOtherCS, "Xxx", cloudformation.ResourceStatusUpdateComplete),   // same
				makeResourceEvent(deployOtherCS, "Yyy", cloudformation.ResourceStatusUpdateComplete),   // change set
				makeResourceEvent(deploy, "Foo", cloudformation.ResourceStatusCreateComplete),
				makeResourceEvent(deploy, "Bar", cloudformation.ResourceStatusDeleteComplete),
				makeStackEvent(deploy, cloudformation.ResourceStatusUpdateComplete),
			}.Paginate(2),
			want: []Event{
				StackEvent{Operation: StackUpdate, State: StateInProgress},
				ResourceEvent{Operation: ResourceCreate, LogicalID: "Foo", State: StateInProgress},
				ResourceEvent{Operation: ResourceDelete, LogicalID: "Bar", State: StateInProgress},
				ResourceEvent{Operation: ResourceCreate, LogicalID: "Foo", State: StateComplete},
				ResourceEvent{Operation: ResourceDelete, LogicalID: "Bar", State: StateComplete},
				StackEvent{Operation: StackUpdate, State: StateComplete},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cli := &Client{
				api: &mockCF{
					DescribeStackEvents: tc.events,
				},
			}

			var got []Event
			for ev := range cli.Events(context.Background(), deploy) {
				got = append(got, ev)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

func makeStackEvent(deploy *Deployment, status cloudformation.ResourceStatus) cloudformation.StackEvent {
	return cloudformation.StackEvent{
		ClientRequestToken: aws.String(deploy.ChangeSet.Name),
		LogicalResourceId:  aws.String(deploy.ChangeSet.Stack.Name),
		ResourceStatus:     status,
		ResourceType:       aws.String("AWS::CloudFormation::Stack"),
		StackName:          aws.String(deploy.ChangeSet.Stack.Name),
		Timestamp:          aws.Time(time.Now()),
	}
}

func makeResourceEvent(deploy *Deployment, logicalID string, status cloudformation.ResourceStatus) cloudformation.StackEvent {
	return cloudformation.StackEvent{
		ClientRequestToken: aws.String(deploy.ChangeSet.Name),
		LogicalResourceId:  aws.String(logicalID),
		ResourceStatus:     status,
		ResourceType:       aws.String("AWS::CloudFormation::TestResource"),
		StackName:          aws.String(deploy.ChangeSet.Stack.Name),
		Timestamp:          aws.Time(time.Now()),
	}
}

type mockEvents []cloudformation.StackEvent

func (ee mockEvents) Paginate(pageSize int) DescribeStackEventsHook {
	offset := 0
	return func(input *cloudformation.DescribeStackEventsInput) (*cloudformation.DescribeStackEventsOutput, error) {
		var out []cloudformation.StackEvent
		for offset < len(ee) {
			e := ee[offset]
			offset++
			if *e.StackName != *input.StackName {
				continue
			}
			out = append(out, e)
			if len(out) == pageSize {
				break
			}
		}
		return &cloudformation.DescribeStackEventsOutput{
			StackEvents: out,
		}, nil
	}
}

// ---

func compareTemplate() cmp.Option {
	return cmpopts.IgnoreUnexported(Template{})
}

type CreateChangeSetHook func(input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error)
type DeleteChangeSetHook func(input *cloudformation.DeleteChangeSetInput) (*cloudformation.DeleteChangeSetOutput, error)
type DescribeStacksHook func(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error)
type DescribeChangeSetHook func(input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error)
type ExecuteChangeSetHook func(input *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error)
type DescribeStackEventsHook func(input *cloudformation.DescribeStackEventsInput) (*cloudformation.DescribeStackEventsOutput, error)

type mockCF struct {
	cloudformationiface.ClientAPI

	// Hooks
	CreateChangeSet     CreateChangeSetHook
	DeleteChangeSet     DeleteChangeSetHook
	DescribeStacks      DescribeStacksHook
	DescribeChangeSet   DescribeChangeSetHook
	ExecuteChangeSet    ExecuteChangeSetHook
	DescribeStackEvents DescribeStackEventsHook
}

func (m *mockCF) req() *aws.Request {
	return &aws.Request{
		HTTPRequest:  &http.Request{URL: &url.URL{}, Header: make(http.Header)},
		HTTPResponse: &http.Response{},
		Retryer:      aws.NoOpRetryer{},
	}
}

func (m *mockCF) CreateChangeSetRequest(input *cloudformation.CreateChangeSetInput) cloudformation.CreateChangeSetRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		r.Data, r.Error = m.CreateChangeSet(input)
	})
	return cloudformation.CreateChangeSetRequest{Request: req, Input: input}
}

func (m *mockCF) DeleteChangeSetRequest(input *cloudformation.DeleteChangeSetInput) cloudformation.DeleteChangeSetRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		r.Data, r.Error = m.DeleteChangeSet(input)
	})
	return cloudformation.DeleteChangeSetRequest{Request: req, Input: input}
}

func (m *mockCF) DescribeStacksRequest(input *cloudformation.DescribeStacksInput) cloudformation.DescribeStacksRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		r.Data, r.Error = m.DescribeStacks(input)
	})
	return cloudformation.DescribeStacksRequest{Request: req, Input: input}
}

func (m *mockCF) DescribeChangeSetRequest(input *cloudformation.DescribeChangeSetInput) cloudformation.DescribeChangeSetRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		r.Data, r.Error = m.DescribeChangeSet(input)
	})
	return cloudformation.DescribeChangeSetRequest{Request: req, Input: input}
}

func (m *mockCF) ExecuteChangeSetRequest(input *cloudformation.ExecuteChangeSetInput) cloudformation.ExecuteChangeSetRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		r.Data, r.Error = m.ExecuteChangeSet(input)
	})
	return cloudformation.ExecuteChangeSetRequest{Request: req, Input: input}
}

func (m *mockCF) DescribeStackEventsRequest(input *cloudformation.DescribeStackEventsInput) cloudformation.DescribeStackEventsRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		r.Data, r.Error = m.DescribeStackEvents(input)
	})
	return cloudformation.DescribeStackEventsRequest{Request: req, Input: input}
}
