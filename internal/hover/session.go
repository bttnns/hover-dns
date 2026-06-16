package hover

import (
	"encoding/json"
	"os"
	"time"

	neturl "net/url"
)

func loadSession(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var s savedSession
	if err := json.Unmarshal(data, &s); err != nil || time.Now().After(s.Expires) {
		return ""
	}
	return s.Value
}

func saveSession(path, value string, expires time.Time) {
	data, _ := json.Marshal(savedSession{Value: value, Expires: expires})
	_ = os.WriteFile(path, data, 0600)
}

func (c *Client) saveAuthCookie(u *neturl.URL) {
	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "hoverauth" {
			exp := cookie.Expires
			if exp.IsZero() {
				exp = time.Now().Add(24 * time.Hour)
			}
			saveSession(c.sessionFile, cookie.Value, exp)
			return
		}
	}
}
