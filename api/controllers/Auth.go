package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	DB *sql.DB
}

type Respon struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ResponLogin struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Token   string
}

type ResponError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type LoginResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Token   string `json:"token"`
}

type LoginAttempt struct {
	Count       int
	LastAttempt time.Time
}

var loginAttempts = make(map[string]*LoginAttempt)
var loginAttemptsMutex = &sync.Mutex{}

const maxAttempts = 3
const lockoutDuration = 1 * time.Minute

func NewAuthController(db *sql.DB) *AuthController {
	return &AuthController{DB: db}
}

func (uc *AuthController) Login(c echo.Context) error {
	var loginData struct {
		Username string `json:"nama"`
		Password string `json:"password"`
	}

	if err := c.Bind(&loginData); err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, "Permintaan Tidak Valid")
	}

	// Memeriksa percobaan login
	loginAttemptsMutex.Lock()
	attempt, exists := loginAttempts[loginData.Username]
	if exists && attempt.Count >= maxAttempts && time.Since(attempt.LastAttempt) < lockoutDuration {
		loginAttemptsMutex.Unlock()
		return c.JSON(http.StatusTooManyRequests, "Terlalu banyak percobaan login. Silakan coba lagi nanti.")
	}
	loginAttemptsMutex.Unlock()

	isAdminLogin := strings.Contains(loginData.Username, "@")
	var user struct {
		ID       string
		Nama     string
		Role     string
		Password string
	}

	if isAdminLogin {
		parts := strings.Split(loginData.Username, "@")
		if len(parts) != 2 {
			return c.JSON(http.StatusBadRequest, "Format username tidak valid")
		}
		adminName, actualUsername := parts[0], parts[1]

		var admin struct {
			Password string
		}
		err := uc.DB.QueryRow(`SELECT password FROM admin WHERE nama = ?`, adminName).Scan(&admin.Password)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusNotFound, "Admin tidak ditemukan")
			}
			log.Println(err)
			return c.JSON(http.StatusInternalServerError, "Kesalahan Internal Server")
		}

		err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(loginData.Password))
		if err != nil {
			log.Println(err)
			updateLoginAttempts(loginData.Username)
			return c.JSON(http.StatusUnauthorized, "Password Admin Salah!")
		}

		err = uc.DB.QueryRow(`SELECT id, nama, role, password FROM users WHERE nama = ?`, actualUsername).Scan(&user.ID, &user.Nama, &user.Role, &user.Password)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusNotFound, "User tidak ditemukan")
			}
			log.Println(err)
			return c.JSON(http.StatusInternalServerError, "Kesalahan Internal Server")
		}
	} else {
		err := uc.DB.QueryRow(`SELECT id, nama, role, password FROM users WHERE nama = ?`, loginData.Username).Scan(&user.ID, &user.Nama, &user.Role, &user.Password)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusNotFound, "User tidak ditemukan")
			}
			log.Println(err)
			return c.JSON(http.StatusInternalServerError, "Kesalahan Internal Server")
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password))
		if err != nil {
			log.Println(err)
			updateLoginAttempts(loginData.Username)
			return c.JSON(http.StatusUnauthorized, "Password Anda Salah!")
		}
	}

	// Mengatur ulang percobaan login pada login yang berhasil
	resetLoginAttempts(loginData.Username)

	claims := jwt.MapClaims{
		"userId": user.ID,
		"nama":   user.Nama,
		"role":   user.Role,
		"exp":    time.Now().Add(time.Minute * 2880).Unix(),
		"iat":    time.Now().Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessTokenString, err := accessToken.SignedString([]byte("2131fwhdfh56e2s7j8"))
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan Internal Server")
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refreshTokenString, err := refreshToken.SignedString([]byte("2131fwhdfh56e2s7j8"))
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan Internal Server")
	}

	_, err = uc.DB.Exec(`UPDATE users SET token = ? WHERE id = ?`, refreshTokenString, user.ID)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan Internal Server")
	}

	cookie := &http.Cookie{
		Name:     "token",
		Value:    refreshTokenString,
		HttpOnly: true,
		MaxAge:   24 * 60 * 60,
	}
	c.SetCookie(cookie)

	response := LoginResponse{
		Status:  "Berhasil",
		Message: fmt.Sprintf("Selamat Datang %s", user.Nama),
		Token:   accessTokenString,
	}

	return c.JSON(http.StatusOK, response)
}

