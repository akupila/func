package cloudformation

// A Deployment is an ongoing deployment from an executed change set.
type Deployment struct {
	ChangeSet *ChangeSet
}
