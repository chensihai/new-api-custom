package middleware

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	serverAddr := system_setting.ServerAddress
	if serverAddr != "" && (strings.HasPrefix(serverAddr, "https://") || strings.HasPrefix(serverAddr, "http://")) {
		origins := []string{serverAddr}
		u := strings.TrimPrefix(serverAddr, "https://")
		u = strings.TrimPrefix(u, "http://")
		if strings.HasPrefix(u, "localhost:") {
			origins = append(origins, "http://127.0.0.1:"+strings.TrimPrefix(u, "localhost:"))
		}
		config.AllowOrigins = origins
		config.AllowCredentials = true
	} else {
		config.AllowAllOrigins = true
		config.AllowCredentials = false
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	return cors.New(config)
}

func PoweredBy() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-New-Api-Version", common.Version)
		c.Next()
	}
}
