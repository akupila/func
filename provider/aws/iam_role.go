package aws

import (
	"encoding/json"
	"time"
)

type namedIAMPolicyDocument struct {
	Name       string               `input:"name,label"`
	Version    *string              `input:"version"`
	Statements []iamPolicyStatement `input:"statement"`
}

func (p namedIAMPolicyDocument) CloudFormation() (interface{}, error) {
	b, err := marshalPolicy(p.Version, p.Statements)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"PolicyName":     p.Name,
		"PolicyDocument": b,
	}, nil
}

type iamPolicyDocument struct {
	Version    *string              `input:"version"`
	Statements []iamPolicyStatement `input:"statement"`
}

func (p iamPolicyDocument) CloudFormation() (interface{}, error) {
	return marshalPolicy(p.Version, p.Statements)
}

type iamPrincipal struct {
	AWS           []string `input:"aws"`
	Service       []string `input:"service"`
	CanonicalUser *string  `input:"canonical_user"`
	Federated     *string  `input:"federated"`
}

func (p *iamPrincipal) asMap() map[string]interface{} {
	if p == nil {
		return nil
	}
	m := make(map[string]interface{})
	if len(p.AWS) > 0 {
		m["AWS"] = p.AWS
	}
	if len(p.Service) > 0 {
		m["Service"] = p.Service
	}
	if p.CanonicalUser != nil {
		m["CanonicalUser"] = *p.CanonicalUser
	}
	if p.Federated != nil {
		m["Federated"] = *p.Federated
	}
	return m
}

type iamPolicyStatement struct {
	ID           *string                      `input:"id"`
	Effect       string                       `input:"effect"`
	Principal    *iamPrincipal                `input:"principal"`
	NotPrincipal *iamPrincipal                `input:"not_principal"`
	Action       []string                     `input:"action"`
	NotAction    []string                     `input:"not_action"`
	Resource     []string                     `input:"resource"`
	NotResource  []string                     `input:"not_resource"`
	Conditions   map[string]map[string]string `input:"condition"`
}

// IAMRole provides an IAM Role.
type IAMRole struct {
	AssumeRolePolicy    iamPolicyDocument        `input:"assume_role_policy" cloudformation:"AssumeRolePolicyDocument"`
	Description         *string                  `input:"description" cloudformation:"Description"`
	ManagedPolicies     []string                 `input:"managed_policies" cloudformation:"ManagedPolicyArns"`
	MaxSessionDuration  *time.Duration           `input:"max_session_duration" cloudformation:"MaxSessionDuration"`
	Path                *string                  `input:"path" cloudformation:"Path"`
	PermissionsBoundary *string                  `input:"permissions_boundary" cloudformation:"PermissionsBoundary"`
	Policies            []namedIAMPolicyDocument `input:"policy" cloudformation:"Policies"`
	Name                *string                  `input:"name" cloudformation:"RoleName,ref"`
	Tags                Tags                     `input:"tags" cloudformation:"Tags"`

	ARN       *string   `output:"arn" cloudformation:"Arn,att"`
	CreatedAt time.Time `output:"created_at"`
	ID        *string   `output:"id" cloudformation:"RoleId,att"`
}

// CloudFormationType returns the AWS CloudFormation type for an IAM role.
func (IAMRole) CloudFormationType() string {
	return "AWS::IAM::Role"
}

func marshalPolicy(version *string, statements []iamPolicyStatement) (json.RawMessage, error) {
	type statement struct {
		Sid          string                       `json:"Sid,omitempty"`
		Effect       string                       `json:"Effect"`                 // Allow / Deny
		Action       interface{}                  `json:"Action,omitempty"`       // string or []string
		NotAction    interface{}                  `json:"NotAction,omitempty"`    // string or []string
		Principal    map[string]interface{}       `json:"Principal,omitempty"`    // map to string or []string
		NotPrincipal map[string]interface{}       `json:"NotPrincipal,omitempty"` // map to string or []string
		Resource     interface{}                  `json:"Resource,omitempty"`     // string or []string
		NotResource  interface{}                  `json:"NotResource,omitempty"`  // string or []string
		Condition    map[string]map[string]string `json:"Condition,omitempty"`
	}

	type doc struct {
		Version    string      `json:"Version"`
		Statements []statement `json:"Statement"`
	}

	stringOrSlice := func(ss []string) interface{} {
		switch len(ss) {
		case 0:
			return nil
		case 1:
			return ss[0]
		default:
			return ss
		}
	}

	d := doc{Version: "2012-10-17"}
	if version != nil {
		d.Version = *version
	}

	for _, stmt := range statements {
		s := statement{
			Effect: stmt.Effect,
		}
		if stmt.ID != nil {
			s.Sid = *stmt.ID
		}
		if stmt.Action != nil {
			s.Action = stringOrSlice(stmt.Action)
		}
		if stmt.Resource != nil {
			s.Resource = stringOrSlice(stmt.Resource)
		}
		if stmt.NotAction != nil {
			s.NotAction = stringOrSlice(stmt.NotAction)
		}
		if stmt.NotResource != nil {
			s.NotResource = stringOrSlice(stmt.NotResource)
		}
		s.Principal = stmt.Principal.asMap()
		s.NotPrincipal = stmt.NotPrincipal.asMap()
		if stmt.Conditions != nil {
			s.Condition = stmt.Conditions
		}

		d.Statements = append(d.Statements, s)
	}

	return json.Marshal(d)
}
