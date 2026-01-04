package server

import (
	"fmt"
	"ispace/common/types"
	"ispace/config"
	"ispace/web/router"
	"net/http"
)

func New() *http.Server {
	server := http.Server{
		Addr:           fmt.Sprintf("%s:%d", *config.HttpHost, *config.HttpPort),
		MaxHeaderBytes: int(types.KB * 4), // 最大4Kb
		Handler:        router.New(),
	}
	return &server
}
