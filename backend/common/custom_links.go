package common

import (
	"encoding/base64"
	"net/url"
	"sync"
)

// CustomLink is one admin-configured sidebar entry. When Proxy is true the
// user opens a same-origin route (/ext/<id>/) and the gateway reverse-proxies
// to Target, so the real address never appears in the browser; otherwise the
// sidebar item is a plain external link.
type CustomLink struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Target  string `json:"target"`
	Icon    string `json:"icon,omitempty"`
	Proxy   bool   `json:"proxy"`
	Enabled bool   `json:"enabled"`
}

var (
	customLinksMu     sync.RWMutex
	customLinksParsed []CustomLink
)

// SetCustomLinks caches the parsed links after an option update.
func SetCustomLinks(raw string) {
	var links []CustomLink
	if err := UnmarshalJsonStr(raw, &links); err != nil {
		links = nil
	}
	valid := make([]CustomLink, 0, len(links))
	for _, l := range links {
		if l.ID == "" || l.Title == "" || l.Target == "" {
			continue
		}
		if u, err := url.Parse(l.Target); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		valid = append(valid, l)
	}
	customLinksMu.Lock()
	customLinksParsed = valid
	customLinksMu.Unlock()
}

// GetCustomLinks returns the currently configured links for the console.
func GetCustomLinks() []CustomLink {
	customLinksMu.RLock()
	defer customLinksMu.RUnlock()
	out := make([]CustomLink, len(customLinksParsed))
	copy(out, customLinksParsed)
	return out
}

// FindCustomLink resolves a proxy id to its link definition; only enabled
// proxy links are servable through the same-origin proxy route.
func FindCustomLink(id string) (CustomLink, bool) {
	customLinksMu.RLock()
	defer customLinksMu.RUnlock()
	for _, l := range customLinksParsed {
		if l.ID == id && l.Enabled && l.Proxy {
			return l, true
		}
	}
	return CustomLink{}, false
}

// EncodeProxyPath packs the target host so the proxy route can resolve the
// upstream without exposing it in any client-visible markup.
func EncodeProxyTarget(target string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(target))
}
