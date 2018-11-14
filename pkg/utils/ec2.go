package utils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// TagMap stores the ec2 tags as a map and provides some convenience methods.
type TagMap struct {
	m map[string]string
}

// Get gets the tag with the given key and whether or not it exists
func (tm *TagMap) Get(key string) (string, bool) {
	v, ok := tm.m[key]
	return v, ok
}

// GetDefault returns the value of the tag matching the given key. If there is
// no tag matching the key, it returns the default value.
func (tm *TagMap) GetDefault(key string, defaultValue string) string {
	v, ok := tm.m[key]
	if !ok {
		return defaultValue
	}
	return v
}

// TagSliceToMap takes a slice of *ec2.Tags and returns a mapping of their
// key -> value.
func TagSliceToMap(tags []*ec2.Tag) TagMap {
	m := make(map[string]string)
	for _, t := range tags {
		m[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}
	return TagMap{m: m}
}
