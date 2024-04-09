package controllers

import (
	"database/sql"
	"log"
	"net/http"

	"Tonton.in-Api/api/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type UserController struct {
	DB *sql.DB
}

type ResponsArray struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Respons struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func NewUserController(db *sql.DB) *UserController {
	return &UserController{DB: db}
}

func (uc *UserController) GetAllVideos(c echo.Context) error {
	rows, err := uc.DB.Query(`SELECT v.id, v.title, v.slug, v.description, v.views, v.vid_like, v.dislike,id_tag,id_category,vid_thumbnail,e.name as episode
	FROM videos v
	INNER JOIN episode e on v.id_episode = e.id`)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan Pada Database")
	}
	defer rows.Close()

	var Videos []models.GetAllVideos
	for rows.Next() {
		var video models.GetAllVideos
		err := rows.Scan(&video.Id, &video.Title, &video.Slug, &video.Description, &video.Views, &video.Like, &video.Dislike, &video.Id_tag, &video.Id_category, &video.Vid_thumbnail, &video.Episode)
		if err != nil {
			log.Println(err)
			return c.JSON(http.StatusInternalServerError, "Gagal Mengambil data videos")
		}
		Videos = append(Videos, video)
	}

	return c.JSON(http.StatusOK, ResponsArray{
		Status:  "Berhasil",
		Message: "Data Video Ditemukan",
		Data:    Videos,
	})
}

func (uc *UserController) GetAllVideosBokepIndo(c echo.Context) error {
	rows, err := uc.DB.Query("SELECT id,title,slug,description,views,vid_like,`dislike`,id_tag,id_category,vid_thumbnail  FROM `videos` WHERE id_category=1")
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan Pada Database")
	}
	defer rows.Close()

	var Videos []models.GetAllVideos
	for rows.Next() {
		var video models.GetAllVideos
		err := rows.Scan(&video.Id, &video.Title, &video.Slug, &video.Description, &video.Views, &video.Like, &video.Dislike, &video.Id_tag, &video.Id_category, &video.Vid_thumbnail)
		if err != nil {
			log.Println(err)
			return c.JSON(http.StatusInternalServerError, "Gagal Mengambil data videos")
		}
		Videos = append(Videos, video)
	}

	return c.JSON(http.StatusOK, ResponsArray{
		Status:  "Berhasil",
		Message: "Data Video Ditemukan",
		Data:    Videos,
	})
}

func (uc *UserController) GetAllVideosBokepLiveRecord(c echo.Context) error {
	rows, err := uc.DB.Query("SELECT id,title,slug,description,views,vid_like,`dislike`,id_tag,id_category,vid_thumbnail  FROM `videos` WHERE id_category=2")
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan Pada Database")
	}
	defer rows.Close()

	var Videos []models.GetAllVideos
	for rows.Next() {
		var video models.GetAllVideos
		err := rows.Scan(&video.Id, &video.Title, &video.Slug, &video.Description, &video.Views, &video.Like, &video.Dislike, &video.Id_tag, &video.Id_category, &video.Vid_thumbnail)
		if err != nil {
			log.Println(err)
			return c.JSON(http.StatusInternalServerError, "Gagal Mengambil data videos")
		}
		Videos = append(Videos, video)
	}

	return c.JSON(http.StatusOK, ResponsArray{
		Status:  "Berhasil",
		Message: "Data Video Ditemukan",
		Data:    Videos,
	})
}

