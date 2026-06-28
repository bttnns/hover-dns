package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// ipClient is reused across calls so the transport and connection pool survive
// between DDNS interval checks.
var ipClient = resty.New().SetTimeout(5 * time.Second)

func ExternalIP() (string, error) {
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://checkip.amazonaws.com",
	}
	for _, svc := range services {
		resp, err := ipClient.R().Get(svc)
		if err != nil {
			continue
		}
		// resty does not treat non-2xx as an error, so skip error responses
		// rather than returning an error page body as the IP.
		if !resp.IsSuccess() {
			continue
		}
		if ip := strings.TrimSpace(resp.String()); ip != "" {
			return ip, nil
		}
	}
	return "", fmt.Errorf("all IP lookup services failed")
}
