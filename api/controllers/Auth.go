package controllers

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"Tonton.in-Api/api/models"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"
)

type AuthController struct {
	DB *sql.DB
}

type CustomClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

type Mahasiswa struct {
	ID    int    `json:"id"`
	Nomor int    `json:"nomor"`
	NRP   string `json:"nrp"`
	Email string `json:"email"`
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

type ForgotPassword struct {
	ID           string `json:"id"`
	NRP          string `json:"nrp"`
	Email        string `json:"email"`
	NewPassword  string `json:"newpassword"`
	ConfPassword string `json:"confpassword"`
	Expired      string `json:"expired"`
	Expired_Date string `json:"expired_date"`
}

func NewAuthController(db *sql.DB) *AuthController {
	return &AuthController{DB: db}
}

func (uc *AuthController) Login(c echo.Context) error {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.Bind(&loginData); err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}

	var user models.Users
	err := uc.DB.QueryRow(`SELECT role, password FROM users WHERE username = ?`, loginData.Username).Scan(&user.Role, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusUnauthorized, "User tidak ditemukan")
		}
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Nim atau password salah")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password))
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusUnauthorized, ResponError{
			Status:  "Gagal",
			Message: "Password Salah",
		})
	}

	claims := jwt.MapClaims{
		"username": loginData.Username,
		"role":     user.Role,
		"exp":      jwt.TimeFunc().Add(time.Minute * 2880).Unix(),
		"iat":      jwt.TimeFunc().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte("access-secret")) // Replace with a stronger secret key
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Internal Server Error")
	}

	// Update user token
	user.Token = accessToken

	response := ResponLogin{
		Status:  "Berhasil",
		Message: fmt.Sprintf("Selamat Datang %s", loginData.Username),
		Token:   accessToken,
	}

	return c.JSON(http.StatusOK, response)
}

