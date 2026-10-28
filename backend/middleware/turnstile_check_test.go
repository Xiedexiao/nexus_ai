package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type turnstileFixture struct {
	rec  *httptest.ResponseRecorder
	ran  bool
	abrt bool
}

// runTurnstileWithServer drives the middleware against a fake Cloudflare
// siteverify endpoint so success, failure, and timeout paths are all
// deterministic. It returns the response recorder and whether the handler
// chain continued past the middleware.
func runTurnstileWithServer(t *testing.T, backend *httptest.Server, token string) *turnstileFixture {
	t.Helper()

	originalEnabled := common.TurnstileCheckEnabled
	originalSecret := common.TurnstileSecretKey
	originalVerifyURL := turnstileVerifyURL
	common.TurnstileCheckEnabled = true
	common.TurnstileSecretKey = "test-secret"
	turnstileVerifyURL = backend.URL
	t.Cleanup(func() {
		common.TurnstileCheckEnabled = originalEnabled
		common.TurnstileSecretKey = originalSecret
		turnstileVerifyURL = originalVerifyURL
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	var ran, abrt bool
	router.GET("/probe", TurnstileCheck(), func(c *gin.Context) {
		ran = true
		c.Status(http.StatusOK)
	})

	q := ""
	if token != "" {
		q = "?turnstile=" + url.QueryEscape(token)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe"+q, nil)
	req.RemoteAddr = "203.0.113.50:1234"
	router.ServeHTTP(rec, req)
	return &turnstileFixture{rec: rec, ran: ran, abrt: abrt}
}

func TestTurnstileCheckAcceptsValidToken(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "test-secret", r.PostFormValue("secret"))
		assert.Equal(t, "good-token", r.PostFormValue("response"))
		assert.Equal(t, "203.0.113.50", r.PostFormValue("remoteip"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer backend.Close()

	fx := runTurnstileWithServer(t, backend, "good-token")
	assert.Equal(t, http.StatusOK, fx.rec.Code)
	assert.True(t, fx.ran, "the chain must continue after a valid token")
}

func TestTurnstileCheckRejectsMissingToken(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("siteverify must not be called without a token")
	}))
	defer backend.Close()

	fx := runTurnstileWithServer(t, backend, "")
	assert.Equal(t, http.StatusOK, fx.rec.Code)
	assert.False(t, fx.ran, "the chain must stop on an empty token")
	assert.Contains(t, fx.rec.Body.String(), "Turnstile token is empty")
}

func TestTurnstileCheckRejectsFailedVerification(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer backend.Close()

	fx := runTurnstileWithServer(t, backend, "bad-token")
	assert.False(t, fx.ran, "the chain must stop on a failed verification")
	assert.Contains(t, fx.rec.Body.String(), "Turnstile verification failed")
}

func TestTurnstileCheckFailsClosedOnBackendTimeout(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(turnstileVerifyTimeout + 3*time.Second)
	}))
	defer backend.Close()

	fx := runTurnstileWithServer(t, backend, "slow-token")
	assert.False(t, fx.ran, "an unreachable verifier must not let the request through")
	assert.Contains(t, fx.rec.Body.String(), "verification service")
}

func TestTurnstileCheckFailsClosedOnOversizedBody(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"junk":"` + strings.Repeat("x", turnstileMaxResponseBytes) + `"}`))
	}))
	defer backend.Close()

	fx := runTurnstileWithServer(t, backend, "big-token")
	assert.False(t, fx.ran, "an oversized siteverify reply must not pass")
}
