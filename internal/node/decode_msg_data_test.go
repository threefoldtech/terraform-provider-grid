package client

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeMsgData(t *testing.T) {
	s1 := "not base64 encoded"
	s2Before := "to be base64 encoded"
	s2After := base64.StdEncoding.EncodeToString([]byte(s2Before))

	ret1 := getDecodedMsgData(s1)
	assert.Equal(t, s1, string(ret1))

	ret2 := getDecodedMsgData(s2After)
	assert.Equal(t, s2Before, string(ret2))

}
