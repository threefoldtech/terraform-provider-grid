// Package provider is the terraform provider
package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceContains(t *testing.T) {

	stringSlice := []string{"a", "b", "c"}
	intSlice := []int{1, 2, 3}
	byteSlice := []byte("test bytes")

	t.Run("test_valid_string_slice", func(t *testing.T) {
		element := "a"
		assert.True(t, Contains(stringSlice, element))
	})

	t.Run("test_invalid_string_slice", func(t *testing.T) {
		element := "d"
		assert.False(t, Contains(stringSlice, element))
	})

	t.Run("test_valid_int_slice", func(t *testing.T) {
		element := 1
		assert.True(t, Contains(intSlice, element))
	})

	t.Run("test_invalid_int_slice", func(t *testing.T) {
		element := 0
		assert.False(t, Contains(intSlice, element))
	})

	t.Run("test_valid_byte_slice", func(t *testing.T) {
		element := []byte("test bytes")[5]
		assert.True(t, Contains(byteSlice, element))
	})

	t.Run("test_invalid_byte_slice", func(t *testing.T) {
		element := []byte("w")[0]
		assert.False(t, Contains(byteSlice, element))
	})
}
