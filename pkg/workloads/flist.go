// Package workloads includes workloads types (vm, zdb, qsfs, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FlistChecksumURL returns flist check sum url format
func FlistChecksumURL(url string) string {
	return fmt.Sprintf("%s.md5", url)
}

// GetFlistChecksum gets flist check sum url
func GetFlistChecksum(url string) (string, error) {
	response, err := http.Get(FlistChecksumURL(url))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	hash, err := io.ReadAll(response.Body)
	return strings.TrimSpace(string(hash)), err
}
