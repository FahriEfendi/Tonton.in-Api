package models

type Respons struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Users struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     int    `json:"role"`
	Token    string `json:"token"`
}
