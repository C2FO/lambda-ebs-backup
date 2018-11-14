package backup

import (
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/c2fo/lambda-ebs-backup/pkg/utils"
)

// ImageNameTemplateContext is what gets passed in as the context to the
// go template when attempting to create the name of the image for backup.
type ImageNameTemplateContext struct {
	Name     string
	Date     string
	FullDate string
}

func newImageNameTemplateContext(i *ec2.Instance) ImageNameTemplateContext {
	tags := utils.TagSliceToMap(i.Tags)

	return ImageNameTemplateContext{
		Name:     tags.GetDefault("Name", ""),
		Date:     time.Now().Format("2006-01-02"),
		FullDate: time.Now().Format("2006-01-02-15-04-05"),
	}
}
