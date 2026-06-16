package util

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func ExternalIP() (string, error) {
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://checkip.amazonaws.com",
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, svc := range services {
		resp, err := client.Get(svc)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		if ip := strings.TrimSpace(string(body)); ip != "" {
			return ip, nil
		}
	}
	return "", fmt.Errorf("all IP lookup services failed")
}
