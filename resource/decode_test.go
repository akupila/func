package resource_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestDecoder_Decode(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		input     string
		want      *resource.Graph
		wantDiags hcl.Diagnostics
	}{
		// Attributes
		{
			name:     "Attributes",
			filename: "file.hcl",
			input: `
resource "func" {
	type        = "aws:lambda_function"
	handler     = "index.handler"
	runtime     = "nodejs10.x"
	role        = "testrole"
	memory_size = 256
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"func": {
						Type: "aws:lambda_function",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
						},
						Config: LambdaFunction{
							Handler:    "index.handler",
							Runtime:    "nodejs10.x",
							Role:       "testrole",
							MemorySize: intptr(256),
						},
					},
				},
			},
		},

		// Blocks
		{
			name:     "Block",
			filename: "file.hcl",
			input: `
resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"

	environment {
		variables = {
			"foo" = "bar"
		}
	}
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"func": {
						Type: "aws:lambda_function",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
						},
						Config: LambdaFunction{
							Handler: "index.handler",
							Runtime: "nodejs10.x",
							Role:    "testrole",
							Environment: &LambdaEnvironment{
								Variables: map[string]string{
									"foo": "bar",
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "BlockMapList",
			filename: "file.hcl",
			input: `
resource "role" {
	type = "aws:iam_role"

	assume_role_policy {
		statement {
			effect     = "Allow"
			actions    = ["sts:AssumeRole"]
			principals = {
				"Service" = ["lambda.amazonaws.com"]
			}
		}
	}

	policy "Logs" {
		statement {
			effect    = "Allow"
			actions   = ["logs:*"]
			resources = ["*"]
		}
	}

	policy "DynamoDB" {
		statement {
			effect    = "Allow"
			actions   = ["dynamodb:GetItem"]
			resources = ["table1"]
		}
		statement {
			effect    = "Allow"
			actions   = ["dynamodb:PutItem"]
			resources = ["table2"]
		}
	}
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"role": {
						Type: "aws:iam_role",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
						},
						Config: IAMRole{
							AssumeRolePolicy: IAMPolicyDocument{
								Statements: []IAMPolicyStatement{{
									Effect:  "Allow",
									Actions: []string{"sts:AssumeRole"},
									Principals: map[string][]string{
										"Service": []string{"lambda.amazonaws.com"},
									},
								}},
							},
							Policies: []NamedIAMPolicyDocument{
								{
									Name: "Logs",
									Statements: []IAMPolicyStatement{{
										Effect:    "Allow",
										Actions:   []string{"logs:*"},
										Resources: []string{"*"},
									}},
								},
								{
									Name: "DynamoDB",
									Statements: []IAMPolicyStatement{{
										Effect:    "Allow",
										Actions:   []string{"dynamodb:GetItem"},
										Resources: []string{"table1"},
									}, {
										Effect:    "Allow",
										Actions:   []string{"dynamodb:PutItem"},
										Resources: []string{"table2"},
									}},
								},
							},
						},
					},
				},
			},
		},

		// Source code
		{
			name:     "Source",
			filename: "file.hcl",
			input: `
resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"

	source {
		dir = "./src"
	}
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"func": {
						Type: "aws:lambda_function",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
						},
						Config: LambdaFunction{
							Handler: "index.handler",
							Runtime: "nodejs10.x",
							Role:    "testrole",
						},
						SourceCode: &resource.SourceCode{
							Dir: "src",
						},
					},
				},
			},
		},
		{
			name:     "SourceNested",
			filename: "a/b/c/foo.hcl",
			input: `
resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"

	source {
		dir = "source"
	}
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"func": {
						Type: "aws:lambda_function",
						Definition: hcl.Range{
							Filename: "a/b/c/foo.hcl",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
						},
						Config: LambdaFunction{
							Handler: "index.handler",
							Runtime: "nodejs10.x",
							Role:    "testrole",
						},
						SourceCode: &resource.SourceCode{
							Dir: "a/b/c/source",
						},
					},
				},
			},
		},

		// References
		{
			name:     "ReferenceToInput",
			filename: "file.hcl",
			input: `
resource "a" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"
}

resource "b" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = a.role
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"a": {
						Type: "aws:lambda_function",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 13, Byte: 13},
						},
						Config: LambdaFunction{
							Handler: "index.handler",
							Runtime: "nodejs10.x",
							Role:    "testrole",
						},
					},
					"b": {
						Type: "aws:lambda_function",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 9, Column: 1, Byte: 125},
							End:      hcl.Pos{Line: 9, Column: 13, Byte: 137},
						},
						Config: LambdaFunction{
							Handler: "index.handler",
							Runtime: "nodejs10.x",
							Role:    "",
						},
						Refs: []resource.Reference{
							{
								Field: cty.GetAttrPath("role"),
								Expression: &hclsyntax.ScopeTraversalExpr{
									Traversal: hcl.Traversal{
										hcl.TraverseRoot{Name: "a"},
										hcl.TraverseAttr{Name: "role"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "ReferenceToOutput",
			filename: "file.hcl",
			input: `
resource "role" {
	type = "aws:iam_role"

	assume_role_policy {
		statement {
			effect     = "Allow"
			actions    = ["sts:AssumeRole"]
			principals = {
				"Service" = ["lambda.amazonaws.com"]
			}
		}
	}

	policy "Logs" {
		statement {
			effect    = "Allow"
			actions   = ["logs:*"]
			resources = ["*"]
		}
	}
}

resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = role.arn
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"role": {
						Type: "aws:iam_role",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
						},
						Config: IAMRole{
							AssumeRolePolicy: IAMPolicyDocument{
								Statements: []IAMPolicyStatement{{
									Effect:  "Allow",
									Actions: []string{"sts:AssumeRole"},
									Principals: map[string][]string{
										"Service": []string{"lambda.amazonaws.com"},
									},
								}},
							},
							Policies: []NamedIAMPolicyDocument{{
								Name: "Logs",
								Statements: []IAMPolicyStatement{{
									Effect:    "Allow",
									Actions:   []string{"logs:*"},
									Resources: []string{"*"},
								}},
							}},
						},
					},
					"func": {
						Type: "aws:lambda_function",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 24, Column: 1, Byte: 321},
							End:      hcl.Pos{Line: 24, Column: 16, Byte: 336},
						},
						Config: LambdaFunction{
							Handler: "index.handler",
							Runtime: "nodejs10.x",
							Role:    "",
						},
						Refs: []resource.Reference{
							{
								Field: cty.GetAttrPath("role"),
								Expression: &hclsyntax.ScopeTraversalExpr{
									Traversal: hcl.Traversal{
										hcl.TraverseRoot{Name: "role"},
										hcl.TraverseAttr{Name: "arn"},
									},
								},
							},
						},
					},
				},
			},
		},

		// Conversion
		{
			name:     "ConvertNumberToString",
			filename: "file.hcl",
			input: `
resource "func" {
	type        = "aws:lambda_function"
	handler     = "index.handler"
	runtime     = "nodejs10.x"
	role        = "testrole"
	description = 12345
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"func": {
						Type: "aws:lambda_function",
						Definition: hcl.Range{
							Filename: "file.hcl",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
							End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
						},
						Config: LambdaFunction{
							Handler:     "index.handler",
							Runtime:     "nodejs10.x",
							Role:        "testrole",
							Description: strptr("12345"),
						},
					},
				},
			},
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagWarning,
				Summary:  "Value is converted from number to string",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 7, Column: 16, Byte: 156},
					End:      hcl.Pos{Line: 7, Column: 21, Byte: 161},
				},
				Expression: &hclsyntax.LiteralValueExpr{
					Val: cty.NumberIntVal(12345),
				},
			}},
		},

		// Errors
		{
			name:     "ErrTypeMissing",
			filename: "file.hcl",
			input: `
resource "err" {
	# No type
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   "The argument \"type\" is required, but no definition was found.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 16, Byte: 16},
					End:      hcl.Pos{Line: 2, Column: 16, Byte: 16},
				},
			}},
		},
		{
			name:     "ErrTypeVariable",
			filename: "file.hcl",
			input: `
resource "err" {
	type = foo
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Variables not allowed",
				Detail:   "Variables may not be used here.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 9, Byte: 26},
					End:      hcl.Pos{Line: 3, Column: 12, Byte: 29},
				},
				Expression: &hclsyntax.ScopeTraversalExpr{
					Traversal: hcl.Traversal{
						hcl.TraverseRoot{Name: "foo"},
					},
				},
			}},
		},
		{
			name:     "ErrTypeNotFound",
			filename: "file.hcl",
			input: `
resource "err" {
	type = "invalid"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported resource",
				Detail:   "Resources of type \"invalid\" are not supported.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 9, Byte: 26},
					End:      hcl.Pos{Line: 3, Column: 18, Byte: 35},
				},
			}},
		},
		{
			name:     "ErrTypeNotFoundSuggest",
			filename: "file.hcl",
			input: `
resource "err" {
	type = "aws/lambda-function"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported resource",
				Detail:   "Resources of type \"aws/lambda-function\" are not supported. Did you mean \"aws:lambda_function\"?",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 3, Column: 9, Byte: 26},
					End:      hcl.Pos{Line: 3, Column: 30, Byte: 47},
				},
			}},
		},
		{
			name:     "ErrUnsupportedBlock",
			filename: "file.hcl",
			input: `
xxx {
	type = ""
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported block type",
				Detail:   "Blocks of type \"xxx\" are not expected here.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 1, Byte: 1},
					End:      hcl.Pos{Line: 2, Column: 4, Byte: 4},
				},
			}},
		},
		{
			name:     "ErrEmptyName",
			filename: "file.hcl",
			input: `
resource "" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "foo"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Resource name not set",
				Detail:   "A resource name cannot be blank.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 10, Byte: 10},
					End:      hcl.Pos{Line: 2, Column: 12, Byte: 12},
				},
			}},
		},
		{
			name:     "ErrDuplicateResourceName",
			filename: "file.hcl",
			input: `
resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "foo"
}

resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler2"
	runtime = "go1.x"
	role    = "bar"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Duplicate resource",
				Detail:   "Another resource named \"func\" was defined in file.hcl on line 2.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 9, Column: 1, Byte: 123},
					End:      hcl.Pos{Line: 9, Column: 16, Byte: 138},
				},
			}},
		},
		{
			name:     "ErrRequiredAttributeNotSet",
			filename: "file.hcl",
			input: `
resource "func" {
	type = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   "The argument \"role\" is required, but no definition was found.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 17, Byte: 17},
					End:      hcl.Pos{Line: 2, Column: 17, Byte: 17},
				},
			}},
		},
		{
			name:     "ErrRequiredBlockNotSet",
			filename: "file.hcl",
			input: `
resource "role" {
	type = "aws:iam_role"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing assume_role_policy block",
				Detail:   "A block of type \"assume_role_policy\" is required here.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 17, Byte: 17},
					End:      hcl.Pos{Line: 2, Column: 17, Byte: 17},
				},
			}},
		},
		{
			name:     "ErrTooManyBlocks", // Not targeting slice of structs
			filename: "file.hcl",
			input: `
resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "foo"

	environment {
		# 1
	}

	environment {
		# 2
	}
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Duplicate environment block",
				Detail:   "Only one block of type \"environment\" is allowed. Previous definition was at file.hcl:8,2-13.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 12, Column: 2, Byte: 147},
					End:      hcl.Pos{Line: 12, Column: 13, Byte: 158},
				},
			}},
		},
		{
			name:     "ErrTooFewBlocks",
			filename: "file.hcl",
			input: `
resource "func" {
	type = "min_blocks"

	nested {
	}
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Insufficient nested blocks",
				Detail:   "At least 2 \"nested\" blocks are required.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 17, Byte: 17},
					End:      hcl.Pos{Line: 2, Column: 17, Byte: 17},
				},
			}},
		},
		{
			name:     "ErrTooManyBlocksSlice",
			filename: "file.hcl",
			input: `
resource "func" {
	type = "max_blocks"

	nested {
	}
	nested {
	}
	nested {
	}
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Too many nested blocks",
				Detail:   "No more than 2 \"nested\" blocks are allowed",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 9, Column: 9, Byte: 75},
					End:      hcl.Pos{Line: 9, Column: 9, Byte: 75},
				},
			}},
		},
		{
			name:     "ErrCountBlocksTooFew",
			filename: "file.hcl",
			input: `
resource "a" {
	type = "min_max_blocks"

	nested {
	}
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Insufficient nested blocks",
				Detail:   "At least 2 \"nested\" blocks are required.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 14, Byte: 14},
					End:      hcl.Pos{Line: 2, Column: 14, Byte: 14},
				},
			}},
		},
		{
			name:     "ErrCountBlocksTooMany",
			filename: "file.hcl",
			input: `
resource "a" {
	type = "min_max_blocks"

	nested {
	}
	nested {
	}
	nested {
	}
	nested {
	}
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Too many nested blocks",
				Detail:   "No more than 3 \"nested\" blocks are allowed",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 11, Column: 9, Byte: 89},
					End:      hcl.Pos{Line: 11, Column: 9, Byte: 89},
				},
			}},
		},
		{
			name:     "ErrExtraneousLabel",
			filename: "file.hcl",
			input: `
resource "role" {
	type = "aws:iam_role"

	assume_role_policy {
		statement {
			effect     = "Allow"
			actions    = ["sts:AssumeRole"]
			principals = {
				"Service" = ["lambda.amazonaws.com"]
			}
		}
	}

	policy "foo" "bar" {
	}
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Extraneous label for policy",
				Detail:   "Only 1 labels (name) are expected for policy blocks.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 15, Column: 15, Byte: 224},
					End:      hcl.Pos{Line: 15, Column: 20, Byte: 229},
				},
				Context: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 15, Column: 2, Byte: 211},
					End:      hcl.Pos{Line: 15, Column: 22, Byte: 231},
				},
			}},
		},
		{
			name:     "ErrConvert",
			filename: "file.hcl",
			input: `
resource "func" {
	type        = "aws:lambda_function"
	handler     = "index.handler"
	runtime     = {}
	role        = "testrole"
}
			`,
			want: &resource.Graph{
				Resources: map[string]resource.Resource{
					"func": {
						Type: "aws:lambda_function",
						Config: LambdaFunction{
							Handler:     "index.handler",
							Runtime:     "nodejs10.x",
							Role:        "testrole",
							Description: strptr("12345"),
						},
					},
				},
			},
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Incorrect attribute value type",
				Detail:   "Inappropriate value for attribute: string required.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 5, Column: 16, Byte: 102},
					End:      hcl.Pos{Line: 5, Column: 18, Byte: 104},
				},
				Expression: &hclsyntax.ObjectConsExpr{},
			}},
		},
		{
			name:     "ErrReferenceResource",
			filename: "file.hcl",
			input: `
resource "a" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"
}

