package cloudformation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/google/go-cmp/cmp"
)

func TestChangeSet_loadChanges_wait(t *testing.T) {
	attempts := 0
	hook := func(input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error) {
		attempts++
		switch attempts {
		case 1:
			return &cloudformation.DescribeChangeSetOutput{
				Status: cloudformation.ChangeSetStatusCreatePending,
			}, nil
		case 2:
			return &cloudformation.DescribeChangeSetOutput{
				Status: cloudformation.ChangeSetStatusCreateInProgress,
			}, nil
		case 3:
			return &cloudformation.DescribeChangeSetOutput{
				Status: cloudformation.ChangeSetStatusCreateComplete,
				Changes: []cloudformation.Change{
					makeChange(cloudformation.ChangeActionAdd, "Foo"),
				},
			}, nil
		default:
			panic(fmt.Sprintf("Unhandled attempts: %d", attempts))
		}
	}

	api := &mockCF{DescribeChangeSet: hook}
	cs := &ChangeSet{
		Stack: &Stack{Name: "test"},
	}
	err := cs.loadChanges(context.Background(), api, 0*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	want := []Change{
		{Operation: ResourceCreate, LogicalID: "Foo"},
	}
	if diff := cmp.Diff(cs.Changes, want); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}
