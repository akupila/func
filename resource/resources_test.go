package resource_test

import "time"

type LambdaDeadLetterConfig struct {
	Target string `input:"target"`
}

type LambdaEnvironment struct {
	Variables map[string]string `input:"variables"`
}

type LambdaTracingConfig struct {
	Mode string `input:"mode"`
}

type LambdaVPCConfig struct {
	SecurityGroups []string `input:"security_groups"`
	Subnets        []string `input:"subnets"`
}

type LambdaFunction struct {
	DeadLetterConfig   *LambdaDeadLetterConfig `input:"dead_letter_config"`
	Description        *string                 `input:"description"`
	Environment        *LambdaEnvironment      `input:"environment"`
	Name               *string                 `input:"name"`
	Handler            string                  `input:"handler"`
	KMSKeyArn          *string                 `input:"kms_key_arn"`
	Layers             []string                `input:"layers"`
	MemorySize         *int                    `input:"memory_size"`
	ReservedExecutions *int                    `input:"reserved_concurrent_executions"`
	Role               string                  `input:"role"`
	Runtime            string                  `input:"runtime"`
	Tags               map[string]string       `input:"tags"`
	Timeout            *int                    `input:"timeout"`
	TracingConfig      *LambdaTracingConfig    `input:"tracing_config"`
	VPCConfig          *LambdaVPCConfig        `input:"vpc_config"`

	CodeSha256   string    `output:"code_sha256"`
	CodeSize     int64     `output:"code_size"`
	ARN          string    `output:"arn,default"`
	LastModified time.Time `output:"last_modified"`
	MasterARN    *string   `output:"master_arn"`
	RevisionID   *string   `output:"revision_id"`
	Version      *string   `output:"version"`
}

type NamedIAMPolicyDocument struct {
	Name       string               `input:"name,label"`
	Version    *string              `input:"version"`
	Statements []IAMPolicyStatement `input:"statement"`
}

type IAMPolicyDocument struct {
	Version    *string              `input:"version"`
	Statements []IAMPolicyStatement `input:"statement"`
}

type IAMPolicyStatement struct {
	ID            *string                      `input:"id"`
	Effect        string                       `input:"effect"`
	Principals    map[string][]string          `input:"principals"`
	NotPrincipals map[string][]string          `input:"not_principals"`
	Actions       []string                     `input:"actions"`
	NotActions    []string                     `input:"not_actions"`
	Resources     []string                     `input:"resources"`
	NotResources  []string                     `input:"not_resources"`
	Conditions    map[string]map[string]string `input:"conditions"`
}

type IAMRole struct {
	AssumeRolePolicy    IAMPolicyDocument        `input:"assume_role_policy"`
	Description         *string                  `input:"description"`
	ManagedPolicies     []string                 `input:"managed_policies"`
	MaxSessionDuration  *time.Duration           `input:"max_session_duration"`
	Path                *string                  `input:"path"`
	PermissionsBoundary *string                  `input:"permissions_boundary"`
	Policies            []NamedIAMPolicyDocument `input:"policy"`
	Name                *string                  `input:"name"`
	Tags                map[string]string        `input:"tags"`

	ARN       *string   `output:"arn"`
	CreatedAt time.Time `output:"created_at"`
	ID        *string   `output:"id"`
}

type MinBlocks struct {
	Nested []struct{} `input:"nested" min:"2"`
}

type MaxBlocks struct {
	Nested []struct{} `input:"nested" max:"2"`
}

type MinMaxBlocks struct {
	Nested []struct{} `input:"nested" min:"2" max:"3"`
}
