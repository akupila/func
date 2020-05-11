package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseService(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *Service
	}{
		{
			name: "Meta",
			input: `
{
  "version": "2.0",
  "metadata": {
    "apiVersion": "2020-10-05",
    "endpointPrefix": "testing",
    "protocol": "rest-json",
    "serviceFullName": "Test Service",
    "serviceId": "Test",
    "signatureVersion": "v4",
    "uid": "test-2020-10-05"
  }
}
            `,
			want: &Service{
				Metadata: Metadata{
					APIVersion:       "2020-10-05",
					EndpointPrefix:   "testing",
					Protocol:         "rest-json",
					FullName:         "Test Service",
					ServiceID:        "Test",
					SignatureVersion: "v4",
					UID:              "test-2020-10-05",
				},
			},
		},
		{
			name: "Op",
			input: `
{
  "operations": {
    "CreateTest": {
      "name": "CreateTest",
      "http": {
        "method": "POST",
        "requestUri": "/2020-10-05/foo/bar",
        "responseCode": 201
      },
      "input": {
        "shape": "TestRequest"
      },
      "output": {
        "shape": "TestResponse"
      },
      "errors": [
        {
          "shape": "TestException"
        }
      ]
    }
  },
  "shapes": {
    "TestRequest": {
      "type": "structure",
      "members": {
        "foo": {
          "shape": "String"
        }
      }
    },
    "TestResponse": {
      "type": "structure",
      "members": {
        "bar": {
          "shape": "String"
        }
      }
    },
    "TestException": {
      "type": "structure",
      "members": {
        "baz": {
          "shape": "String"
        }
      }
    },
    "String": {
      "type": "string"
    }
  }
}
            `,
			want: &Service{
				Operations: []Operation{
					{
						Name: "CreateTest",
						HTTP: HTTPInfo{
							Method:       "POST",
							RequestURI:   "/2020-10-05/foo/bar",
							ResponseCode: 201,
						},
						Input: Struct{
							{Name: "foo", Type: String{}},
						},
						Output: Struct{
							{Name: "bar", Type: String{}},
						},
						Errors: []Struct{{
							{Name: "baz", Type: String{}},
						}},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseService(strings.NewReader(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			opts := []cmp.Option{
				cmpopts.EquateEmpty(),
			}
			if diff := cmp.Diff(got, tc.want, opts...); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}