func (uc *UserController) GetVideosByID(c echo.Context) error {
	uuid := c.Param("id")

	rows, err := uc.DB.Query(`SELECT v.id, v.title, v.slug, v.description, v.views, v.vid_like, v.dislike, t.name as tag, c.name as category
	FROM videos v
	INNER JOIN tag t ON v.id_tag = t.id
    INNER JOIN category c ON v.id_category = c.id  
	WHERE v.uuid = ?`, uuid)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}
	defer rows.Close()

	var video models.GetAllVideos
	if rows.Next() {
		err := rows.Scan(&video.Id, &video.Title, &video.Slug, &video.Description, &video.Views, &video.Like, &video.Dislike, &video.Id_tag, &video.Id_category)
		if err != nil {
			log.Println(err)
			return c.JSON(http.StatusInternalServerError, "Gagal mengambil data video")
		}
	} else {
		return c.JSON(http.StatusNotFound, Respons{
			Status:  "Gagal",
			Message: "Video Tidak ditemukan",
		})
	}

	return c.JSON(http.StatusOK, Respon{
		Status:  "Berhasil",
		Message: "Data video ditemukan",
		Data:    video,
	})
}

func (uc *UserController) CreateVideos(c echo.Context) error {
	var err error
	var videos struct {
		Title       string `json:"title"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		Views       string `json:"views"`
		Like        string `json:"vid_like"`
		Dislike     string `json:"dislike"`
		Id_tag      string `json:"id_tag"`
		Id_category string `json:"id_category"`
		Episode     string `json:"episode"`
	}

	if err := c.Bind(&videos); err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, Respons{
			Status:  "Gagal",
			Message: "Permintaan tidak valid",
		})
	}

	if videos.Title == "" || videos.Description == "" {
		return c.JSON(http.StatusBadRequest, Respons{
			Status:  "Gagal",
			Message: "Judul and Deskripsi tidak boleh kosong!",
		})
	}

	// Generate UUID baru
	uuidNew, err := uuid.NewRandom()
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Gagal menghasilkan UUID")
	}

	_, err = uc.DB.Exec(
		"INSERT INTO videos (title, slug, description, views, vid_like, dislike, id_tag, id_category,uuid,episode) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		videos.Title, videos.Slug, videos.Description, videos.Views, videos.Like, videos.Dislike, videos.Id_tag, videos.Id_category, uuidNew.String(), videos.Episode,
	)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, Respons{
			Status:  "Gagal",
			Message: "Kesalahan menginput data kedatabase",
		})
	}

	response := Respons{
		Status:  "Berhasil",
		Message: "Video Berhasil Ditambahkan",
	}

	return c.JSON(http.StatusCreated, response)
}

func (uc *UserController) EditVideos(c echo.Context) error {
	// Mendapatkan ID video dari URL
	uuid := c.Param("id")

	// Mendapatkan data video yang akan diubah dari body request
	var reqBody models.GetAllVideos
	if err := c.Bind(&reqBody); err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, "Permintaan tidak valid")
	}

	// Query untuk mengupdate data video berdasarkan ID
	queryUpdate := `
        UPDATE videos
        SET title = ?, slug = ?, description = ?, views = ?, vid_like = ?, dislike = ?, id_tag = ?, id_category = ?
        WHERE uuid = ?
    `
	_, err := uc.DB.Exec(queryUpdate, reqBody.Title, reqBody.Slug, reqBody.Description, reqBody.Views, reqBody.Like, reqBody.Dislike, reqBody.Id_tag, reqBody.Id_category, uuid)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	// Mengambil data video yang sudah diupdate
	var video models.GetAllVideos
	querySelect := "SELECT id, title, description, views, vid_like, dislike, id_tag, id_category FROM videos WHERE uuid = ?"
	rows, err := uc.DB.Query(querySelect, uuid)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}
	defer rows.Close()

	// Memindai baris hasil
	if rows.Next() {
		err := rows.Scan(&video.Id, &video.Title, &video.Description, &video.Views, &video.Like, &video.Dislike, &video.Id_tag, &video.Id_category)
		if err != nil {
			log.Println(err)
			return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
		}
	} else {
		return c.JSON(http.StatusNotFound, Respons{
			Status:  "Gagal",
			Message: "Video tidak ditemukan!",
		})
	}

	// Mengembalikan respons sukses
	return c.JSON(http.StatusOK, Respons{
		Status:  "Berhasil",
		Message: "Data video berhasil diubah",
	})
}

func (uc *UserController) DeleteVideos(c echo.Context) error {
	uuid := c.Param("id")

	var count int
	err := uc.DB.QueryRow("SELECT COUNT(*) FROM videos WHERE uuid = ?", uuid).Scan(&count)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Gagal memeriksa data videos")
	}

	// Jika ID tidak ditemukan, kirim pesan error
	if count == 0 {
		return c.JSON(http.StatusNotFound, Respons{
			Status:  "Gagal",
			Message: "videos tidak ditemukan",
		})
	}

	// Menghapus pengumuman dari database berdasarkan ID
	_, err = uc.DB.Exec("DELETE FROM videos WHERE uuid = ?", uuid)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Gagal memeriksa data videos")
	}

	response := Respons{
		Status:  "Berhasil",
		Message: "Video Berhasil Dihapus",
	}

	return c.JSON(http.StatusOK, response)
}

func (uc *UserController) Inclikevideos(c echo.Context) error {
	uuid := c.Param("id")

	// Prepared statement untuk UPDATE
	stmt, err := uc.DB.Prepare(`UPDATE videos SET vid_like = vid_like + 1 WHERE uuid = ?`)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}
	defer stmt.Close()

	// Eksekusi prepared statement
	result, err := stmt.Exec(uuid)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	// Periksa jumlah baris yang terpengaruh
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, Respons{
			Status:  "Gagal",
			Message: "Video Tidak ditemukan",
		})
	}

	// Jika berhasil, kembalikan respons OK
	return c.JSON(http.StatusOK, Respon{
		Status:  "Berhasil",
		Message: "Jumlah like video bertambah",
	})
}

func (uc *UserController) Declikevideos(c echo.Context) error {
	uuid := c.Param("id")

	// Prepared statement untuk UPDATE
	stmt, err := uc.DB.Prepare(`UPDATE videos SET vid_like = vid_like - 1 WHERE uuid = ?`)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}
	defer stmt.Close()

	// Eksekusi prepared statement
	result, err := stmt.Exec(uuid)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	// Periksa jumlah baris yang terpengaruh
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, Respons{
			Status:  "Gagal",
			Message: "Video Tidak ditemukan",
		})
	}

	// Jika berhasil, kembalikan respons OK
	return c.JSON(http.StatusOK, Respon{
		Status:  "Berhasil",
		Message: "Jumlah like video berkurang",
	})
}

func (uc *UserController) IncDislikevideos(c echo.Context) error {
	uuid := c.Param("id")

	// Prepared statement untuk UPDATE
	stmt, err := uc.DB.Prepare(`UPDATE videos SET dislike = dislike + 1 WHERE uuid = ?`)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}
	defer stmt.Close()

	// Eksekusi prepared statement
	result, err := stmt.Exec(uuid)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	// Periksa jumlah baris yang terpengaruh
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, Respons{
			Status:  "Gagal",
			Message: "Video Tidak ditemukan",
		})
	}

	// Jika berhasil, kembalikan respons OK
	return c.JSON(http.StatusOK, Respon{
		Status:  "Berhasil",
		Message: "Jumlah dislike video bertambah",
	})
}

func (uc *UserController) DecDislikevideos(c echo.Context) error {
	uuid := c.Param("id")

	// Prepared statement untuk UPDATE
	stmt, err := uc.DB.Prepare(`UPDATE videos SET dislike = dislike - 1 WHERE uuid = ?`)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}
	defer stmt.Close()

	// Eksekusi prepared statement
	result, err := stmt.Exec(uuid)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	// Periksa jumlah baris yang terpengaruh
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, "Kesalahan pada database")
	}

	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, Respons{
			Status:  "Gagal",
			Message: "Video Tidak ditemukan",
		})
	}

	// Jika berhasil, kembalikan respons OK
	return c.JSON(http.StatusOK, Respon{
		Status:  "Berhasil",
		Message: "Jumlah dislike video berkurang",
	})
}
