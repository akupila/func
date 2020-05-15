// Code generated by awsgen from api version 2010-05-08. DO NOT EDIT.

package iam

// AccountAlias manages AWS Identity and Access Management AccountAliases.
type AccountAlias struct {
	// The account alias to create.This parameter allows (through its regex
	// pattern) a string of characters consisting of lowercase letters, digits,
	// and dashes. You cannot start or finish with a dash, nor can you have two
	// dashes in a row.
	Alias string `input:"alias" json:"AccountAlias"`
}
