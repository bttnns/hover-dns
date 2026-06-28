package hover

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	neturl "net/url"
	"time"

	"github.com/go-resty/resty/v2"
)

func NewClient(cfg *Config, verbose bool) (*Client, error) {
	jar, _ := cookiejar.New(nil)
	rc := resty.New().
		SetTimeout(15 * time.Second).
		SetCookieJar(jar)

	c := &Client{
		rc:          rc,
		baseURL:     "https://www.hover.com",
		verbose:     verbose,
		sessionFile: cfg.SessionFile,
		cfg:         cfg,
	}

	u, _ := neturl.Parse(c.baseURL)
	if tok := loadSession(cfg.SessionFile); tok != "" {
		jar.SetCookies(u, []*http.Cookie{{Name: "hoverauth", Value: tok}})
		_, status, err := c.doRequest("GET", c.baseURL+"/api/domains", nil)
		if err == nil && status == 200 {
			return c, nil
		}
		if status == 429 {
			return nil, fmt.Errorf("rate limited (HTTP 429), please wait before retrying: %w", ErrRateLimit)
		}
		// session expired or rejected, fall through to full login
		jar.SetCookies(u, []*http.Cookie{{Name: "hoverauth", Value: "", MaxAge: -1}})
	}

	if err := c.login(cfg); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	return c, nil
}

func (c *Client) login(cfg *Config) error {
	u, _ := neturl.Parse(c.baseURL)

	if _, err := c.rc.R().Get(c.baseURL + "/signin"); err != nil {
		return fmt.Errorf("step 1: %w", err)
	}

	loginPayload, err := json.Marshal(map[string]any{
		"username": cfg.Username,
		"password": cfg.Password,
		"token":    nil,
	})
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}
	resp, err := c.rc.R().
		SetHeader("Content-Type", "application/json;charset=UTF-8").
		SetBody(loginPayload).
		Post(c.baseURL + "/signin/auth.json")
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}
	if resp.StatusCode() == 429 {
		return fmt.Errorf("rate limited (HTTP 429), please wait before retrying: %w", ErrRateLimit)
	}
	raw := resp.Body()

	var authResp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(raw, &authResp); err != nil {
		log.Printf("warning: could not parse auth response: %v", err)
	}

	if c.hasAuthCookie(u) {
		c.saveAuthCookie(u)
		return nil
	}

	if authResp.Status != "need_2fa" {
		msg := authResp.Error
		if msg == "" {
			msg = string(raw)
		}
		return fmt.Errorf("credentials rejected: %s", msg)
	}

	if cfg.TOTPSecret == "" {
		return fmt.Errorf("2FA required but totp_secret is not set in config")
	}
	code, err := totp(cfg.TOTPSecret)
	if err != nil {
		return fmt.Errorf("generating TOTP: %w", err)
	}

	codePayload, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return fmt.Errorf("marshaling 2FA code: %w", err)
	}
	resp, err = c.rc.R().
		SetHeader("Content-Type", "application/json;charset=UTF-8").
		SetBody(codePayload).
		Post(c.baseURL + "/signin/auth2.json")
	if err != nil {
		return fmt.Errorf("step 3 (2FA): %w", err)
	}
	if resp.StatusCode() == 429 {
		return fmt.Errorf("rate limited (HTTP 429), please wait before retrying: %w", ErrRateLimit)
	}

	if c.hasAuthCookie(u) {
		c.saveAuthCookie(u)
		return nil
	}
	return fmt.Errorf("2FA failed: no auth cookie returned")
}

func (c *Client) hasAuthCookie(u *neturl.URL) bool {
	for _, cookie := range c.jar().Cookies(u) {
		if cookie.Name == "hoverauth" {
			return true
		}
	}
	return false
}
