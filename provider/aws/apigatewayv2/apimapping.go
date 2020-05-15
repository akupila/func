// Code generated by awsgen from api version 2018-11-29. DO NOT EDIT.

package apigatewayv2

// ApiMapping manages AmazonApiGatewayV2 ApiMappings.
type ApiMapping struct {
	// The API identifier.
	API string `cloudformation:"ApiId" input:"api" json:"ApiId"`

	// The API mapping key.
	Key *string `cloudformation:"ApiMappingKey" input:"key" json:"ApiMappingKey"`

	// The domain name.
	DomainName string `cloudformation:"DomainName" input:"domain_name"`

	// The API stage.
	Stage string `cloudformation:"Stage" input:"stage"`

	// Outputs:

	// The API mapping identifier.
	ID *string `json:"ApiMappingId" output:"id"`
}

// CloudFormationType returns the CloudFormation type for a AmazonApiGatewayV2 ApiMapping.
func (ApiMapping) CloudFormationType() string { return "AWS::ApiGatewayV2::ApiMapping" }