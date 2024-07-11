package models

type Respons struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Users struct {
	ID       string `json:"id"`
	Nama     string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Token    string `json:"token"`
}

type Admin struct {
	Nama     string `json:"username"`
	Password string `json:"password"`
}
