package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func MethodNotAllowed(c *gin.Context) {
	slog.WarnContext(c.Request.Context(),
		"MethodNotAllowed",
		slog.String("path", c.Request.URL.Path),
		slog.String("method", c.Request.Method),
		slog.String("remote", c.RemoteIP()),
		slog.String("user-agent", c.Request.UserAgent()),
	)
	c.AbortWithStatus(http.StatusMethodNotAllowed)
}
