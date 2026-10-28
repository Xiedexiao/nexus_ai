package middleware

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// turnstileVerifyURL is a variable so tests can point the middleware at a
// stub verifier instead of Cloudflare.
var turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// turnstileVerifyTimeout bounds the outbound siteverify call. Without it a
// hung challenge endpoint would park every registration goroutine indefinitely.
const turnstileVerifyTimeout = 5 * time.Second

// turnstileMaxResponseBytes bounds the siteverify reply read. The success
// payload is a few hundred bytes; anything far beyond that is not Cloudflare.
const turnstileMaxResponseBytes = 64 * 1024

// turnstileHTTPClient is a dedicated client with a global timeout, so the
// verifier call never inherits long-lived relay transport settings.
var turnstileHTTPClient = &http.Client{Timeout: turnstileVerifyTimeout}

type turnstileCheckResponse struct {
	Success bool `json:"success"`
}

func TurnstileCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.TurnstileCheckEnabled {
			response := c.Query("turnstile")
			if response == "" {
				common.ApiErrorT(c, "Turnstile token is empty")
				c.Abort()
				return
			}
			form := url.Values{
				"secret":   {common.TurnstileSecretKey},
				"response": {response},
				"remoteip": {c.ClientIP()},
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), turnstileVerifyTimeout)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(form.Encode()))
			if err != nil {
				common.SysError(common.LogText("turnstile siteverify request build failed: %s", err.Error()))
				common.ApiErrorT(c, "Turnstile verification failed, please refresh and try again!")
				c.Abort()
				return
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rawRes, err := turnstileHTTPClient.Do(req)
			if err != nil {
				// Fail closed: a verifier that cannot be reached must never
				// default to letting the request through.
				common.SysError(common.LogText("turnstile siteverify unreachable: %s", err.Error()))
				common.ApiErrorT(c, "Turnstile verification service is unavailable, please try again later!")
				c.Abort()
				return
			}
			defer rawRes.Body.Close()
			var res turnstileCheckResponse
			if err = common.DecodeJson(io.LimitReader(rawRes.Body, turnstileMaxResponseBytes), &res); err != nil {
				common.SysError(common.LogText("turnstile siteverify response invalid: %s", err.Error()))
				common.ApiErrorT(c, "Turnstile verification failed, please refresh and try again!")
				c.Abort()
				return
			}
			if !res.Success {
				common.ApiErrorT(c, "Turnstile verification failed, please refresh and try again!")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
