package backend

import (
	"fmt"

	"github.com/free-ran-ue/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

func authticate(c *gin.Context, jwtSecret string) error {
	authenticateHeader := c.GetHeader("Authorization")
	if authenticateHeader == "" {
		return fmt.Errorf("no authentication header")
	}

	if _, err := util.ValidateJWT(authenticateHeader, jwtSecret); err != nil {
		return fmt.Errorf("failed to validate JWT: %v", err)
	}

	return nil
}
