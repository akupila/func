// Code generated by awsgen from api version 2018-11-29. DO NOT EDIT.

package apigatewayv2

// IntegrationResponse manages AmazonApiGatewayV2 IntegrationResponses.
type IntegrationResponse struct {
	// The API identifier.
	API string `cloudformation:"ApiId" input:"api" json:"ApiId"`

	// Specifies how to handle response payload content type conversions.
	// Supported values are CONVERT_TO_BINARY and CONVERT_TO_TEXT, with the
	// following behaviors:CONVERT_TO_BINARY: Converts a response payload from
	// a Base64-encoded string to the corresponding binary
	// blob.CONVERT_TO_TEXT: Converts a response payload from a binary blob to
	// a Base64-encoded string.If this property is not defined, the response
	// payload will be passed through from the integration response to the
	// route response or method response without modification.
	ContentHandling *string `cloudformation:"ContentHandlingStrategy" input:"content_handling" json:"ContentHandlingStrategy"`

	// The integration ID.
	Integration string `cloudformation:"IntegrationId" input:"integration" json:"IntegrationId"`

	// The integration response key.
	ResponseKey string `cloudformation:"IntegrationResponseKey" input:"response_key" json:"IntegrationResponseKey"`

	// A key-value map specifying response parameters that are passed to the
	// method response from the backend. The key is a method response header
	// parameter name and the mapped value is an integration response header
	// value, a static value enclosed within a pair of single quotes, or a JSON
	// expression from the integration response body. The mapping key must
	// match the pattern of method.response.header.{name}, where {name} is a
	// valid and unique header name. The mapped non-static value must match the
	// pattern of integration.response.header.{name} or
	// integration.response.body.{JSON-expression}, where {name} is a valid and
	// unique response header name and {JSON-expression} is a valid JSON
	// expression without the $ prefix.
	ResponseParameters map[string]string `cloudformation:"ResponseParameters" input:"response_parameters"`

	// The collection of response templates for the integration response as a
	// string-to-string map of key-value pairs. Response templates are
	// represented as a key/value map, with a content-type as the key and a
	// template as the value.
	ResponseTemplates map[string]string `cloudformation:"ResponseTemplates" input:"response_templates"`

	// The template selection expression for the integration response.
	// Supported only for WebSocket APIs.
	TemplateSelection *string `cloudformation:"TemplateSelectionExpression" input:"template_selection" json:"TemplateSelectionExpression"`

	// Outputs:

	// The integration response ID.
	ID *string `json:"IntegrationResponseId" output:"id"`
}

// CloudFormationType returns the CloudFormation type for a AmazonApiGatewayV2 IntegrationResponse.
func (IntegrationResponse) CloudFormationType() string {
	return "AWS::ApiGatewayV2::IntegrationResponse"
}
