package model

type ConsoleLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ConsoleLoginResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

type ConsoleLogoutResponse struct {
	Message string `json:"message"`
}
