package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runSliderCheck(t *testing.T, enabled bool, token, ip string) *httptest.ResponseRecorder {
	t.Helper()
	original := common.SliderCaptchaEnabled
	common.SliderCaptchaEnabled = enabled
	t.Cleanup(func() { common.SliderCaptchaEnabled = original })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	var ran bool
	router.GET("/probe", SliderCaptchaCheck(), func(c *gin.Context) { ran = true; c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if token != "" {
		req.Header.Set("X-Slider-Captcha", token)
	}
	req.RemoteAddr = ip + ":5555"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.False(t, ran && rec.Code != http.StatusOK, "handler must not run on rejection")
	return rec
}

func TestSliderCaptchaCheckTransparentWhenDisabled(t *testing.T) {
	rec := runSliderCheck(t, false, "", "10.0.0.1")
	assert.Equal(t, http.StatusOK, rec.Code, "disabled captcha must not guard routes")
}

func TestSliderCaptchaCheckRejectsMissingToken(t *testing.T) {
	rec := runSliderCheck(t, true, "", "10.0.0.2")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "slider captcha")
}

func TestSliderCaptchaCheckAcceptsFreshToken(t *testing.T) {
	token := common.IssueSliderPassToken("10.0.0.3")
	rec := runSliderCheck(t, true, token, "10.0.0.3")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSliderCaptchaCheckBindsTokenToIP(t *testing.T) {
	token := common.IssueSliderPassToken("10.0.0.4")
	rec := runSliderCheck(t, true, token, "10.0.0.99")
	assert.Contains(t, rec.Body.String(), "expired")
}

func TestSliderCaptchaCheckRejectsGarbage(t *testing.T) {
	rec := runSliderCheck(t, true, "not-a-token", "10.0.0.5")
	assert.Contains(t, rec.Body.String(), "expired")
}
