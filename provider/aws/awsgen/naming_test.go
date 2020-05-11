package main

import "testing"

func TestPackageName(t *testing.T) {
	tests := []struct {
		serviceID string
		want      string
	}{
		{"ApiGatewayV2", "apigatewayv2"},
		{"DynamoDB", "dynamodb"},
		{"Lambda", "lambda"},
		{"Route 53", "route53"},
		{"S3", "s3"},
		{"SQS", "sqs"},
	}

	for _, tc := range tests {
		t.Run(tc.serviceID, func(t *testing.T) {
			got := PackageName(tc.serviceID)
			if got != tc.want {
				t.Errorf("Got = %q, want = %q", got, tc.want)
			}
		})
	}
}
