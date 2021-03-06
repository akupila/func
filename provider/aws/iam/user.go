// Code generated by awsgen from api version 2010-05-08. DO NOT EDIT.

package iam

import "time"

// User manages AWS Identity and Access Management Users.
type User struct {
	//  The path for the user name. For more information about paths, see IAM
	// Identifiers:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
	// in the IAM User Guide.This parameter is optional. If it is not included,
	// it defaults to a slash (/).This parameter allows (through its regex
	// pattern) a string of characters consisting of either a forward slash (/)
	// by itself or a string that must begin and end with forward slashes. In
	// addition, it can contain any ASCII character from the ! (\u0021) through
	// the DEL character (\u007F), including most punctuation characters,
	// digits, and upper and lowercased letters.
	Path *string `cloudformation:"Path" input:"path"`

	// The ARN of the policy that is used to set the permissions boundary for
	// the user.
	PermissionsBoundary *string `cloudformation:"PermissionsBoundary" input:"permissions_boundary"`

	// A list of tags that you want to attach to the newly created user. Each
	// tag consists of a key name and an associated value. For more information
	// about tagging, see Tagging IAM Identities:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_tags.html in the IAM
	// User Guide.If any one of the tags is invalid or if you exceed the
	// allowed number of tags per user, then the entire request fails and the
	// user is not created.
	Tags []struct {
		// The key name that can be used to look up or retrieve the associated
		// value. For example, Department or Cost Center are common choices.
		Key string `input:"key"`

		// The value associated with this tag. For example, tags with a key name of
		// Department could have values such as Human Resources, Accounting, and
		// Support. Tags with a key name of Cost Center might have values that
		// consist of the number associated with the different cost centers in your
		// company. Typically, many resources have tags with the same key name but
		// with different values.AWS always interprets the tag Value as a single
		// string. If you need to store an array, you can store comma-separated
		// values in the string. However, you must interpret the value in your
		// code.
		Value string `input:"value"`
	} `cloudformation:"Tags" input:"tags"`

	// The name of the user to create.IAM user, group, role, and policy names
	// must be unique within the account. Names are not distinguished by case.
	// For example, you cannot create resources named both "MyResource" and
	// "myresource".
	UserName string `cloudformation:"UserName" input:"user_name"`

	// Outputs:

	// A structure with details about the new IAM user.
	User *struct {
		// The Amazon Resource Name (ARN) that identifies the user. For more
		// information about ARNs and how to use ARNs in policies, see IAM
		// Identifiers:
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
		// in the IAM User Guide.
		ARN string `json:"Arn" output:"arn"`

		// The date and time, in ISO 8601 date-time format, when the user was
		// created.
		CreateDate time.Time `output:"create_date"`

		// The date and time, in ISO 8601 date-time format, when the user's password was last used to sign in to an
		// AWS website. For a list of AWS websites that capture a user's last sign-in time, see the Credential
		// Reports topic in the IAM User Guide. If a password is used more than once in a five-minute span, only the
		// first use is returned in this field. If the field is null (no value), then it indicates that they never
		// signed in with a password. This can be because:  The user never had a password.
		//   A password exists but has not been used since IAM started tracking this information on October 20, 2014.
		//
		// A null value does not mean that the user never had a password. Also, if the user does not currently have a
		// password but had one in the past, then this field contains the date and time the most recent password was
		// used.This value is returned only in the GetUser and ListUsers operations.
		PasswordLastUsed *time.Time `output:"password_last_used"`

		// The path to the user. For more information about paths, see IAM
		// Identifiers:
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
		// in the IAM User Guide.
		Path string `output:"path"`

		// The ARN of the policy used to set the permissions boundary for the
		// user.For more information about permissions boundaries, see Permissions
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

		// A list of tags that are associated with the specified user. For more
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

		// The stable and unique string identifying the user. For more information
		// about IDs, see IAM Identifiers:
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
		// in the IAM User Guide.
		UserID string `json:"UserId" output:"user_id"`

		// The friendly name identifying the user.
		UserName string `output:"user_name"`
	} `output:"user"`
}

// CloudFormationType returns the CloudFormation type for a AWS Identity and Access Management User.
func (User) CloudFormationType() string { return "AWS::IAM::User" }
