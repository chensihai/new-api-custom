package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func RealNameCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.PhoneAuthForceRealNameAuth {
			c.Next()
			return
		}

		if !common.PhoneLoginEnabled {
			c.Next()
			return
		}

		userId, exists := c.Get("id")
		if !exists {
			c.Next()
			return
		}

		id, ok := userId.(int)
		if !ok {
			c.Next()
			return
		}

		phoneBound := model.IsUserPhoneBound(id)
		if phoneBound {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "phone real name verification required",
		})
		c.Abort()
	}
}
