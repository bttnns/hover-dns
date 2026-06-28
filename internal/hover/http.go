package hover

import (
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"os"
)

func (c *Client) doRequestOnce(method, urlStr string, payload []byte) ([]byte, int, error) {
	req := c.rc.R().SetHeader("X-Requested-With", "XMLHttpRequest")
	if payload != nil {
		req.SetHeader("Content-Type", "application/json;charset=UTF-8").SetBody(payload)
	}

	if c.verbose {
		log.Printf("DEBUG %s %s payload=%s", method, urlStr, payload)
	}

	resp, err := req.Execute(method, urlStr)
	if err != nil {
		return nil, 0, err
	}
	body := resp.Body()

	if c.verbose {
		log.Printf("DEBUG response (HTTP %d): %s", resp.StatusCode(), body)
	}

	return body, resp.StatusCode(), nil
}

func (c *Client) doRequest(method, urlStr string, payload []byte) ([]byte, int, error) {
	body, status, err := c.doRequestOnce(method, urlStr, payload)
	if err != nil || status != 401 {
		return body, status, err
	}

	// 401: the session expired. Serialize re-login through loginMu so concurrent
	// callers (the ddns loop and API handlers share one Client) don't stampede
	// Hover's auth endpoint or race on the session file.
	c.loginMu.Lock()
	defer c.loginMu.Unlock()

	// Another goroutine may have re-logged in while we waited for the lock; retry
	// once before forcing a fresh login.
	body, status, err = c.doRequestOnce(method, urlStr, payload)
	if err != nil || status != 401 {
		return body, status, err
	}

	// Still 401: clear stale session and re-login, then retry once.
	os.Remove(c.sessionFile)
	u, _ := neturl.Parse(c.baseURL)
	c.jar().SetCookies(u, []*http.Cookie{{Name: "hoverauth", MaxAge: -1}})
	if loginErr := c.login(c.cfg); loginErr != nil {
		return nil, status, fmt.Errorf("session expired, re-login failed: %w", loginErr)
	}
	return c.doRequestOnce(method, urlStr, payload)
}

// jar returns the cookie jar resty uses for requests, so the auth/session code
// and the 401 handler all read and mutate the same cookies.
func (c *Client) jar() http.CookieJar {
	return c.rc.GetClient().Jar
}
