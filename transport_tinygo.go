//go:build tinygo

package ecosystems

import "net/http"

func defaultTransport() *http.Transport {
	return &http.Transport{}
}
