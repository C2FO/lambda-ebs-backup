package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/c2fo/lambda-ebs-backup/pkg/backup"
)

// HandleRequest handles the lambda request
func HandleRequest(ctx context.Context) error {
	sess := session.Must(
		session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
			},
		),
	)

	ec2Client := ec2.New(sess)
	opts, err := backup.NewManagerOptsFromConfig(ec2Client)
	if err != nil {
		return err
	}

	backupManager, err := backup.NewManager(opts)
	if err != nil {
		return err
	}

	err = backupManager.Search()
	if err != nil {
		return err
	}

	if err = backupManager.Backup(); err != nil {
		return err
	}

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
