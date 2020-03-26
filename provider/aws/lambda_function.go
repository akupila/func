package aws

import (
	"time"
)

type LambdaDeadLetterConfig struct {
	Target string `input:"target" cloudformation:"TargetArn"`
}

type LambdaEnvironment struct {
	Variables map[string]string `input:"variables" cloudformation:"Variables"`
}

type LambdaTracingConfig struct {
	Mode string `input:"mode" cloudformation:"Mode"`
}

type LambdaVPCConfig struct {
	SecurityGroups []string `input:"security_groups"`
	Subnets        []string `input:"subnets"`
}

type LambdaFunction struct {
	DeadLetterConfig   *LambdaDeadLetterConfig `input:"dead_letter_config" cloudformation:"DeadLetterConfig"`
	Description        *string                 `input:"description" cloudformation:"Description"`
	Environment        *LambdaEnvironment      `input:"environment" cloudformation:"Environment"`
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
	TracingConfig      *LambdaTracingConfig    `input:"tracing_config" cloudformation:"TracingConfig"`
	VPCConfig          *LambdaVPCConfig        `input:"vpc_config" cloudformation:"VpcConfig"`

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

func (l LambdaFunction) CloudFormationType() string {
	return "AWS::Lambda::Function"
}

func (l *LambdaFunction) SetS3SourceCode(bucket, key string) {
	l.Code.S3Bucket = bucket
	l.Code.S3Key = key
}
