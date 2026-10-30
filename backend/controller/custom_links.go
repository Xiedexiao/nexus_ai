package controller

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// CustomLinksStatus returns the admin-configured sidebar links for the
// console. Proxy targets are never included: the client only receives ids.
func GetCustomLinksForConsole(c *gin.Context) {
	links := common.GetCustomLinks()
	type consoleLink struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Icon    string `json:"icon,omitempty"`
		Proxy   bool   `json:"proxy"`
		Enabled bool   `json:"enabled"`
		Href    string `json:"href,omitempty"`
	}
	out := make([]consoleLink, 0, len(links))
	for _, l := range links {
		item := consoleLink{
			ID:      l.ID,
			Title:   l.Title,
			Icon:    l.Icon,
			Proxy:   l.Proxy,
			Enabled: l.Enabled,
		}
		if !l.Proxy {
			item.Href = l.Target
		}
		out = append(out, item)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

// ProxyCustomLink serves /ext/:id/*target through a same-origin reverse
// proxy, so the browser address bar and page markup only ever show this
// site's own path. Only enabled proxy links resolve; every request is
// re-resolved from the in-memory configuration, so disabling a link takes
// effect immediately without a restart.
func ProxyCustomLink(c *gin.Context) {
	id := c.Param("id")
	link, ok := common.FindCustomLink(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Unknown or disabled link",
		})
		return
	}
	target, err := url.Parse(link.Target)
	if err != nil || target.Scheme == "" || target.Host == "" {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "Invalid proxy target"})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		// Keep the origin headers aligned with the upstream so cookie domains
		// and redirects behave like a direct visit.
		req.Header.Set("X-Forwarded-Host", c.Request.Host)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		common.SysError(common.LogText("custom link proxy %s -> %s failed: %s", id, link.Target, err.Error()))
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream unavailable"))
	}
	// The reverse proxy handles everything, including WebSocket upgrades on
	// modern Go versions; strip this route from the SPA fallback chain.
	c.Header("X-Custom-Link", id)
	proxy.ServeHTTP(c.Writer, c.Request)
}
