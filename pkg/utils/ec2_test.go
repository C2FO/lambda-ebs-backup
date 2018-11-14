package utils

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"
)

func TestTagMap(t *testing.T) {
	tags := []*ec2.Tag{
		&ec2.Tag{
			Key:   aws.String("Name"),
			Value: aws.String("TestName"),
		},
		&ec2.Tag{
			Key:   aws.String("TestKey"),
			Value: aws.String("TestValue"),
		},
	}

	tm := TagSliceToMap(tags)

	assert.Equal(t, "TestName", tm.GetDefault("Name", "defaultName"))
	assert.Equal(t, "defaultValue", tm.GetDefault("MissingKey", "defaultValue"))

	val, ok := tm.Get("TestKey")
	assert.True(t, ok)
	assert.Equal(t, "TestValue", val)

	val, ok = tm.Get("MissingKey")
	assert.False(t, ok)
	assert.Equal(t, "", val)
}
