// Code generated by awsgen from api version 2010-05-08. DO NOT EDIT.

package iam

import "time"

// AccessKey manages AWS Identity and Access Management AccessKeys.
type AccessKey struct {
	// The name of the IAM user that the new key will belong to.This parameter
	// allows (through its regex pattern) a string of characters consisting of
	// upper and lowercase alphanumeric characters with no spaces. You can also
	// include any of the following characters: _+=,.@-
	UserName *string `cloudformation:"UserName" input:"user_name"`

	// Outputs:

	// A structure with details about the access key.
	AccessKey struct {
		// The ID for this access key.
		AccessKeyID string `json:"AccessKeyId" output:"access_key_id"`

		// The date when the access key was created.
		CreateDate *time.Time `output:"create_date"`

		// The secret key used to sign requests.
		SecretAccessKey string `output:"secret_access_key"`

		// The status of the access key. Active means that the key is valid for API
		// calls, while Inactive means it is not.
		Status string `output:"status"`

		// The name of the IAM user that the access key is associated with.
		UserName string `output:"user_name"`
	} `output:"access_key"`
}

// CloudFormationType returns the CloudFormation type for a AWS Identity and Access Management AccessKey.
func (AccessKey) CloudFormationType() string { return "AWS::IAM::AccessKey" }