package hover

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	neturl "net/url"
	"time"
)

func NewClient(cfg *Config, verbose bool) (*Client, error) {
	jar, _ := cookiejar.New(nil)
	c := &Client{
		http:        &http.Client{Jar: jar, Timeout: 15 * time.Second},
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
			return nil, fmt.Errorf("rate limited (HTTP 429), please wait before retrying")
		}
		// session expired or rejected — fall through to full login
		jar.SetCookies(u, []*http.Cookie{{Name: "hoverauth", Value: "", MaxAge: -1}})
	}

	if err := c.login(cfg); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	return c, nil
}

func (c *Client) login(cfg *Config) error {
	u, _ := neturl.Parse(c.baseURL)

	resp, err := c.http.Get(c.baseURL + "/signin")
	if err != nil {
		return fmt.Errorf("step 1: %w", err)
	}
	resp.Body.Close()

	loginPayload, err := json.Marshal(map[string]any{
		"username": cfg.Username,
		"password": cfg.Password,
		"token":    nil,
	})
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}
	resp, err = c.http.Post(
		c.baseURL+"/signin/auth.json",
		"application/json;charset=UTF-8",
		bytes.NewBuffer(loginPayload),
	)
	if err != nil {
		return fmt.Errorf("step 2: %w", err)
	}
	if resp.StatusCode == 429 {
		resp.Body.Close()
		return fmt.Errorf("rate limited (HTTP 429), please wait before retrying")
	}

	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("reading auth response: %w", err)
	}

	var authResp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(raw, &authResp); err != nil {
		log.Printf("warning: could not parse auth response: %v", err)
	}

	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "hoverauth" {
			c.saveAuthCookie(u)
			return nil
		}
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
	resp, err = c.http.Post(
		c.baseURL+"/signin/auth2.json",
		"application/json;charset=UTF-8",
		bytes.NewBuffer(codePayload),
	)
	if err != nil {
		return fmt.Errorf("step 3 (2FA): %w", err)
	}
	resp.Body.Close()

	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "hoverauth" {
			c.saveAuthCookie(u)
			return nil
		}
	}
	return fmt.Errorf("2FA failed: no auth cookie returned")
}
