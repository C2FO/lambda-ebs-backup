package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testVar = "TEST_VAR"

var trueValues = []string{"True", "true", "TRUE", "TrUe", "tRUE"}
var falseVaues = []string{"False", "false", "FaLSE"}

func TestEnvDefaultBool(t *testing.T) {
	var before = os.Getenv(testVar)
	defer func() { os.Setenv(testVar, before) }()

	// Test the default works
	assert.True(t, envDefaultBool(testVar, true))
	assert.False(t, envDefaultBool(testVar, false))

	// Test the true case with opposite default
	for _, v := range trueValues {
		os.Setenv(testVar, v)
		assert.True(t, envDefaultBool(testVar, false))
	}

	// Test the false case with opposite default
	for _, v := range falseVaues {
		os.Setenv(testVar, v)
		assert.False(t, envDefaultBool(testVar, true))
	}

	// Test the true case with same default
	os.Setenv(testVar, "True")
}
