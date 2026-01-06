package web

// SignInApiRequest 登录
type SignInApiRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}
