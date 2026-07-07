package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const contextKeyRetailGrokKey = "retail_grok_key"

func NewRetailGrokKeyAuthMiddleware(retailService *service.RetailGrokService) RetailGrokKeyAuthMiddleware {
	return RetailGrokKeyAuthMiddleware(func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		if raw == "" || !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			response.Unauthorized(c, "Retail grok API key required")
			c.Abort()
			return
		}
		keyValue := strings.TrimSpace(raw[len("Bearer "):])
		allowUsage := c.Request.URL.Path == "/retail/v1/usage" ||
			c.Request.URL.Path == "/retail/v1/models" ||
			c.Request.URL.Path == "/api/v1/retail-grok/usage" ||
			(c.Request.Method == "GET" && strings.HasPrefix(c.Request.URL.Path, "/retail/v1/videos/"))
		retailKey, err := retailService.Authenticate(c.Request.Context(), keyValue, allowUsage)
		if err != nil {
			response.ErrorFrom(c, err)
			c.Abort()
			return
		}
		c.Set(contextKeyRetailGrokKey, retailKey)
		c.Next()
	})
}

func GetRetailGrokKeyFromContext(c *gin.Context) (*service.RetailGrokKey, bool) {
	value, exists := c.Get(contextKeyRetailGrokKey)
	if !exists {
		return nil, false
	}
	key, ok := value.(*service.RetailGrokKey)
	return key, ok
}
