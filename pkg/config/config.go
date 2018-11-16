package config

import (
	"os"
	"strconv"
	"strings"
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

func envDefaultBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	if strings.ToUpper(value) == "TRUE" {
		return true
	}
	return false
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

// DefaultImageNameFormat is the default template to use for naming ec2 images
// if a tag is not found. By default, we will use the name of the Instance
// postfixed with the date.
func DefaultImageNameFormat() string {
	return envDefault("DEFAULT_IMAGE_NAME_FORMAT", "{{.Name}}-{{.Date}}")
}

// SnapshotNameTag is the tag to look for on volumes that decides how a snapshot
// will be named. This tag supports a GO Template and overrides the default
// SnapshotNameFormat on an volume by volume basis.
func SnapshotNameTag() string {
	return envDefault("SNAPSHOT_NAME_TAG", "lambda-ebs-backup/snapshot-name")
}

// DefaultSnapshotNameFormat is the default format for a snapshot's name if
// one is not supplied on the volume.
func DefaultSnapshotNameFormat() string {
	return envDefault("DEFAULT_SNAPSHOT_NAME_FORMAT", "{{.Name}}-{{.Date}}")
}

// MaxKeepImagesTag is the tag to look at for the maximum number of images to
// keep for an instance.
func MaxKeepImagesTag() string {
	return envDefault("MAX_KEEP_IMAGES_TAG", "lambda-ebs-backup/max-keep-images")
}

// DefaultMaxKeepImages is the default number of images to keep if not specified
// on the instance via the MaxKeepImagesTag
func DefaultMaxKeepImages() (int, error) {
	return envDefaultInt("DEFAULT_MAX_KEEP_IMAGES", "2")
}

// MaxKeepSnapshotsTag is the tag to look at for the maximum number of snapshots
// to keep for a volume.
func MaxKeepSnapshotsTag() string {
	return envDefault("MAX_KEEP_SNAPSHOTS_TAG", "lambda-ebs-backup/max-keep-snapshots")
}

// DefaultMaxKeepSnapshots is the default number of snapshots to keep for a
// volume if not specified on the volume via the MaxKeepSnapshotsTag.
func DefaultMaxKeepSnapshots() (int, error) {
	return envDefaultInt("DEFAULT_MAX_KEEP_SNAPSHOTS", "2")
}

// RebootOnImageTag returns the name of the EC2 tag to look at to see if the
// instance should be rebooted or not when an image is made. If not supplied,
// the value will default to that of "DEFAULT_REBOOT_ON_IMAGE"
func RebootOnImageTag() string {
	return envDefault("REBOOT_ON_IMAGE_TAG", "lambda-ebs-backup/reboot-on-image")
}

// DefaultRebootOnImage determines the default behavior for rebooting when we
// take an image of an ec2 instance. If not specified, it defaults to true as
// this is the aws CreateImage default.
func DefaultRebootOnImage() bool {
	return envDefaultBool("DEFAULT_REBOOT_ON_IMAGE", true)
}

// ManagedTagKey will get added to all resources created by the backup so we
// can search for them later for things like auditing and cleaning up backups
func ManagedTagKey() string {
	return envDefault("MANAGED_TAG_KEY", "lambda-ebs-backup/managed")
}

// ManagedTagValue is the value that corresponds to the ManagedTagKey that gets
// added to all resources created by backup.
func ManagedTagValue() string {
	return envDefault("MANAGED_TAG_VALUE", "true")
}
