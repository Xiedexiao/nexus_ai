package middleware

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

// SliderCaptchaCheck enforces a freshly verified slider-puzzle pass token on
// self-service routes. The token is issued by POST /api/captcha/slider/verify,
// is HMAC-signed and IP-bound, and expires in minutes. When the captcha is
// disabled the middleware is transparent.
func SliderCaptchaCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.SliderCaptchaEnabled {
			c.Next()
			return
		}
		token := c.Query("slider")
		if token == "" {
			// also accept the header for POST bodies without query strings
			token = c.GetHeader("X-Slider-Captcha")
		}
		if token == "" {
			common.ApiErrorT(c, "Please complete the slider captcha first")
			c.Abort()
			return
		}
		if !common.ValidateSliderPassToken(token, c.ClientIP()) {
			common.ApiErrorT(c, "Slider captcha expired, please verify again")
			c.Abort()
			return
		}
		c.Next()
	}
}
