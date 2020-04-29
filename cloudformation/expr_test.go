package cloudformation

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestConvertExpr(t *testing.T) {
	tests := []struct {
		name    string
		configs configProvider
		expr    hcl.Expression
		want    string
	}{
		{
			name: "OutputRef",
			configs: configMap{
				"a": struct {
					ID string `output:"id" cloudformation:"Id,ref"`
				}{},
			},
			expr: parseExpr(t, "a.id"),
			want: `{"Ref": "A"}`,
		},
		{
			name: "InputRef",
			configs: configMap{
				"a": struct {
					Name string `output:"name" cloudformation:"Name,ref"`
				}{},
			},
			expr: parseExpr(t, "a.name"),
			want: `{"Ref": "A"}`,
		},
		{
			name: "Att",
			configs: configMap{
				"a": struct {
					B string `output:"b" cloudformation:"Id,att"`
				}{},
			},
			expr: parseExpr(t, "a.b"),
			want: `{"Fn::GetAtt": "A.Id"}`,
		},
		{
			name: "Mixed",
			configs: configMap{
				"a": struct {
					Out string `output:"out" cloudformation:"Output,att"`
				}{},
			},
			expr: parseExpr(t, `">>>${a.out}<<<"`),
			want: `{"Fn::Sub": ">>>${A.Output}<<<"}`,
		},
		{
			name: "MixedWrappeed",
			configs: configMap{
				"parent": struct {
					A int `output:"a" cloudformation:"A,att"`
					B int `output:"b" cloudformation:"B,att"`
					C int `output:"c" cloudformation:"C,att"`
				}{},
			},
			expr: &hclsyntax.TemplateWrapExpr{
				Wrapped: parseExpr(t, `"${parent.a} + ${parent.b} + ${parent.c}"`),
			},
			want: `{"Fn::Sub": "${Parent.A} + ${Parent.B} + ${Parent.C}"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := convertExpr(tc.expr, tc.configs)
			if diags.HasErrors() {
				t.Fatal(diags)
			}
			equalAsJSON(t, got, tc.want)
		})
	}
}

func TestConvertExpr_diags(t *testing.T) {
	tests := []struct {
		name    string
		configs configProvider
		expr    string
		want    hcl.Diagnostics
	}{
		{
			name:    "InvalidRefName",
			configs: configMap{},
			expr:    "nonexisting.a",
			want: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference",
					Detail:   "No resource named \"nonexisting\".",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:   hcl.Pos{Line: 1, Column: 12, Byte: 11},
					},
				},
			},
		},
		{
			name: "InvalidRefNotCFOutput",
			configs: configMap{
				"a": struct {
					Output string `output:"output"` // No cloudformation output
				}{},
			},
			expr: "a.output",
			want: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference",
					Detail:   "CloudFormation does not support reading this field.",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 2, Byte: 1},
						End:   hcl.Pos{Line: 1, Column: 9, Byte: 8},
					},
				},
			},
		},
		{
			name: "InvalidRefNotOutput",
			configs: configMap{
				"a": struct {
					// No outputs
				}{},
			},
			expr: "a.output",
			want: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference",
					Detail:   "No such output.",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 2, Byte: 1},
						End:   hcl.Pos{Line: 1, Column: 9, Byte: 8},
					},
				},
			},
		},
		{
			name: "RefToInput",
			configs: configMap{
				"a": struct {
					Input string `input:"input" cloudformation:"Input"`
				}{},
			},
			expr: "a.input",
			want: hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Invalid reference",
					Detail:   "CloudFormation does not support reading this field.",
					Subject: &hcl.Range{
						Start: hcl.Pos{Line: 1, Column: 2, Byte: 1},
						End:   hcl.Pos{Line: 1, Column: 8, Byte: 7},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, writeDiags := parseExprDiags(t, tc.expr)
			_, diags := convertExpr(expr, tc.configs)

			opts := []cmp.Option{
				cmp.FilterPath(func(p cmp.Path) bool {
					return p.String() == "Subject.Filename"
				}, cmp.Ignore()),
			}
			if diff := cmp.Diff(diags, tc.want, opts...); diff != "" {
				t.Fatalf(
					"Diagnostics do not match\n\nGot\n%s\n\nWant\n%s\n\nDiff (-got +want):\n%s",
					writeDiags(diags), writeDiags(tc.want), diff,
				)
			}
		})
	}
}

type configMap map[string]interface{}

func (cfg configMap) config(name string) interface{} {
	return cfg[name]
}

func parseExpr(t *testing.T, input string) hclsyntax.Expression {
	expr, _ := parseExprDiags(t, input)
	return expr
}

func parseExprDiags(t *testing.T, input string) (hclsyntax.Expression, func(hcl.Diagnostics) string) {
	v, diags := hclsyntax.ParseExpression([]byte(input), t.Name(), hcl.InitialPos)
	if diags.HasErrors() {
		t.Helper()
		t.Fatal(diags)
	}
	return v, func(diags hcl.Diagnostics) string {
		if len(diags) == 0 {
			return "No diagnostics"
		}
		var buf bytes.Buffer
		files := map[string]*hcl.File{t.Name(): {
			Bytes: []byte(input),
		}}
		wr := hcl.NewDiagnosticTextWriter(&buf, files, 0, true)
		if err := wr.WriteDiagnostics(diags); err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(buf.String())
	}
}
