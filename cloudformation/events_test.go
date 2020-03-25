package cloudformation

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func TestParseResourceOp(t *testing.T) {
	tests := []struct {
		status cloudformation.ResourceStatus
		want   ResourceOperation
	}{
		{cloudformation.ResourceStatusCreateInProgress, ResourceCreate},
		{cloudformation.ResourceStatusCreateFailed, ResourceCreate},
		{cloudformation.ResourceStatusCreateComplete, ResourceCreate},
		{cloudformation.ResourceStatusDeleteInProgress, ResourceDelete},
		{cloudformation.ResourceStatusDeleteFailed, ResourceDelete},
		{cloudformation.ResourceStatusDeleteComplete, ResourceDelete},
		{cloudformation.ResourceStatusDeleteSkipped, ResourceDelete},
		{cloudformation.ResourceStatusUpdateInProgress, ResourceUpdate},
		{cloudformation.ResourceStatusUpdateFailed, ResourceUpdate},
		{cloudformation.ResourceStatusUpdateComplete, ResourceUpdate},
		{cloudformation.ResourceStatusImportFailed, ResourceImport},
		{cloudformation.ResourceStatusImportComplete, ResourceImport},
		{cloudformation.ResourceStatusImportInProgress, ResourceImport},
		{cloudformation.ResourceStatusImportRollbackInProgress, ResourceImport},
		{cloudformation.ResourceStatusImportRollbackFailed, ResourceImport},
		{cloudformation.ResourceStatusImportRollbackComplete, ResourceImport},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			got := parseResourceOp(tc.status)
			if got != tc.want {
				t.Errorf("Got = %s, want = %s", got, tc.want)
			}
		})
	}
}

func TestParseStackOp(t *testing.T) {
	tests := []struct {
		status cloudformation.StackStatus
		want   StackOperation
	}{
		{cloudformation.StackStatusCreateInProgress, StackCreate},
		{cloudformation.StackStatusCreateFailed, StackCreate},
		{cloudformation.StackStatusCreateComplete, StackCreate},
		{cloudformation.StackStatusRollbackInProgress, StackRollback},
		{cloudformation.StackStatusRollbackFailed, StackRollback},
		{cloudformation.StackStatusRollbackComplete, StackRollback},
		{cloudformation.StackStatusDeleteInProgress, StackDelete},
		{cloudformation.StackStatusDeleteFailed, StackDelete},
		{cloudformation.StackStatusDeleteComplete, StackDelete},
		{cloudformation.StackStatusUpdateInProgress, StackUpdate},
		{cloudformation.StackStatusUpdateCompleteCleanupInProgress, StackUpdate},
		{cloudformation.StackStatusUpdateComplete, StackUpdate},
		{cloudformation.StackStatusUpdateRollbackInProgress, StackRollback},
		{cloudformation.StackStatusUpdateRollbackFailed, StackRollback},
		{cloudformation.StackStatusUpdateRollbackCompleteCleanupInProgress, StackRollback},
		{cloudformation.StackStatusUpdateRollbackComplete, StackRollback},
		{cloudformation.StackStatusReviewInProgress, StackReview},
		{cloudformation.StackStatusImportInProgress, StackImport},
		{cloudformation.StackStatusImportComplete, StackImport},
		{cloudformation.StackStatusImportRollbackInProgress, StackRollback},
		{cloudformation.StackStatusImportRollbackFailed, StackRollback},
		{cloudformation.StackStatusImportRollbackComplete, StackRollback},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			got := parseStackOp(tc.status)
			if got != tc.want {
				t.Errorf("Got = %s, want = %s", got, tc.want)
			}
		})
	}
}

func TestParseState(t *testing.T) {
	tests := []struct {
		status cloudformation.ResourceStatus
		want   State
	}{
		{cloudformation.ResourceStatusCreateInProgress, StateInProgress},
		{cloudformation.ResourceStatusCreateFailed, StateFailed},
		{cloudformation.ResourceStatusCreateComplete, StateComplete},
		{cloudformation.ResourceStatusDeleteInProgress, StateInProgress},
		{cloudformation.ResourceStatusDeleteFailed, StateFailed},
		{cloudformation.ResourceStatusDeleteComplete, StateComplete},
		{cloudformation.ResourceStatusDeleteSkipped, StateSkipped},
		{cloudformation.ResourceStatusUpdateInProgress, StateInProgress},
		{cloudformation.ResourceStatusUpdateFailed, StateFailed},
		{cloudformation.ResourceStatusUpdateComplete, StateComplete},
		{cloudformation.ResourceStatusImportFailed, StateFailed},
		{cloudformation.ResourceStatusImportComplete, StateComplete},
		{cloudformation.ResourceStatusImportInProgress, StateInProgress},
		{cloudformation.ResourceStatusImportRollbackInProgress, StateInProgress},
		{cloudformation.ResourceStatusImportRollbackFailed, StateFailed},
		{cloudformation.ResourceStatusImportRollbackComplete, StateComplete},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			got := parseState(tc.status)
			if got != tc.want {
				t.Errorf("Got = %s, want = %s", got, tc.want)
			}
		})
	}
}
