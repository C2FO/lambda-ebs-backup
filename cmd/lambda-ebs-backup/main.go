package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/c2fo/lambda-ebs-backup/pkg/config"
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

	params := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String(fmt.Sprintf("tag:%s", config.BackupTagKey())),
				Values: []*string{aws.String(config.BackupTagValue())},
			},
		},
		MaxResults: aws.Int64(500),
	}

	fmt.Printf("Searching for volumes with tag %s:%s\n", config.BackupTagKey(), config.BackupTagValue())
	err := ec2Client.DescribeVolumesPages(params,
		func(page *ec2.DescribeVolumesOutput, lastPage bool) bool {
			for _, v := range page.Volumes {
				fmt.Printf("Found %s\n", aws.StringValue(v.VolumeId))
				snap, snapErr := ec2Client.CreateSnapshot(&ec2.CreateSnapshotInput{
					VolumeId: v.VolumeId,
				})
				if snapErr != nil {
					fmt.Printf("Err: %s\n", snapErr)
				}
				fmt.Printf("Created snapshot '%s' for volume '%s'\n",
					aws.StringValue(snap.SnapshotId),
					aws.StringValue(v.VolumeId),
				)
			}
			return !lastPage
		})

	if err != nil {
		return err
	}

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
