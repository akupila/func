// Code generated by awsgen from api version 2010-05-08. DO NOT EDIT.

package iam

import "time"

// Policy manages AWS Identity and Access Management Policies.
type Policy struct {
	// A friendly description of the policy.Typically used to store information
	// about the permissions defined in the policy. For example, "Grants access
	// to production DynamoDB tables."The policy description is immutable.
	// After a value is assigned, it cannot be changed.
	Description *string `input:"description"`

	// The path for the policy.For more information about paths, see IAM
	// Identifiers:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
	// in the IAM User Guide.This parameter is optional. If it is not included,
	// it defaults to a slash (/).This parameter allows (through its regex
	// pattern) a string of characters consisting of either a forward slash (/)
	// by itself or a string that must begin and end with forward slashes. In
	// addition, it can contain any ASCII character from the ! (\u0021) through
	// the DEL character (\u007F), including most punctuation characters,
	// digits, and upper and lowercased letters.
	Path *string `input:"path"`

	// The JSON policy document that you want to use as the content for the new policy.You must provide
	// policies in JSON format in IAM. However, for AWS CloudFormation templates formatted in YAML, you
	// can provide the policy in JSON or YAML format. AWS CloudFormation always converts a YAML policy to
	// JSON format before submitting it to IAM.The regex pattern used to validate this parameter is a
	// string of characters consisting of the following:  Any printable ASCII character ranging from the
	// space character (\u0020) through the end of the ASCII character range
	//   The printable characters in the Basic Latin and Latin-1 Supplement character set (through \u00FF)
	//   The special characters tab (\u0009), line feed (\u000A), and carriage return (\u000D)
	//
	//
	Document string `cloudformation:"PolicyDocument" input:"document" json:"PolicyDocument"`

	// The friendly name of the policy.IAM user, group, role, and policy names
	// must be unique within the account. Names are not distinguished by case.
	// For example, you cannot create resources named both "MyResource" and
	// "myresource".
	Name string `cloudformation:"PolicyName" input:"name" json:"PolicyName"`

	// Outputs:

	ARN *string `json:"Arn" output:"arn"`

	// The number of entities (users, groups, and roles) that the policy is
	// attached to.
	AttachmentCount *int `output:"attachment_count"`

	// The date and time, in ISO 8601 date-time format, when the policy was
	// created.
	CreateDate *time.Time `output:"create_date"`

	// The identifier for the version of the policy that is set as the default
	// version.
	DefaultVersionID *string `json:"DefaultVersionId" output:"default_version_id"`

	// Specifies whether the policy can be attached to an IAM user, group, or
	// role.
	IsAttachable *bool `output:"is_attachable"`

	// The number of entities (users and roles) for which the policy is used to
	// set the permissions boundary. For more information about permissions
	// boundaries, see Permissions Boundaries for IAM Identities :
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_boundaries.html
	// in the IAM User Guide.
	PermissionsBoundaryUsageCount *int `output:"permissions_boundary_usage_count"`

	// The stable and unique string identifying the policy.For more information
	// about IDs, see IAM Identifiers:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
	// in the IAM User Guide.
	PolicyID *string `json:"PolicyId" output:"policy_id"`

	// The date and time, in ISO 8601 date-time format, when the policy was
	// last updated.When a policy has only one version, this field contains the
	// date and time when the policy was created. When a policy has more than
	// one version, this field contains the date and time when the most recent
	// policy version was created.
	UpdateDate *time.Time `output:"update_date"`
}

// CloudFormationType returns the CloudFormation type for a AWS Identity and Access Management Policy.
func (Policy) CloudFormationType() string { return "AWS::IAM::Policy" }
