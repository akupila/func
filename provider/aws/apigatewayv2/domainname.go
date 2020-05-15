// Code generated by awsgen from api version 2018-11-29. DO NOT EDIT.

package apigatewayv2

import "time"

// DomainName manages AmazonApiGatewayV2 DomainNames.
type DomainName struct {
	// The domain name.
	Name string `cloudformation:"DomainName" input:"name" json:"DomainName"`

	// The domain name configurations.
	Config []struct {
		// A domain name for the API.
		DomainName *string `input:"domain_name" json:"ApiGatewayDomainName"`

		// An AWS-managed certificate that will be used by the edge-optimized
		// endpoint for this domain name. AWS Certificate Manager is the only
		// supported source.
		CertificateARN *string `cloudformation:"CertificateArn" input:"certificate_arn" json:"CertificateArn"`

		// The user-friendly name of the certificate that will be used by the
		// edge-optimized endpoint for this domain name.
		CertificateName *string `cloudformation:"CertificateName" input:"certificate_name"`

		// The timestamp when the certificate that was used by edge-optimized
		// endpoint for this domain name was uploaded.
		CertificateUploadDate *time.Time `input:"certificate_upload_date"`

		// The status of the domain name migration. The valid values are AVAILABLE
		// and UPDATING. If the status is UPDATING, the domain cannot be modified
		// further until the existing operation is complete. If it is AVAILABLE,
		// the domain can be updated.
		DomainNameStatus *string `input:"domain_name_status"`

		// An optional text message containing detailed information about status of
		// the domain name migration.
		DomainNameStatusMessage *string `input:"domain_name_status_message"`

		// The endpoint type.
		EndpointType *string `cloudformation:"EndpointType" input:"endpoint_type"`

		// The Amazon Route 53 Hosted Zone ID of the endpoint.
		HostedZoneID *string `input:"hosted_zone_id" json:"HostedZoneId"`

		// The Transport Layer Security (TLS) version of the security policy for
		// this domain name. The valid values are TLS_1_0 and TLS_1_2.
		SecurityPolicy *string `input:"security_policy"`
	} `cloudformation:"DomainNameConfigurations" input:"config" json:"DomainNameConfigurations"`

	// The collection of tags associated with a domain name.
	Tags map[string]string `cloudformation:"Tags" input:"tags"`

	// Outputs:

	// The API mapping selection expression.
	Mapping *string `json:"ApiMappingSelectionExpression" output:"mapping"`
}

// CloudFormationType returns the CloudFormation type for a AmazonApiGatewayV2 DomainName.
func (DomainName) CloudFormationType() string { return "AWS::ApiGatewayV2::DomainName" }