// Code generated by awsgen from api version 2010-05-08. DO NOT EDIT.

package iam

import "time"

// ServiceLinkedRole manages AWS Identity and Access Management
// ServiceLinkedRoles.
type ServiceLinkedRole struct {
	// The service principal for the AWS service to which this role is
	// attached. You use a string similar to a URL but without the http:// in
	// front. For example: elasticbeanstalk.amazonaws.com. Service principals
	// are unique and case-sensitive. To find the exact service principal for
	// your service-linked role, see AWS Services That Work with IAM:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_aws-services-that-work-with-iam.html
	// in the IAM User Guide. Look for the services that have Yes in the
	// Service-Linked Role column. Choose the Yes link to view the
	// service-linked role documentation for that service.
	AWSServiceName string `cloudformation:"AWSServiceName" input:"aws_service_name"`

	//
	// A string that you provide, which is combined with the service-provided
	// prefix to form the complete role name. If you make multiple requests for
	// the same service, then you must supply a different CustomSuffix for each
	// request. Otherwise the request fails with a duplicate role name error.
	// For example, you could add -1 or -debug to the suffix.Some services do
	// not support the CustomSuffix parameter. If you provide an optional
	// suffix and the operation fails, try the operation again without the
	// suffix.
	CustomSuffix *string `cloudformation:"CustomSuffix" input:"custom_suffix"`

	// The description of the role.
	Description *string `cloudformation:"Description" input:"description"`

	// Outputs:

	// A Role object that contains details about the newly created role.
	Role *struct {
		//  The Amazon Resource Name (ARN) specifying the role. For more
		// information about ARNs and how to use them in policies, see IAM
		// Identifiers:
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
		// in the IAM User Guide guide.
		ARN string `json:"Arn" output:"arn"`

		// The policy that grants an entity permission to assume the role.
		AssumeRolePolicyDocument *string `output:"assume_role_policy_document"`

		// The date and time, in ISO 8601 date-time format, when the role was
		// created.
		CreateDate time.Time `output:"create_date"`

		// A description of the role that you provide.
		Description *string `output:"description"`

		// The maximum session duration (in seconds) for the specified role. Anyone
		// who uses the AWS CLI, or API to assume the role can specify the duration
		// using the optional DurationSeconds API parameter or duration-seconds CLI
		// parameter.
		MaxSessionDuration *int `output:"max_session_duration"`

		//  The path to the role. For more information about paths, see IAM
		// Identifiers:
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
		// in the IAM User Guide.
		Path string `output:"path"`

		// The ARN of the policy used to set the permissions boundary for the
		// role.For more information about permissions boundaries, see Permissions
		// Boundaries for IAM Identities :
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_boundaries.html
		// in the IAM User Guide.
		PermissionsBoundary *struct {
			//  The ARN of the policy used to set the permissions boundary for the user
			// or role.
			PermissionsBoundaryARN *string `json:"PermissionsBoundaryArn" output:"permissions_boundary_arn"`

			//  The permissions boundary usage type that indicates what type of IAM
			// resource is used as the permissions boundary for an entity. This data
			// type can only have a value of Policy.
			PermissionsBoundaryType *string `output:"permissions_boundary_type"`
		} `output:"permissions_boundary"`

		//  The stable and unique string identifying the role. For more information
		// about IDs, see IAM Identifiers:
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
		// in the IAM User Guide.
		RoleID string `json:"RoleId" output:"role_id"`

		// Contains information about the last time that an IAM role was used. This
		// includes the date and time and the Region in which the role was last
		// used. Activity is only reported for the trailing 400 days. This period
		// can be shorter if your Region began supporting these features within the
		// last year. The role might have been used more than 400 days ago. For
		// more information, see Regions Where Data Is Tracked:
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_access-advisor.html#access-advisor_tracking-period
		// in the IAM User Guide.
		RoleLastUsed *struct {
			// The date and time, in ISO 8601 date-time format that the role was last
			// used.This field is null if the role has not been used within the IAM
			// tracking period. For more information about the tracking period, see
			// Regions Where Data Is Tracked:
			// https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_access-advisor.html#access-advisor_tracking-period
			// in the IAM User Guide.
			LastUsedDate *time.Time `output:"last_used_date"`

			// The name of the AWS Region in which the role was last used.
			Region *string `output:"region"`
		} `output:"role_last_used"`

		// The friendly name that identifies the role.
		RoleName string `output:"role_name"`

		// A list of tags that are attached to the specified role. For more
		// information about tagging, see Tagging IAM Identities:
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_tags.html in the IAM
		// User Guide.
		Tags []struct {
			// The key name that can be used to look up or retrieve the associated
			// value. For example, Department or Cost Center are common choices.
			Key string `output:"key"`

			// The value associated with this tag. For example, tags with a key name of
			// Department could have values such as Human Resources, Accounting, and
			// Support. Tags with a key name of Cost Center might have values that
			// consist of the number associated with the different cost centers in your
			// company. Typically, many resources have tags with the same key name but
			// with different values.AWS always interprets the tag Value as a single
			// string. If you need to store an array, you can store comma-separated
			// values in the string. However, you must interpret the value in your
			// code.
			Value string `output:"value"`
		} `output:"tags"`
	} `output:"role"`
}

// CloudFormationType returns the CloudFormation type for a AWS Identity and Access Management ServiceLinkedRole.
func (ServiceLinkedRole) CloudFormationType() string { return "AWS::IAM::ServiceLinkedRole" }