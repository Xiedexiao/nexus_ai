package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// CheckRegistrationEmail reports whether an email can start a registration.
//
// Enumeration trade-off: a plain "taken / free" answer would let attackers
// probe which addresses hold accounts, so the endpoint is only reachable
// behind the same slider captcha + rate limits as the send-code route it
// preflights, and its answer gates exactly the action the caller already sees
// (send code → "already in use"). The frontend additionally only reveals the
// "sign in instead" hint after a real send-code rejection, so the probe adds
// no information an attacker could not get by calling send-code directly.
func CheckRegistrationEmail(c *gin.Context) {
	email, err := service.ValidateAccountEmail(c.Query("email"))
	if err != nil {
		writeSecurityOperationError(c, err)
		return
	}
	taken := model.IsEmailAlreadyTaken(email)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"email": email,
			"taken": taken,
		},
	})
}
