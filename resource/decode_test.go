package resource_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/func/func/resource"
	"github.com/func/func/source"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/tools/txtar"
)

func TestDecoder_Decode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      resource.List
		wantDiags hcl.Diagnostics
	}{
		// Attributes
		{
			name: "Attributes",
			input: `
-- file.hcl --
resource "func" {
	type        = "aws:lambda_function"
	handler     = "index.handler"
	runtime     = "nodejs10.x"
	role        = "testrole"
	memory_size = 256
}
			`,
			want: resource.List{
				{
					Name: "func",
					Type: "aws:lambda_function",
					Definition: hcl.Range{
						Filename: "<DIR>/file.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
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

		// Blocks
		{
			name: "Block",
			input: `
-- file.hcl --
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
			want: resource.List{
				{
					Name: "func",
					Type: "aws:lambda_function",
					Definition: hcl.Range{
						Filename: "<DIR>/file.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
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
		{
			name: "BlockMapList",
			input: `
-- file.hcl --
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
			want: resource.List{
				{
					Name: "role",
					Type: "aws:iam_role",
					Definition: hcl.Range{
						Filename: "<DIR>/file.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
					},
					Config: IAMRole{
						AssumeRolePolicy: IAMPolicyDocument{
							Statements: []IAMPolicyStatement{{
								Effect:  "Allow",
								Actions: []string{"sts:AssumeRole"},
								Principals: map[string][]string{
									"Service": {"lambda.amazonaws.com"},
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

		// Source code
		{
			name: "Source",
			input: `
-- file.hcl --
resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"

	source {
		dir = "."
	}
}

-- .git/HEAD --
-- index.js --
module.exports = function() {}
			`,
			want: resource.List{
				{
					Name: "func",
					Type: "aws:lambda_function",
					Definition: hcl.Range{
						Filename: "<DIR>/file.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
					},
					Config: LambdaFunction{
						Handler: "index.handler",
						Runtime: "nodejs10.x",
						Role:    "testrole",
					},
					SourceCode: &source.FileList{
						Root:  "<DIR>",
						Files: []string{"index.js"},
					},
				},
			},
		},
		{
			name: "SourceNested",
			input: `
-- a/b/c/foo.hcl --
resource "func" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"

	source {
		dir = "source"
	}
}

-- a/b/c/README.md --
# Test
-- a/b/c/source/index.js --
module.exports = function() {}
			`,
			want: resource.List{
				{
					Name: "func",
					Type: "aws:lambda_function",
					Definition: hcl.Range{
						Filename: "<DIR>/a/b/c/foo.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
					},
					Config: LambdaFunction{
						Handler: "index.handler",
						Runtime: "nodejs10.x",
						Role:    "testrole",
					},
					SourceCode: &source.FileList{
						Root:  "<DIR>/a/b/c/source",
						Files: []string{"index.js"},
					},
				},
			},
		},

		// References
		{
			name: "ReferenceToInput",
			input: `
-- a.hcl --
resource "a" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = "testrole"
}

-- b.hcl --
resource "b" {
	type    = "aws:lambda_function"
	handler = "index.handler"
	runtime = "nodejs10.x"
	role    = a.role
}
			`,
			want: resource.List{
				{
					Name: "a",
					Type: "aws:lambda_function",
					Definition: hcl.Range{
						Filename: "<DIR>/a.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 13, Byte: 12},
					},
					Config: LambdaFunction{
						Handler: "index.handler",
						Runtime: "nodejs10.x",
						Role:    "testrole",
					},
				},
				{
					Name: "b",
					Type: "aws:lambda_function",
					Definition: hcl.Range{
						Filename: "<DIR>/b.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 13, Byte: 12},
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
		{
			name: "ReferenceToOutput",
			input: `
-- file.hcl --
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
			want: resource.List{
				{
					Name: "role",
					Type: "aws:iam_role",
					Definition: hcl.Range{
						Filename: "<DIR>/file.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
					},
					Config: IAMRole{
						AssumeRolePolicy: IAMPolicyDocument{
							Statements: []IAMPolicyStatement{{
								Effect:  "Allow",
								Actions: []string{"sts:AssumeRole"},
								Principals: map[string][]string{
									"Service": {"lambda.amazonaws.com"},
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
				{
					Name: "func",
					Type: "aws:lambda_function",
					Definition: hcl.Range{
						Filename: "<DIR>/file.hcl",
						Start:    hcl.Pos{Line: 23, Column: 1, Byte: 320},
						End:      hcl.Pos{Line: 23, Column: 16, Byte: 335},
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

		// Conversion
		{
			name: "ConvertNumberToString",
			input: `
-- file.hcl --
resource "func" {
	type        = "aws:lambda_function"
	handler     = "index.handler"
	runtime     = "nodejs10.x"
	role        = "testrole"
	description = 12345
}
			`,
			want: resource.List{
				{
					Name: "func",
					Type: "aws:lambda_function",
					Definition: hcl.Range{
						Filename: "<DIR>/file.hcl",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
					},
					Config: LambdaFunction{
						Handler:     "index.handler",
						Runtime:     "nodejs10.x",
						Role:        "testrole",
						Description: strptr("12345"),
					},
				},
			},
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagWarning,
				Summary:  "Value is converted from number to string",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 6, Column: 16, Byte: 155},
					End:      hcl.Pos{Line: 6, Column: 21, Byte: 160},
				},
				Expression: &hclsyntax.LiteralValueExpr{
					Val: cty.NumberIntVal(12345),
				},
			}},
		},

		// Errors
		{
			name: "ErrTypeMissing",
			input: `
-- file.hcl --
resource "err" {
	# No type
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing required argument",
				Detail:   "The argument \"type\" is required, but no definition was found.",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 16, Byte: 15},
					End:      hcl.Pos{Line: 1, Column: 16, Byte: 15},
				},
			}},
		},
		{
			name: "ErrTypeVariable",
			input: `
-- file.hcl --
resource "err" {
	type = foo
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Variables not allowed",
				Detail:   "Variables may not be used here.",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 9, Byte: 25},
					End:      hcl.Pos{Line: 2, Column: 12, Byte: 28},
				},
				Expression: &hclsyntax.ScopeTraversalExpr{
					Traversal: hcl.Traversal{
						hcl.TraverseRoot{Name: "foo"},
					},
				},
			}},
		},
		{
			name: "ErrTypeNotFound",
			input: `
-- file.hcl --
resource "err" {
	type = "invalid"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported resource",
				Detail:   "Resources of type \"invalid\" are not supported.",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 9, Byte: 25},
					End:      hcl.Pos{Line: 2, Column: 18, Byte: 34},
				},
			}},
		},
		{
			name: "ErrTypeNotFoundSuggest",
			input: `
-- file.hcl --
resource "err" {
	type = "aws/lambda-function"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported resource",
				Detail:   "Resources of type \"aws/lambda-function\" are not supported. Did you mean \"aws:lambda_function\"?",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 2, Column: 9, Byte: 25},
					End:      hcl.Pos{Line: 2, Column: 30, Byte: 46},
				},
			}},
		},
		{
			name: "ErrUnsupportedBlock",
			input: `
-- file.hcl --
xxx {
	type = ""
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Unsupported block type",
				Detail:   "Blocks of type \"xxx\" are not expected here.",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 4, Byte: 3},
				},
			}},
		},
		{
			name: "ErrEmptyName",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
					End:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
				},
			}},
		},
		{
			name: "ErrDuplicateResourceName",
			input: `
-- file.hcl --
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
				Detail:   "Another resource named \"func\" was defined in <DIR>/file.hcl on line 1.",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 8, Column: 1, Byte: 122},
					End:      hcl.Pos{Line: 8, Column: 16, Byte: 137},
				},
			}},
		},
		{
			name: "ErrRequiredAttributeNotSet",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 17, Byte: 16},
					End:      hcl.Pos{Line: 1, Column: 17, Byte: 16},
				},
			}},
		},
		{
			name: "ErrRequiredBlockNotSet",
			input: `
-- file.hcl --
resource "role" {
	type = "aws:iam_role"
}
			`,
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Missing assume_role_policy block",
				Detail:   "A block of type \"assume_role_policy\" is required here.",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 17, Byte: 16},
					End:      hcl.Pos{Line: 1, Column: 17, Byte: 16},
				},
			}},
		},
		{
			name: "ErrTooManyBlocks", // Not targeting slice of structs
			input: `
-- file.hcl --
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
				Detail:   "Only one block of type \"environment\" is allowed. Previous definition was at <DIR>/file.hcl:7,2-13.",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 11, Column: 2, Byte: 146},
					End:      hcl.Pos{Line: 11, Column: 13, Byte: 157},
				},
			}},
		},
		{
			name: "ErrTooFewBlocks",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 17, Byte: 16},
					End:      hcl.Pos{Line: 1, Column: 17, Byte: 16},
				},
			}},
		},
		{
			name: "ErrTooManyBlocksSlice",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 8, Column: 9, Byte: 74},
					End:      hcl.Pos{Line: 8, Column: 9, Byte: 74},
				},
			}},
		},
		{
			name: "ErrCountBlocksTooFew",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 1, Column: 14, Byte: 13},
					End:      hcl.Pos{Line: 1, Column: 14, Byte: 13},
				},
			}},
		},
		{
			name: "ErrCountBlocksTooMany",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 10, Column: 9, Byte: 88},
					End:      hcl.Pos{Line: 10, Column: 9, Byte: 88},
				},
			}},
		},
		{
			name: "ErrExtraneousLabel",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 14, Column: 15, Byte: 223},
					End:      hcl.Pos{Line: 14, Column: 20, Byte: 228},
				},
				Context: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 14, Column: 2, Byte: 210},
					End:      hcl.Pos{Line: 14, Column: 22, Byte: 230},
				},
			}},
		},
		{
			name: "ErrConvert",
			input: `
-- file.hcl --
resource "func" {
	type        = "aws:lambda_function"
	handler     = "index.handler"
	runtime     = {}
	role        = "testrole"
}
			`,
			want: resource.List{
				{
					Name: "func",
					Type: "aws:lambda_function",
					Config: LambdaFunction{
						Handler:     "index.handler",
						Runtime:     "nodejs10.x",
						Role:        "testrole",
						Description: strptr("12345"),
					},
				},
			},
			wantDiags: hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Incorrect attribute value type",
				Detail:   "Inappropriate value for attribute: string required.",
				Subject: &hcl.Range{
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 4, Column: 16, Byte: 101},
					End:      hcl.Pos{Line: 4, Column: 18, Byte: 103},
				},
				Expression: &hclsyntax.ObjectConsExpr{},
			}},
		},
		{
			name: "ErrReferenceResource",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 12, Column: 12, Byte: 234},
					End:      hcl.Pos{Line: 12, Column: 13, Byte: 235},
				},
			}},
		},
		{
			name: "ErrReferenceInvalid",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 12, Column: 17, Byte: 251},
					End:      hcl.Pos{Line: 12, Column: 22, Byte: 256},
				},
			}},
		},
		{
			name: "ErrReferenceInputNotSet",
			input: `
-- file.hcl --
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
					Filename: "<DIR>/file.hcl",
					Start:    hcl.Pos{Line: 12, Column: 16, Byte: 250},
					End:      hcl.Pos{Line: 12, Column: 29, Byte: 263},
				},
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := tempdir(t)
			writeTxtar(t, dir, tc.input)
			body, writeDiags := parseConfig(t, dir)

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
				cmp.Comparer(func(a, b string) bool {
					a = strings.ReplaceAll(a, "<DIR>", dir)
					b = strings.ReplaceAll(b, "<DIR>", dir)
					return a == b
				}),
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

func tempdir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Helper()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func writeTxtar(t *testing.T, dir, input string) {
	archive := txtar.Parse([]byte(input))
	for _, f := range archive.Files {
		filename := filepath.Join(dir, f.Name)
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(filename, f.Data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func parseConfig(t *testing.T, dir string) (hcl.Body, func(hcl.Diagnostics) string) {
	t.Helper()

	parser := hclparse.NewParser()
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(info.Name()) != ".hcl" {
			return nil
		}
		_, diags := parser.ParseHCLFile(path)
		if diags.HasErrors() {
			return diags
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	files := parser.Files()
	filelist := make([]*hcl.File, 0, len(files))
	for _, f := range parser.Files() {
		filelist = append(filelist, f)
	}
	body := hcl.MergeFiles(filelist)

	return body, func(diags hcl.Diagnostics) string {
		if len(diags) == 0 {
			return "No diagnostics"
		}
		var buf bytes.Buffer
		wr := hcl.NewDiagnosticTextWriter(&buf, files, 0, true)
		if err := wr.WriteDiagnostics(diags); err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(buf.String())
	}
}

func strptr(str string) *string { return &str }
func intptr(val int) *int       { return &val }
