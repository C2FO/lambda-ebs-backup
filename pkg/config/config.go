package config

import (
	"os"
	"strconv"
)

func envDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func envDefaultInt(key, defaultValue string) (int, error) {
	return strconv.Atoi(envDefault(key, defaultValue))
}

// BackupTagKey is the volume tag key to look for when determing if we should
// perform a backup or not.
func BackupTagKey() string {
	return envDefault("BACKUP_TAG_KEY", "lambda-ebs-backup/backup")
}

// BackupTagValue is the volume tag value to look for when determing if we
// should perform a backup or not.
func BackupTagValue() string {
	return envDefault("BACKUP_TAG_VALUE", "true")
}

// ImageTagKey is the ec2 instance tag key to look for when deciding if we
// should create an image for the instance.
func ImageTagKey() string {
	return envDefault("IMAGE_TAG_KEY", "lambda-ebs-backup/image")
}

// ImageTagValue is the ec2 instance tag value to look for when deciding if an
// image should be created for the instance.
func ImageTagValue() string {
	return envDefault("IMAGE_TAG_VALUE", "true")
}

// ImageNameTag is the tag to look for on instances that decides how an image
// will be named. This tag supports a GO Template and overrides the default
// ImageNameFormat on an instance by instance basis.
func ImageNameTag() string {
	return envDefault("IMAGE_NAME_TAG", "lambda-ebs-backup/image-name")
}

// DefaultImageNameFormat is the default template to use for nameing ec2 images
// if a tag is not found. By default, we will use the name of the Instance
// postfixed with the date.
func DefaultImageNameFormat() string {
	return envDefault("DEFAULT_IMAGE_NAME_FORMAT", "{{.Name}}-{{.Date}}")
}

// MaxKeepImagesTag is the tag to look at for the maximum number of images to
// keep for an instance.
func MaxKeepImagesTag() string {
	return envDefault("MAX_KEEP_IMAGES_TAG", "lambda-ebs-backup/max-keep-images")
}

// MaxKeepImagesDefault is the default number of images to keep if not specified
// on the instance via the MaxKeepImagesTag
func DefaultMaxKeepImages() (int, error) {
	return envDefaultInt("DEFAULT_MAX_KEEP_IMAGES", "2")
}
