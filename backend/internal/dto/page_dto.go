package dto

import "time"

// PageItemResponse adalah satu butir bagian halaman.
type PageItemResponse struct {
	Label     string `json:"label"`
	Value     string `json:"value"`
	Note      string `json:"note"`
	Icon      string `json:"icon"`
	Href      string `json:"href"`
	SortOrder int    `json:"sort_order"`
}

// PageSectionResponse adalah satu bagian halaman beserta butirnya.
type PageSectionResponse struct {
	Key         string             `json:"key"`
	Kind        string             `json:"kind"`
	Eyebrow     string             `json:"eyebrow"`
	Title       string             `json:"title"`
	Lead        string             `json:"lead"`
	Body        string             `json:"body"`
	Layout      string             `json:"layout"`
	SortOrder   int                `json:"sort_order"`
	IsPublished bool               `json:"is_published"`
	Items       []PageItemResponse `json:"items"`
}

// PageResponse adalah satu halaman publik lengkap dengan bagiannya.
type PageResponse struct {
	Slug            string                `json:"slug"`
	Eyebrow         string                `json:"eyebrow"`
	Title           string                `json:"title"`
	Lead            string                `json:"lead"`
	MetaTitle       string                `json:"meta_title"`
	MetaDescription string                `json:"meta_description"`
	IsPublished     bool                  `json:"is_published"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Sections        []PageSectionResponse `json:"sections"`
}

// PageUpdateRequest mengganti isi satu halaman secara utuh.
//
// Penggantian utuh, bukan tambal per bagian: panel admin selalu mengirim
// keadaan akhir halaman, sehingga isi yang dihapus admin benar-benar hilang
// dan urutan bagian tidak pernah setengah tersimpan.
type PageUpdateRequest struct {
	Eyebrow         string               `json:"eyebrow" binding:"omitempty,max=80"`
	Title           string               `json:"title" binding:"omitempty,max=200"`
	Lead            string               `json:"lead" binding:"omitempty,max=4000"`
	MetaTitle       string               `json:"meta_title" binding:"omitempty,max=200"`
	MetaDescription string               `json:"meta_description" binding:"omitempty,max=320"`
	IsPublished     *bool                `json:"is_published"`
	Sections        []PageSectionRequest `json:"sections" binding:"omitempty,max=40,dive"`
}

// PageSectionRequest adalah satu bagian yang dikirim panel admin.
type PageSectionRequest struct {
	Key         string            `json:"key" binding:"required,max=60"`
	Kind        string            `json:"kind" binding:"required,max=20"`
	Eyebrow     string            `json:"eyebrow" binding:"omitempty,max=80"`
	Title       string            `json:"title" binding:"omitempty,max=200"`
	Lead        string            `json:"lead" binding:"omitempty,max=4000"`
	Body        string            `json:"body" binding:"omitempty,max=40000"`
	Layout      string            `json:"layout" binding:"omitempty,max=20"`
	SortOrder   int               `json:"sort_order"`
	IsPublished *bool             `json:"is_published"`
	Items       []PageItemRequest `json:"items" binding:"omitempty,max=60,dive"`
}

// PageItemRequest adalah satu butir bagian yang dikirim panel admin.
type PageItemRequest struct {
	Label     string `json:"label" binding:"required,max=255"`
	Value     string `json:"value" binding:"omitempty,max=8000"`
	Note      string `json:"note" binding:"omitempty,max=4000"`
	Icon      string `json:"icon" binding:"omitempty,max=40"`
	Href      string `json:"href" binding:"omitempty,max=255"`
	SortOrder int    `json:"sort_order"`
}
