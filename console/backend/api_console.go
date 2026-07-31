package backend

import (
	"net/http"

	"github.com/free-ran-ue/free-ran-ue/v2/console/model"
	"github.com/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

func (cs *console) handleConsoleLogin(c *gin.Context) {
	var loginRequest model.ConsoleLoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		cs.LoginLog.Warnf("Failed to bind login request: %v", err)
		c.JSON(http.StatusBadRequest, model.ConsoleLoginResponse{
			Message: "Invalid request format",
		})
		return
	}

	if loginRequest.Username != cs.username || loginRequest.Password != cs.password {
		cs.LoginLog.Warnf("Invalid credentials")
		c.JSON(http.StatusUnauthorized, model.ConsoleLoginResponse{
			Message: "Invalid credentials",
		})
		return
	}

	token, err := util.CreateJWT(cs.jwt.secret, c.ClientIP(), cs.jwt.expiresIn, nil)
	if err != nil {
		cs.LoginLog.Errorf("Failed to create JWT: %v", err)
		c.JSON(http.StatusInternalServerError, model.ConsoleLoginResponse{
			Message: "Failed to create JWT",
		})
		return
	}

	c.JSON(http.StatusOK, model.ConsoleLoginResponse{
		Message: "Login successful",
		Token:   token,
	})
}

func (cs *console) handleConsoleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, model.ConsoleLogoutResponse{
		Message: "Logout successful",
	})
}

func (cs *console) handleAuthenticate(c *gin.Context) {
	authenticateHeader := c.GetHeader("Authorization")
	if authenticateHeader == "" {
		cs.AuthLog.Warnln("No authentication header")
		c.JSON(http.StatusUnauthorized, model.AuthenticateResponse{
			Message: "No authentication header",
		})
		return
	}

	if _, err := util.ValidateJWT(authenticateHeader, cs.jwt.secret); err != nil {
		cs.AuthLog.Warnf("Failed to validate JWT: %v", err)
		c.JSON(http.StatusUnauthorized, model.AuthenticateResponse{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.AuthenticateResponse{
		Message: "Authenticate successful",
	})
}
