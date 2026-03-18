package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yoophi/codepush-server-golang/internal/application"
	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

const accountContextKey = "account"

func RequireAuth(service *application.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimSpace(header[7:])
		account, err := service.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(application.HTTPStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.Set(accountContextKey, account)
		c.Next()
	}
}

func CurrentAccount(c *gin.Context) domain.Account {
	value, _ := c.Get(accountContextKey)
	account, _ := value.(domain.Account)
	return account
}
