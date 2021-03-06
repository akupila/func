// Code generated by awsgen from api. DO NOT EDIT.

package iam

import "reflect"

// Registry maintains a list of supported iam resources.
type Registry interface {
	Add(typename string, typ reflect.Type)
}

// Register registers all AWS Identity and Access Management resources.
func Register(reg Registry) {
	reg.Add("aws:iam_access_key", reflect.TypeOf(&AccessKey{}))
	reg.Add("aws:iam_account_alias", reflect.TypeOf(&AccountAlias{}))
	reg.Add("aws:iam_group", reflect.TypeOf(&Group{}))
	reg.Add("aws:iam_instance_profile", reflect.TypeOf(&InstanceProfile{}))
	reg.Add("aws:iam_login_profile", reflect.TypeOf(&LoginProfile{}))
	reg.Add("aws:iam_open_id_connect_provider", reflect.TypeOf(&OpenIDConnectProvider{}))
	reg.Add("aws:iam_policy", reflect.TypeOf(&Policy{}))
	reg.Add("aws:iam_policy_version", reflect.TypeOf(&PolicyVersion{}))
	reg.Add("aws:iam_role", reflect.TypeOf(&Role{}))
	reg.Add("aws:iam_saml_provider", reflect.TypeOf(&SAMLProvider{}))
	reg.Add("aws:iam_service_linked_role", reflect.TypeOf(&ServiceLinkedRole{}))
	reg.Add("aws:iam_service_specific_credential", reflect.TypeOf(&ServiceSpecificCredential{}))
	reg.Add("aws:iam_user", reflect.TypeOf(&User{}))
	reg.Add("aws:iam_virtual_mfa_device", reflect.TypeOf(&VirtualMFADevice{}))
}
