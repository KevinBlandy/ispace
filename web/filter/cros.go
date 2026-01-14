package filter

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Cors(context *gin.Context) {
	origin := context.GetHeader("Origin")
	if origin != "" {
		context.Header("Access-Control-Allow-Origin", origin) // TODO 限制Origin
		requestHeader := context.GetHeader("Access-Control-Request-Headers")
		if requestHeader != "" {
			context.Header("Access-Control-Allow-Headers", requestHeader)
		}
		context.Header("Access-Control-Allow-Credentials", "true")
		context.Header("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS, TRACE")
		context.Header("Access-Control-Expose-Headers", "*")
		context.Header("Access-Control-Max-Age", "3000")

		if context.Request.Method == http.MethodOptions {
			context.AbortWithStatus(http.StatusNoContent)
			return
		}
	}
	context.Next()
}
