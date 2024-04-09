package models

type GetAllVideos struct {
	Id            string
	Title         string `json:"title"`
	Slug          string `json:"slug"`
	Description   string `json:"description"`
	Views         string `json:"views"`
	Like          string `json:"vid_like"`
	Dislike       string `json:"dislike"`
	Id_tag        string `json:"id_tag"`
	Id_category   string `json:"id_category"`
	Vid_thumbnail string `json:"vid_thumbnail"`
	Episode       string `json:"id_episode"`
}

type ResponError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
