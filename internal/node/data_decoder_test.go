package client

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestTryDecodeBase64OrElse(t *testing.T) {
	testStrings := []string{"hello world", "bye world"}

	plainTextStrings := testStrings[:]
	t.Log("Given plain text input string")
	{
		for i, s := range plainTextStrings {
			t.Run(fmt.Sprintf("\tTest %d: plaintext string: When string is a plaintext string `%s`", i, s), func(t *testing.T) {
				result := TryDecodeBase64OrElse(s)
				if string(result) == s {
					t.Logf("\t\tResult `%s` should be equal to the original plaintext string `%s`", result, s)
				} else {
					t.Errorf("\tTest %d: failed result %s not equal to plain text input string `%s`", i, result, s)
				}

			})
		}
	}
	var encodedBase64Strings []string
	for _, s := range testStrings {
		encodedBase64Strings = append(encodedBase64Strings, base64.StdEncoding.EncodeToString([]byte(s)))
	}

	t.Log("Given encoded base64 string")
	{
		for i, s := range encodedBase64Strings {
			t.Run(fmt.Sprintf("\tTest %d: When string is base64 encoded string `%s`", i, s), func(t *testing.T) {
				result := TryDecodeBase64OrElse(s)
				if string(result) == testStrings[i] {
					t.Logf("\t\tResult `%s` should be equal to the original plaintext string `%s`", result, testStrings[i])
				} else {
					t.Errorf("\tTest %d: failed decoding %s result is  %s not equal to the original plain text input string `%s`", i, s, result, testStrings[i])
				}
			})
		}
	}
}
