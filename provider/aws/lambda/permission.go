// Code generated by awsgen from api version 2015-03-31. DO NOT EDIT.

package lambda

// Permission manages AWS Lambda Permissions.
type Permission struct {
	// The action that the principal can use on the function. For example,
	// lambda:InvokeFunction or lambda:GetFunction.
	Action string `cloudformation:"Action" input:"action"`

	// For Alexa Smart Home functions, a token that must be supplied by the
	// invoker.
	EventSourceToken *string `cloudformation:"EventSourceToken" input:"event_source_token"`

	// The name of the Lambda function, version, or alias.
	//
	// Name formats
	//
	//   Function name - my-function (name-only), my-function:v1 (with alias).
	//   Function ARN - arn:aws:lambda:us-west-2:123456789012:function:my-function.
	//   Partial ARN - 123456789012:function:my-function.
	//
	// You can append a version number or alias to any of the formats. The length
	// constraint applies only to the full ARN. If you specify only the function
	// name, it is limited to 64 characters in length.
	Function string `cloudformation:"FunctionName" input:"function" json:"FunctionName"`

	// The AWS service or account that invokes the function. If you specify a
	// service, use SourceArn or SourceAccount to limit who can invoke the
	// function through that service.
	Principal string `cloudformation:"Principal" input:"principal"`

	// Specify a version or alias to add permissions to a published version of
	// the function.
	Qualifier *string `input:"qualifier"`

	// Only update the policy if the revision ID matches the ID that's
	// specified. Use this option to avoid modifying a policy that has changed
	// since you last read it.
	RevisionID *string `input:"revision_id" json:"RevisionId"`

	// For Amazon S3, the ID of the account that owns the resource. Use this
	// together with SourceArn to ensure that the resource is owned by the
	// specified account. It is possible for an Amazon S3 bucket to be deleted
	// by its owner and recreated by another account.
	SourceAccount *string `cloudformation:"SourceAccount" input:"source_account"`

	// For AWS services, the ARN of the AWS resource that invokes the function.
	// For example, an Amazon S3 bucket or Amazon SNS topic.
	SourceARN *string `cloudformation:"SourceArn" input:"source_arn" json:"SourceArn"`

	// A statement identifier that differentiates the statement from others in
	// the same policy.
	StatementID string `input:"statement_id" json:"StatementId"`

	// Outputs:

	// The permission statement that's added to the function policy.
	Statement *string `output:"statement"`
}

// CloudFormationType returns the CloudFormation type for a AWS Lambda Permission.
func (Permission) CloudFormationType() string { return "AWS::Lambda::Permission" }
