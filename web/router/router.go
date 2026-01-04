package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func New() http.Handler {
	router := gin.New()
	router.RedirectTrailingSlash = false

	return router
}
