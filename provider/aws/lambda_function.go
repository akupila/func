package aws

import (
	"time"
)

type lambdaDeadLetterConfig struct {
	Target string `input:"target" cloudformation:"TargetArn"`
}

type lambdaEnvironment struct {
	Variables map[string]string `input:"variables" cloudformation:"Variables"`
}

type lambdaTracingConfig struct {
	Mode string `input:"mode" cloudformation:"Mode"`
}

type lambdaVPCConfig struct {
	SecurityGroups []string `input:"security_groups"`
	Subnets        []string `input:"subnets"`
}

// LambdaFunction describes an AWS Lambda Function.
type LambdaFunction struct {
	DeadLetterConfig   *lambdaDeadLetterConfig `input:"dead_letter_config" cloudformation:"DeadLetterConfig"`
	Description        *string                 `input:"description" cloudformation:"Description"`
	Environment        *lambdaEnvironment      `input:"environment" cloudformation:"Environment"`
	Name               *string                 `input:"name" cloudformation:"Name,ref"`
	Handler            string                  `input:"handler" cloudformation:"Handler"`
	KMSKeyArn          *string                 `input:"kms_key_arn" cloudformation:"KmsKeyArn"`
	Layers             []string                `input:"layers" cloudformation:"Layers"`
	MemorySize         *int                    `input:"memory_size" cloudformation:"MemorySize"`
	ReservedExecutions *int                    `input:"reserved_concurrent_executions"`
	Role               string                  `input:"role" cloudformation:"Role"`
	Runtime            string                  `input:"runtime" cloudformation:"Runtime"`
	Tags               Tags                    `input:"tags" cloudformation:"Tags"`
	Timeout            *int                    `input:"timeout" cloudformation:"Timeout"`
	TracingConfig      *lambdaTracingConfig    `input:"tracing_config" cloudformation:"TracingConfig"`
	VPCConfig          *lambdaVPCConfig        `input:"vpc_config" cloudformation:"VpcConfig"`

	Code struct {
		S3Bucket string `cloudformation:"S3Bucket"`
		S3Key    string `cloudformation:"S3Key"`
	} `cloudformation:"Code"`

	CodeSha256   string    `output:"code_sha256"`
	CodeSize     int64     `output:"code_size"`
	ARN          string    `output:"arn,default" cloudformation:"Arn,att"`
	LastModified time.Time `output:"last_modified"`
	MasterARN    *string   `output:"master_arn"`
	RevisionID   *string   `output:"revision_id"`
	Version      *string   `output:"version"`
}

// CloudFormationType returns the AWS CloudFormation type for a Lambda Function.
func (l LambdaFunction) CloudFormationType() string {
	return "AWS::Lambda::Function"
}

// SetS3SourceCode sets the given bucket and key to use as source code for the
// function.
func (l *LambdaFunction) SetS3SourceCode(bucket, key string) {
	l.Code.S3Bucket = bucket
	l.Code.S3Key = key
}
