package client

import (
	"encoding/base64"
)

// TryDecodeBase64OrElse tries to decode a possibly base64 encoded string into a byte array or returns the string as a byte array assuming it was decoded already
func TryDecodeBase64OrElse(possiblyEncoded string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(possiblyEncoded)
	if err != nil {
		return []byte(possiblyEncoded)
	}
	return decoded
}
