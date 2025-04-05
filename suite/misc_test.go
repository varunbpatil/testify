package suite_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/varunbpatil/testify/suite"
)

type TestStruct struct {
	Field1 string
	Field2 int
}

func TestCreateAndSetField(t *testing.T) {
	// Test setting a valid field
	input := &TestStruct{}
	err := suite.SetField(input, "Field1", "test value")
	assert.NoError(t, err)
	assert.Equal(t, "test value", input.Field1)

	// Test setting an invalid field
	err = suite.SetField(input, "NonExistentField", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field NonExistentField does not exist")

	// Test setting an incompatible value
	err = suite.SetField(input, "Field2", "invalid type")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value of type string is not assignable to field Field2 of type int")
}
