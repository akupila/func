// Code generated by awsgen from api. DO NOT EDIT.

package lambda

import "reflect"

// Registry maintains a list of supported lambda resources.
type Registry interface {
	Add(typename string, typ reflect.Type)
}

// Register registers all AWS Lambda resources.
func Register(reg Registry) {
	reg.Add("aws:lambda_alias", reflect.TypeOf(&Alias{}))
	reg.Add("aws:lambda_event_source_mapping", reflect.TypeOf(&EventSourceMapping{}))
	reg.Add("aws:lambda_function", reflect.TypeOf(&Function{}))
	reg.Add("aws:lambda_layer_version_permission", reflect.TypeOf(&LayerVersionPermission{}))
	reg.Add("aws:lambda_permission", reflect.TypeOf(&Permission{}))
}