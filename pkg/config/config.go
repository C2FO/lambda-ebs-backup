package config

import "os"

func envDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
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
