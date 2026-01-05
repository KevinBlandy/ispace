package api

import (
	"ispace/common/response"

	"github.com/gin-gonic/gin"
)

type SignInApiRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type SignInApi struct{}

func NewSignInApi() *SignInApi {
	return &SignInApi{}
}

func (s SignInApi) Serve(ctx *gin.Context) (any, error) {
	var request = &SignInApiRequest{}
	if err := ctx.ShouldBindBodyWithJSON(request); err != nil {
		return nil, err
	}
	return response.Ok(request), nil
}
