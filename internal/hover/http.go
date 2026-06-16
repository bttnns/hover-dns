package hover

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
)

func (c *Client) doRequestOnce(method, urlStr string, payload []byte) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewBuffer(payload)
	}
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, 0, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	if c.verbose {
		log.Printf("DEBUG %s %s payload=%s", method, urlStr, payload)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	if c.verbose {
		log.Printf("DEBUG response (HTTP %d): %s", resp.StatusCode, respBody)
	}

	return respBody, resp.StatusCode, nil
}

func (c *Client) doRequest(method, urlStr string, payload []byte) ([]byte, int, error) {
	body, status, err := c.doRequestOnce(method, urlStr, payload)
	if err != nil || status != 401 {
		return body, status, err
	}
	// 401: clear stale session, re-login, retry once
	os.Remove(c.sessionFile)
	u, _ := neturl.Parse(c.baseURL)
	c.http.Jar.SetCookies(u, []*http.Cookie{{Name: "hoverauth", MaxAge: -1}})
	if loginErr := c.login(c.cfg); loginErr != nil {
		return nil, status, fmt.Errorf("session expired, re-login failed: %w", loginErr)
	}
	return c.doRequestOnce(method, urlStr, payload)
}
