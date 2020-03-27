package cli

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/func/func/cloudformation"
)

// Deploy deploys the resources in dir to a CloudFormation stack.
func (a *App) Deploy(ctx context.Context, dir, stackName string) int {
	tmpl, code := a.GenerateCloudFormation(ctx, dir)
	if code != 0 {
		return code
	}

	a.Logger.Traceln("Creating CloudFormation change set")

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		a.Logger.Errorln(err)
		return 1
	}
	cf := cloudformation.NewClient(cfg)

	stack, err := cf.StackByName(ctx, stackName)
	if err != nil {
		a.Logger.Errorf("Could not get stack: %v", err)
		return 1
	}

	cs, err := cf.CreateChangeSet(ctx, stack, tmpl)
	if err != nil {
		a.Logger.Errorf("Could not create change set: %v", err)
		return 1
	}

	if len(cs.Changes) == 0 {
		a.Logger.Infoln("No changes")
		_ = cf.DeleteChangeSet(ctx, cs) // Ignore error
		return 0
	}

	a.Logger.Verboseln("Deploying")

	deployment, err := cf.ExecuteChangeSet(ctx, cs)
	if err != nil {
		a.Logger.Errorf("Could not execute change set: %v", err)
		return 1
	}

	for ev := range cf.Events(ctx, deployment) {
		switch e := ev.(type) {
		case cloudformation.ErrorEvent:
			a.Logger.Errorf("Deployment error: %v", err)
			return 1
		case cloudformation.ResourceEvent:
			name := e.LogicalID
			if v := tmpl.LookupResource(e.LogicalID); v != "" {
				name = v
			}
			a.Logger.Verbosef("  %s: %s %s %s\n", name, e.Operation, e.State, e.Reason)
		case cloudformation.StackEvent:
			if e.State == cloudformation.StateComplete {
				if e.Operation == cloudformation.StackRollback {
					a.Logger.Errorf("Deployment failed: %s\n", e.Reason)
					return 1
				}
				a.Logger.Infoln("Deployed")
			}
		}
	}

	a.Logger.Infoln("Done")

	return 0
}
