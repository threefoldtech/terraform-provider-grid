package workloads

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func FlistChecksumURL(url string) string {
	return fmt.Sprintf("%s.md5", url)
}
func GetFlistChecksum(url string) (string, error) {
	response, err := http.Get(FlistChecksumURL(url))
	if err != nil {
		return "", err
	}
	hash, err := io.ReadAll(response.Body)
	return strings.TrimSpace(string(hash)), err
}