resource "b" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = x.role
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "No such resource",
				Detail:   "A resource named \"x\" has not been declared.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 13, Column: 12, Byte: 235},
					End:      hcl.Pos{Line: 13, Column: 13, Byte: 236},
				},
			}},
		},
		{
			name:     "ErrReferenceInvalid",
			filename: "file.hcl",
			input: `
resource "a" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"
}

resource "b" {
	type        = "aws:lambda_function"
	handler     = "index.handler"
	runtime     = "nodejs10.x"
	role        = a.rule
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   "The resource \"a\" (aws:lambda_function) does not have such a field.",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 13, Column: 17, Byte: 252},
					End:      hcl.Pos{Line: 13, Column: 22, Byte: 257},
				},
			}},
		},
		{
			name:     "ErrReferenceInputNotSet",
			filename: "file.hcl",
			input: `
resource "a" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"
}

resource "b" {
	type        = "aws:lambda_function"
	handler     = "index.handler"
	runtime     = "nodejs10.x"
	role        = a.description
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Input value not set",
				Detail:   "A value has not been set for this field in \"a\".",
				Subject: &hcl.Range{
					Filename: "file.hcl",
					Start:    hcl.Pos{Line: 13, Column: 16, Byte: 251},
					End:      hcl.Pos{Line: 13, Column: 29, Byte: 264},
				},
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, writeDiags := parseBody(t, tc.filename, tc.input)

			reg := &resource.Registry{}
			reg.Add("aws:lambda_function", reflect.TypeOf(LambdaFunction{}))
			reg.Add("aws:iam_role", reflect.TypeOf(IAMRole{}))
			reg.Add("min_blocks", reflect.TypeOf(MinBlocks{}))
			reg.Add("max_blocks", reflect.TypeOf(MaxBlocks{}))
			reg.Add("min_max_blocks", reflect.TypeOf(MinMaxBlocks{}))

			opts := []cmp.Option{
				cmp.Comparer(func(a, b cty.Path) bool { return a.Equals(b) }),
				cmp.Comparer(func(a, b hcl.TraverseRoot) bool { return a.Name == b.Name }),
				cmp.Comparer(func(a, b hcl.TraverseAttr) bool { return a.Name == b.Name }),
				cmp.Comparer(func(a, b hcl.TraverseIndex) bool { return a.Key.RawEquals(b.Key) }),
				cmp.Comparer(func(a, b *hclsyntax.LiteralValueExpr) bool { return a.Val.RawEquals(b.Val) }),
				cmp.FilterPath(func(p cmp.Path) bool {
					switch p.Last().String() {
					case ".SrcRange", ".OpenRange":
						return true
					}
					return false
				}, cmp.Ignore()),
			}

			got, diags := resource.Decode(body, reg)
			if diff := cmp.Diff(diags, tc.wantDiags, opts...); diff != "" {
				t.Fatalf(
					"Diagnostics do not match\n\nGot\n%s\n\nWant\n%s\n\nDiff (-got +want):\n%s",
					writeDiags(diags), writeDiags(tc.wantDiags), diff,
				)
			}
			if tc.wantDiags.HasErrors() {
				return
			}

			if diff := cmp.Diff(got, tc.want, opts...); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}

func parseBody(t *testing.T, filename, input string) (hcl.Body, func(hcl.Diagnostics) string) {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(input), filename, hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("Failed to parse test input:\n%s", diags.Error())
	}

	return f.Body, func(diags hcl.Diagnostics) string {
		if len(diags) == 0 {
			return "No diagnostics"
		}
		var buf bytes.Buffer
		files := map[string]*hcl.File{filename: f}
		wr := hcl.NewDiagnosticTextWriter(&buf, files, 0, true)
		if err := wr.WriteDiagnostics(diags); err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(buf.String())
	}
}

func strptr(str string) *string { return &str }
func intptr(val int) *int       { return &val }
