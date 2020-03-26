package cli

import (
	"reflect"

	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
)

type App struct {
	Logger         *Logger
	Registry       *resource.Registry
	SourceS3Bucket string
}

func NewApp() *App {
	reg := &resource.Registry{}
	reg.Add("aws:iam_role", reflect.TypeOf(aws.IAMRole{}))
	reg.Add("aws:lambda_function", reflect.TypeOf(&aws.LambdaFunction{}))

	return &App{
		Registry: reg,
	}
}
