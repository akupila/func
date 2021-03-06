// Code generated by awsgen from api version 2015-03-31. DO NOT EDIT.

package lambda

// Function manages AWS Lambda Functions.
type Function struct {
	// The code for the function.
	Code struct {
		// An Amazon S3 bucket in the same AWS Region as your function. The bucket
		// can be in a different AWS account.
		S3Bucket *string `cloudformation:"S3Bucket" input:"s3_bucket"`

		// The Amazon S3 key of the deployment package.
		S3Key *string `cloudformation:"S3Key" input:"s3_key"`

		// For versioned objects, the version of the deployment package object to
		// use.
		S3ObjectVersion *string `cloudformation:"S3ObjectVersion" input:"s3_object_version"`

		// The base64-encoded contents of the deployment package. AWS SDK and AWS
		// CLI clients handle the encoding for you.
		ZipFile []byte `cloudformation:"ZipFile" input:"zip_file"`
	} `cloudformation:"Code"`

	// A dead letter queue configuration that specifies the queue or topic
	// where Lambda sends asynchronous events when they fail processing. For
	// more information, see Dead Letter Queues:
	// https://docs.aws.amazon.com/lambda/latest/dg/invocation-async.html#dlq.
	DeadLetter *struct {
		// The Amazon Resource Name (ARN) of an Amazon SQS queue or Amazon SNS
		// topic.
		ARN *string `cloudformation:"TargetArn" input:"arn" json:"TargetArn"`
	} `cloudformation:"DeadLetterConfig" input:"dead_letter" json:"DeadLetterConfig"`

	// A description of the function.
	Description *string `cloudformation:"Description" input:"description"`

	// Environment variables that are accessible from function code during
	// execution.
	Environment *struct {
		// Environment variable key-value pairs.
		Variables map[string]string `cloudformation:"Variables" input:"variables"`
	} `cloudformation:"Environment" input:"environment"`

	// The name of the Lambda function.
	//
	// Name formats
	//
	//   Function name - my-function.
	//   Function ARN - arn:aws:lambda:us-west-2:123456789012:function:my-function.
	//   Partial ARN - 123456789012:function:my-function.
	//
	// The length constraint applies only to the full ARN. If you specify only the
	// function name, it is limited to 64 characters in length.
	Name string `cloudformation:"FunctionName" input:"name" json:"FunctionName"`

	// The name of the method within your code that Lambda calls to execute
	// your function. The format includes the file name. It can also include
	// namespaces and other qualifiers, depending on the runtime. For more
	// information, see Programming Model:
	// https://docs.aws.amazon.com/lambda/latest/dg/programming-model-v2.html.
	Handler string `cloudformation:"Handler" input:"handler"`

	// The ARN of the AWS Key Management Service (AWS KMS) key that's used to
	// encrypt your function's environment variables. If it's not provided, AWS
	// Lambda uses a default service key.
	KMSKeyARN *string `cloudformation:"KmsKeyArn" input:"kms_key_arn" json:"KMSKeyArn"`

	// A list of function layers to add to the function's execution
	// environment. Specify each layer by its ARN, including the version.
	Layers []string `cloudformation:"Layers" input:"layers"`

	// The amount of memory that your function has access to. Increasing the
	// function's memory also increases its CPU allocation. The default value
	// is 128 MB. The value must be a multiple of 64 MB.
	MemorySize *int `cloudformation:"MemorySize" input:"memory_size"`

	// Set to true to publish the first version of the function during
	// creation.
	Publish *bool `input:"publish"`

	// The Amazon Resource Name (ARN) of the function's execution role.
	Role string `cloudformation:"Role" input:"role"`

	// The identifier of the function's runtime.
	Runtime string `cloudformation:"Runtime" input:"runtime"`

	// A list of tags to apply to the function.
	Tags map[string]string `cloudformation:"Tags" input:"tags"`

	// The amount of time that Lambda allows a function to run before stopping
	// it. The default is 3 seconds. The maximum allowed value is 900 seconds.
	Timeout *int `cloudformation:"Timeout" input:"timeout"`

	// Set Mode to Active to sample and trace a subset of incoming requests
	// with AWS X-Ray.
	Tracing *struct {
		// The tracing mode.
		Mode *string `cloudformation:"Mode" input:"mode"`
	} `cloudformation:"TracingConfig" input:"tracing" json:"TracingConfig"`

	// For network connectivity to AWS resources in a VPC, specify a list of
	// security groups and subnets in the VPC. When you connect a function to a
	// VPC, it can only access resources and the internet through that VPC. For
	// more information, see VPC Settings:
	// https://docs.aws.amazon.com/lambda/latest/dg/configuration-vpc.html.
	VPC *struct {
		// A list of VPC security groups IDs.
		SecurityGroups []string `cloudformation:"SecurityGroupIds" input:"security_groups" json:"SecurityGroupIds"`

		// A list of VPC subnet IDs.
		Subnets []string `cloudformation:"SubnetIds" input:"subnets" json:"SubnetIds"`
	} `cloudformation:"VpcConfig" input:"vpc" json:"VpcConfig"`

	// Outputs:

	// The SHA256 hash of the function's deployment package.
	CodeSha256 *string `output:"code_sha256"`

	// The size of the function's deployment package, in bytes.
	CodeSize *int `output:"code_size"`

	// The function's Amazon Resource Name (ARN).
	ARN *string `cloudformation:"Arn,att" json:"FunctionArn" output:"arn"`

	// The date and time that the function was last updated, in ISO-8601 format
	// (YYYY-MM-DDThh:mm:ss.sTZD).
	LastModified *string `output:"last_modified"`

	// The status of the last update that was performed on the function. This
	// is first set to Successful after function creation completes.
	LastUpdateStatus *string `output:"last_update_status"`

	// The reason for the last update that was performed on the function.
	LastUpdateStatusReason *string `output:"last_update_status_reason"`

	// The reason code for the last update that was performed on the function.
	LastUpdateStatusReasonCode *string `output:"last_update_status_reason_code"`

	// For Lambda@Edge functions, the ARN of the master function.
	MasterARN *string `json:"MasterArn" output:"master_arn"`

	// The latest updated revision of the function or alias.
	RevisionID *string `json:"RevisionId" output:"revision_id"`

	// The current state of the function. When the state is Inactive, you can
	// reactivate the function by invoking it.
	State *string `output:"state"`

	// The reason for the function's current state.
	StateReason *string `output:"state_reason"`

	// The reason code for the function's current state. When the code is
	// Creating, you can't invoke or modify the function.
	StateReasonCode *string `output:"state_reason_code"`

	// The version of the Lambda function.
	Version *string `output:"version"`
}

// CloudFormationType returns the CloudFormation type for a AWS Lambda Function.
func (Function) CloudFormationType() string { return "AWS::Lambda::Function" }

// SetS3SourceCode sets the given bucket and key to use as source code for the
// function.
func (f *Function) SetS3SourceCode(bucket, key string) {
	f.Code.S3Bucket = &bucket
	f.Code.S3Key = &key
}
