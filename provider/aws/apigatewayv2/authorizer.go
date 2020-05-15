// Code generated by awsgen from api version 2018-11-29. DO NOT EDIT.

package apigatewayv2

// Authorizer manages AmazonApiGatewayV2 Authorizers.
type Authorizer struct {
	// The API identifier.
	API string `cloudformation:"ApiId" input:"api" json:"ApiId"`

	// Specifies the required credentials as an IAM role for API Gateway to
	// invoke the authorizer. To specify an IAM role for API Gateway to assume,
	// use the role's Amazon Resource Name (ARN). To use resource-based
	// permissions on the Lambda function, specify null. Supported only for
	// REQUEST authorizers.
	CredentialsARN *string `cloudformation:"AuthorizerCredentialsArn" input:"credentials_arn" json:"AuthorizerCredentialsArn"`

	// Authorizer caching is not currently supported. Don't specify this value
	// for authorizers.
	ResultTTL *int `cloudformation:"AuthorizerResultTtlInSeconds" input:"result_ttl" json:"AuthorizerResultTtlInSeconds"`

	// The authorizer type. For WebSocket APIs, specify REQUEST for a Lambda
	// function using incoming request parameters. For HTTP APIs, specify JWT
	// to use JSON Web Tokens.
	Type string `cloudformation:"AuthorizerType" input:"type" json:"AuthorizerType"`

	// The authorizer's Uniform Resource Identifier (URI). For REQUEST authorizers, this must be a well-formed Lambda function URI, for example, arn:aws:apigateway:us-west-2:lambda:path/2015-03-31/functions/arn:aws:lambda:us-west-2:{account_id}:function:{lambda_function_name}/invocations. In general, the URI has this form:
	// arn:aws:apigateway:{region}:lambda:path/{service_api}
	//                , where {region} is the same as the region hosting the Lambda function, path indicates that the remaining substring in the URI should be treated as the path to the resource, including the initial /. For Lambda functions, this is usually of the form /2015-03-31/functions/[FunctionARN]/invocations. Supported only for REQUEST authorizers.
	URI *string `cloudformation:"AuthorizerUri" input:"uri" json:"AuthorizerUri"`

	// The identity source for which authorization is requested.For a REQUEST
	// authorizer, this is optional. The value is a set of one or more mapping
	// expressions of the specified request parameters. Currently, the identity
	// source can be headers, query string parameters, stage variables, and
	// context parameters. For example, if an Auth header and a Name query
	// string parameter are defined as identity sources, this value is
	// route.request.header.Auth, route.request.querystring.Name. These
	// parameters will be used to perform runtime validation for Lambda-based
	// authorizers by verifying all of the identity-related request parameters
	// are present in the request, not null, and non-empty. Only when this is
	// true does the authorizer invoke the authorizer Lambda function.
	// Otherwise, it returns a 401 Unauthorized response without calling the
	// Lambda function.For JWT, a single entry that specifies where to extract
	// the JSON Web Token (JWT )from inbound requests. Currently only
	// header-based and query parameter-based selections are supported, for
	// example "$request.header.Authorization".
	IdentitySource []string `cloudformation:"IdentitySource" input:"identity_source"`

	// This parameter is not used.
	IdentifyValidation *string `cloudformation:"IdentityValidationExpression" input:"identify_validation" json:"IdentityValidationExpression"`

	// Represents the configuration of a JWT authorizer. Required for the JWT
	// authorizer type. Supported only for HTTP APIs.
	JWT *struct {
		// A list of the intended recipients of the JWT. A valid JWT must provide
		// an aud that matches at least one entry in this list. See RFC 7519.
		// Supported only for HTTP APIs.
		Audience []string `cloudformation:"Audience" input:"audience"`

		// The base domain of the identity provider that issues JSON Web Tokens. For example,
		// an Amazon Cognito user pool has the following format:
		// https://cognito-idp.{region}.amazonaws.com/{userPoolId}
		//                . Required for the JWT authorizer type. Supported only for HTTP APIs.
		Issuer *string `cloudformation:"Issuer" input:"issuer"`
	} `cloudformation:"JwtConfiguration" input:"jwt" json:"JwtConfiguration"`

	// The name of the authorizer.
	Name string `cloudformation:"Name" input:"name"`

	// Outputs:

	// The authorizer identifier.
	ID *string `json:"AuthorizerId" output:"id"`
}

// CloudFormationType returns the CloudFormation type for a AmazonApiGatewayV2 Authorizer.
func (Authorizer) CloudFormationType() string { return "AWS::ApiGatewayV2::Authorizer" }
