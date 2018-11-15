package backup

import (
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/c2fo/lambda-ebs-backup/pkg/utils"
)

// CommonTemplateContext that can be shared across more specific contexts
type CommonTemplateContext struct {
	Date     string
	FullDate string
	Name     string
}

func newCommonTemplateContext(name string) CommonTemplateContext {
	return CommonTemplateContext{
		Name:     name,
		Date:     time.Now().Format("2006-01-02"),
		FullDate: time.Now().Format("2006-01-02-15-04-05"),
	}
}

// ImageNameTemplateContext is what gets passed in as the context to the
// go template when attempting to create the name of the image for backup.
type ImageNameTemplateContext struct {
	CommonTemplateContext
}

func newImageNameTemplateContext(i *ec2.Instance) ImageNameTemplateContext {
	tags := utils.TagSliceToMap(i.Tags)
	name := tags.GetDefault("Name", "")

	return ImageNameTemplateContext{
		CommonTemplateContext: newCommonTemplateContext(name),
	}
}

// SnapshotNameTemplateContext is what gets passed in as the context to the
// go template when attempting ot create the name of the snapshot for backup.
type SnapshotNameTemplateContext struct {
	CommonTemplateContext
}

func newSnapshotNameTemplateContext(v *ec2.Volume) SnapshotNameTemplateContext {
	tags := utils.TagSliceToMap(v.Tags)
	name := tags.GetDefault("Name", "")
	return SnapshotNameTemplateContext{
		CommonTemplateContext: newCommonTemplateContext(name),
	}
}