func updateLoginAttempts(username string) {
	loginAttemptsMutex.Lock()
	defer loginAttemptsMutex.Unlock()

	if attempt, exists := loginAttempts[username]; exists {
		attempt.Count++
		attempt.LastAttempt = time.Now()
	} else {
		loginAttempts[username] = &LoginAttempt{Count: 1, LastAttempt: time.Now()}
	}
}

func resetLoginAttempts(username string) {
	loginAttemptsMutex.Lock()
	defer loginAttemptsMutex.Unlock()

	// Hapus percobaan login tanpa memeriksa keberadaannya
	delete(loginAttempts, username)
}

func (uc *AuthController) Register(c echo.Context) error {
	var registerData struct {
		Nama         string `json:"nama"`
		Password     string `json:"password"`
		ConfPassword string `json:"confPassword"`
		Role         string `json:"role"`
	}

	if err := c.Bind(&registerData); err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}

	if registerData.Password != registerData.ConfPassword {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"msg": "Password dan Confirm Password tidak cocok",
		})
	}

	var existingName string
	err := uc.DB.QueryRow(`SELECT nama FROM users WHERE nama = ?`, registerData.Nama).Scan(&existingName)
	if err != nil && err != sql.ErrNoRows {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	if existingName != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"msg": "Nama telah terdaftar",
		})
	}

	// Generate UUID baru
	uuidNew, err := uuid.NewRandom()
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, Respons{
			Status:  "Gagal",
			Message: "Gagal menghasilkan UUID",
		})
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(registerData.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	_, err = uc.DB.Exec(`INSERT INTO users (nama, password, role,uuid) VALUES (?, ?, ?, ?)`, registerData.Nama, hashPassword, registerData.Role, uuidNew.String())
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"msg": "User berhasil didaftarkan",
	})
}

func (uc *AuthController) Logout(c echo.Context) error {
	// Get the token from cookies
	token, err := c.Cookie("token")
	if err != nil {
		// If there's no token, return 204 No Content
		return c.NoContent(http.StatusNoContent)
	}

	// Find the user by token
	var user struct {
		ID int
	}
	err = uc.DB.QueryRow(`SELECT id FROM users WHERE token = ?`, token.Value).Scan(&user.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			// If no user found with the token, return 204 No Content
			return c.NoContent(http.StatusNoContent)
		}
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	// Update the user's token to null
	_, err = uc.DB.Exec(`UPDATE users SET token = NULL WHERE id = ?`, user.ID)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	// Clear the token cookie
	c.SetCookie(&http.Cookie{
		Name:   "token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	return c.NoContent(http.StatusOK)
}

func (uc *AuthController) Token(c echo.Context) error {
	// Get the token from cookies
	tokenCookie, err := c.Cookie("token")
	if err != nil {
		// If there's no token, return 401 Unauthorized
		return c.NoContent(http.StatusUnauthorized)
	}

	tokenString := tokenCookie.Value

	/* // Print the token string
	log.Println("Token String:", tokenString) */

	// Verify the token
	claims := &jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Replace the string key with your actual secret key
		return []byte("2131fwhdfh56e2s7j8"), nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			log.Println("Invalid token signature:", err)
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"message": "Mohon login ulang",
			})
		}
		log.Println("Error parsing token:", err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	if !token.Valid {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "Mohon login ulang",
		})
	}

	// Extract claims
	nama, ok := (*claims)["nama"].(string)
	if !ok {
		log.Println("Invalid token claims: missing 'nama'")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "Mohon login ulang",
		})
	}
	userId, ok := (*claims)["userId"].(string)
	if !ok {
		log.Println("Invalid token claims: missing 'userId'")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "Mohon login ulang",
		})
	}
	role, ok := (*claims)["role"].(string)
	if !ok {
		log.Println("Invalid token claims: missing 'role'")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"message": "Mohon login ulang",
		})
	}

	// Find the user by UUID in the database to ensure it exists
	var user struct {
		UUID string
	}
	err = uc.DB.QueryRow(`SELECT uuid FROM users WHERE token = ?`, tokenString).Scan(&user.UUID)
	if err != nil {
		if err == sql.ErrNoRows {
			// If no user found with the token, return 403 Forbidden
			return c.NoContent(http.StatusForbidden)
		}
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	// Generate a new access token
	accessTokenClaims := jwt.MapClaims{
		"nama":   nama,
		"userId": userId,
		"role":   role,
		"exp":    time.Now().Add(15 * time.Second).Unix(),
	}
	// Sign the access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString([]byte("2131fwhdfh56e2s7j8"))
	if err != nil {
		log.Println("Error signing new access token:", err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	// Send the new access token as a response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"nama":   nama,
		"token":  accessTokenString,
		"userId": userId,
		"role":   role,
	})
}