func (uc *AuthController) UsersEditPassword(c echo.Context) error {
	// Mengecek apakah token valid dan mengambil data payload
	claims, ok := c.Get("jwt").(jwt.MapClaims)
	if !ok {
		return c.JSON(http.StatusUnauthorized, "Token tidak valid")
	}

	// Mendapatkan ID pengguna dari token (asumsi userId tersimpan dalam token sebagai int)
	no_mahasiswa, no_mahasiswaOk := claims["no_mahasiswa"].(string)
	if !no_mahasiswaOk {
		return c.JSON(http.StatusForbidden, "Data payload dalam token tidak lengkap")
	}

	// Mendapatkan password yang ingin diubah dari permintaan HTTP
	var updateData struct {
		OldPassword     string `json:"old_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, "Permintaan tidak valid")
	}

	// Validasi kata sandi baru dan konfirmasi kata sandi
	if updateData.NewPassword != updateData.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, "Kata sandi baru dan konfirmasi kata sandi tidak cocok")
	}

	// Query SQL untuk mengambil password pengguna dari database berdasarkan No
	query := "SELECT password FROM mahasiswa WHERE nomor = :1"
	var storedPassword string
	err := uc.DB.QueryRow(query, no_mahasiswa).Scan(&storedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, "Pengguna tidak ditemukan")
		}
		return c.JSON(http.StatusInternalServerError, "Gagal mengambil data pengguna")
	}

	// Verifikasi password lama menggunakan bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(updateData.OldPassword)); err != nil {
		return c.JSON(http.StatusBadRequest, Respons{
			Status:  "Gagal",
			Message: "Password lama salah",
		})
	}

	// Hash password baru
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updateData.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Gagal menghash password baru")
	}

	// Query SQL untuk mengupdate password baru
	updateQuery := "UPDATE mahasiswa SET password = :1 WHERE nomor = :2"
	_, err = uc.DB.Exec(updateQuery, string(hashedPassword), no_mahasiswa)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Gagal mengupdate password")
	}

	// Delete user token
	_, err = uc.DB.Exec("DELETE FROM USER_TOKENS WHERE user_id = :1", no_mahasiswa)
	if err != nil {
		response := ResponError{
			Status:  "Berhasil",
			Message: "Telah menghapus data tokens",
		}
		return c.JSON(http.StatusOK, response)
	}

	return c.JSON(http.StatusOK, "Password berhasil diubah")
}

func (uc *AuthController) VerifyToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		tokenStr := ""
		if authHeader != "" {
			tokenArr := strings.Split(authHeader, " ")
			if len(tokenArr) == 2 && tokenArr[0] == "Bearer" {
				tokenStr = tokenArr[1]
			}
		}

		if tokenStr == "" {
			return c.JSON(http.StatusBadRequest, Respons{
				Status:  "Gagal",
				Message: "Mohon Login ke akun anda",
			})
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// PENTING: Ganti dengan secret key yang benar
			return []byte("access-secret"), nil
		})
		if err != nil || !token.Valid {
			return c.JSON(http.StatusBadRequest, Respons{
				Status:  "Gagal",
				Message: "Mohon Login ke akun anda",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.JSON(http.StatusForbidden, "Data payload dalam token tidak valid")
		}

		// Simpan data payload dalam konteks
		c.Set("jwt", claims)

		return next(c)
	}
}

func censor(word string) string {
	censored := make([]string, len(word))
	length := len(word)
	target := (length + 1) / 2

	rangeStart := 3
	rangeEnd := target

	for i := 0; i < length; i++ {
		c := string(word[i])
		if i >= rangeStart && i <= rangeEnd {
			if c == " " {
				censored[i] = "&nbsp;&nbsp;"
			} else {
				censored[i] = "*"
			}
		} else {
			censored[i] = c
		}
	}

	return strings.Join(censored, "")
}

func (uc *AuthController) Scannrp(c echo.Context) error {
	request := new(struct {
		NRP string `json:"nrp"`
	})

	if err := c.Bind(request); err != nil {
		return err
	}

	// Query ke database untuk mencari NRP
	var mahasiswa Mahasiswa
	err := uc.DB.QueryRow("SELECT email FROM MAHASISWA WHERE nrp = :1", request.NRP).Scan(&mahasiswa.Email)
	if err != nil {
		response := ResponError{
			Status:  "Gagal",
			Message: "NRP tidak ditemukan",
		}
		return c.JSON(http.StatusNotFound, response)
	}

	// Sensor email dengan menggunakan fungsi censor
	mahasiswa.Email = censor(mahasiswa.Email)

	// Mengembalikan email dengan lima huruf pertama disensor
	response := Respon{
		Status:  "Sukses",
		Message: "Berhasil Menemukan NRP",
		Data: map[string]interface{}{
			"email": mahasiswa.Email,
		},
	}
	return c.JSON(http.StatusOK, response)
}

func (uc *AuthController) ScanEmail(c echo.Context) error {
	request := new(struct {
		Nrp   string `json:"nrp"`
		Email string `json:"email"`
	})

	if err := c.Bind(request); err != nil {
		return err
	}

	// Cek apakah email ada di database Mahasiswa
	var mahasiswa Mahasiswa
	err := uc.DB.QueryRow("	SELECT EMAIL, nrp FROM mahasiswa WHERE NRP = :1 and email=:2", request.Nrp, request.Email).Scan(&mahasiswa.Email, &mahasiswa.NRP)
	if err != nil {
		response := Respon{
			Status:  "Gagal",
			Message: "Email tidak ditemukan",
			Data:    map[string]interface{}{},
		}
		return c.JSON(http.StatusNotFound, response)
	}

	// Generate random string untuk forgot password
	randomString := generateRandomString(10) // Ganti 10 dengan panjang string yang diinginkan

	// Mendapatkan waktu saat ini
	currentTime := time.Now()

	// Menambahkan 3 hari ke waktu saat ini
	expiredDate := currentTime.Add(3 * 24 * time.Hour)

	// Format tanggal kedaluwarsa menjadi string yang sesuai dengan format yang Anda gunakan dalam basis data

	// Tambahkan data ke tabel forgot_password
	_, err = uc.DB.Exec("INSERT INTO forgot_password (id, nrp, email, expired_date) VALUES (:1, :2, :3, :4)", randomString, mahasiswa.NRP, request.Email, expiredDate)

	if err != nil {
		print(err.Error())
		return err
	}

	// Kirim link forgot password ke email
	sendForgotPasswordEmail(request.Email, randomString)

	// Mengembalikan respons berhasil
	response := Respon{
		Status:  "Sukses",
		Message: "Link forgot password telah dikirim ke email Anda.",
	}
	return c.JSON(http.StatusOK, response)
}

// Fungsi untuk menghasilkan random string
func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func sendForgotPasswordEmail(email, randomString string) {
	// Inisialisasi pesan email
	m := gomail.NewMessage()
	m.SetHeader("From", "fahri.3312001092@students.polibatam.ac.id") // alamat email pengirim
	m.SetHeader("To", email)                                         // Alamat email penerima
	m.SetHeader("Subject", "Forgot Password")                        // Subjek email
	// Isi email dengan link forgot password
	m.SetBody("text/html", fmt.Sprintf("Klik link berikut untuk mereset password Anda: <a href='http://localhost:3000/auth/resetpass/%s'>Reset Password</a>", randomString))

	// Konfigurasi pengiriman email
	d := gomail.NewDialer("smtp.office365.com", 587, "fahri.3312001092@students.polibatam.ac.id", "Padang0795")

	// Kirim email
	if err := d.DialAndSend(m); err != nil {
		// Handle error jika pengiriman gagal dan log error
		log.Printf("Gagal mengirim email ke %s: %s", email, err.Error())
		// Anda juga dapat menambahkan penanganan error lain sesuai kebutuhan
	}
}

func (uc *AuthController) Ceklinkforgot(c echo.Context) error {
	request := new(struct {
		ID string `json:"id"`
	})

	if err := c.Bind(request); err != nil {
		return err
	}

	// Query ke database untuk mencari data forgot_password berdasarkan ID
	var forgotPassword ForgotPassword
	err := uc.DB.QueryRow("SELECT nrp, email, expired,expired_date FROM FORGOT_PASSWORD WHERE id = :id", request.ID).Scan(&forgotPassword.NRP, &forgotPassword.Email, &forgotPassword.Expired, &forgotPassword.Expired_Date)
	if err != nil {
		response := ResponError{
			Status:  "Gagal",
			Message: "ID tidak ditemukan",
		}
		return c.JSON(http.StatusNotFound, response)
	}

	// Validasi apakah random string sesuai
	if forgotPassword.ID != forgotPassword.ID {
		response := ResponError{
			Status:  "Gagal",
			Message: "Randomstring tidak ditemukan",
		}
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Validasi apakah expired = 1
	if forgotPassword.Expired == "1" {
		response := ResponError{
			Status:  "Gagal",
			Message: "Link reset password telah digunakan",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// Konversi expired_date  ke time.Time
	expiredDate, err := time.Parse(time.RFC3339, forgotPassword.Expired_Date)
	if err != nil {
		response := ResponError{
			Status:  "Gagal",
			Message: "Eror konversi tanggal",
		}
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Validasi apakah expired_date lebih kecil dari tanggal saat ini
	currentTime := time.Now()
	if expiredDate.Before(currentTime) {
		response := ResponError{
			Status:  "Gagal",
			Message: "Link reset password telah kadaluarsa.",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// Mengembalikan pesan sukses
	response := ResponError{
		Status:  "Berhasil",
		Message: "Link masih berlaku.",
	}
	return c.JSON(http.StatusOK, response)
}

func (uc *AuthController) ForgotPassword(c echo.Context) error {
	// Mendapatkan nilai dari parameter URL
	randomString := c.Param("randomstring")

	request := new(struct {
		NewPassword  string `json:"newpassword"`
		ConfPassword string `json:"confpassword"`
	})

	if err := c.Bind(request); err != nil {
		return err
	}

	// Query ke database untuk mencari data forgot_password berdasarkan ID
	var forgotPassword ForgotPassword
	err := uc.DB.QueryRow("SELECT nrp, email, expired,expired_date FROM FORGOT_PASSWORD WHERE id = :id", randomString).Scan(&forgotPassword.NRP, &forgotPassword.Email, &forgotPassword.Expired, &forgotPassword.Expired_Date)
	if err != nil {
		response := ResponError{
			Status:  "Gagal",
			Message: "ID tidak ditemukan",
		}
		return c.JSON(http.StatusNotFound, response)
	}

	// Validasi apakah random string sesuai
	if forgotPassword.ID != forgotPassword.ID {
		response := ResponError{
			Status:  "Gagal",
			Message: "Randomstring tidak ditemukan",
		}
		return c.JSON(http.StatusNotFound, response)
	}

	// Validasi apakah expired = 1
	if forgotPassword.Expired == "1" {
		response := ResponError{
			Status:  "Gagal",
			Message: "Link reset password telah kadaluarsa.",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// Konversi expired_date  ke time.Time
	expiredDate, err := time.Parse(time.RFC3339, forgotPassword.Expired_Date)
	if err != nil {
		response := ResponError{
			Status:  "Gagal",
			Message: "Eror konversi tanggal",
		}
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Validasi apakah expired_date lebih kecil dari tanggal saat ini
	currentTime := time.Now()
	if expiredDate.Before(currentTime) {
		response := ResponError{
			Status:  "Gagal",
			Message: "Link reset password telah kadaluarsa.",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// Validasi apakah newpassword dan confpassword cocok
	if request.NewPassword != request.ConfPassword {
		response := ResponError{
			Status:  "Gagal",
			Message: "Password baru dan konfirmasi password tidak cocok.",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// Hash the password before storing it
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response := ResponError{
			Status:  "Gagal",
			Message: "Kesalahan Hash Password.",
		}
		return c.JSON(http.StatusInternalServerError, response)
	}
	request.NewPassword = string(hashedPassword) // Simpan password yang di-hash kembali ke dalam user struct

	// Implementasi update password (sesuai dengan kebutuhan Anda)
	// Misalnya, Anda dapat menggunakan SQL untuk memperbarui password pengguna dengan NRP yang sesuai:
	_, err = uc.DB.Exec("UPDATE mahasiswa SET password = :1 WHERE nrp = :2", request.NewPassword, forgotPassword.NRP)
	if err != nil {
		return err
	}

	// Tandai link reset password sebagai sudah kadaluarsa dengan mengubah expired menjadi 1
	_, err = uc.DB.Exec("UPDATE FORGOT_PASSWORD SET expired = 1, newpassword = :1, confpassword = :2 WHERE id = :id", request.NewPassword, request.NewPassword, randomString)
	if err != nil {
		return err
	}

	// Mengembalikan pesan sukses
	response := ResponError{
		Status:  "Sukses",
		Message: "Password Berhasil Di Reset.",
	}
	return c.JSON(http.StatusOK, response)
}

func (uc *AuthController) Logout(c echo.Context) error {
	claims, ok := c.Get("jwt").(jwt.MapClaims)
	if !ok {
		return c.JSON(http.StatusUnauthorized, "Token tidak valid")
	}

	no_mahasiswa := claims["no_mahasiswa"].(string)

	// Query database untuk memeriksa apakah beasiswa dengan nomor tersebut ada
	var count int
	err := uc.DB.QueryRow("SELECT COUNT(*) FROM USER_TOKENS WHERE user_id = :1", no_mahasiswa).Scan(&count)
	if err != nil {
		response := ResponError{
			Status:  "Gagal",
			Message: "Gagal memeriksa data usertoken",
		}
		return c.JSON(http.StatusInternalServerError, response)
	}

	// Jika ID tidak ditemukan, kirim pesan error
	if count == 0 {
		response := ResponError{
			Status:  "Gagal",
			Message: "Silahkan Login Kembali",
		}
		return c.JSON(http.StatusUnauthorized, response)
	}

	_, err = uc.DB.Exec("DELETE FROM USER_TOKENS WHERE user_id = :1", no_mahasiswa)
	if err != nil {
		response := ResponError{
			Status:  "Gagal",
			Message: "Kesalahan Logout",
		}
		return c.JSON(http.StatusInternalServerError, response)
	}

	response := ResponError{
		Status:  "Berhasil",
		Message: "Anda Telah Logout",
	}
	return c.JSON(http.StatusOK, response)
}
