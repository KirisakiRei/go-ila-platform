package dto

// PushPaginationQuery membatasi daftar langganan perangkat per halaman.
type PushPaginationQuery struct {
	Page    int `form:"page,default=1" binding:"omitempty,min=1"`
	PerPage int `form:"per_page,default=10" binding:"omitempty,min=1,max=100"`
}

// PushPaginationMeta adalah metadata paginasi standar daftar perangkat push.
type PushPaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// PushKeysPayload membawa kunci enkripsi ECDH P-256 dan auth secret dari browser.
type PushKeysPayload struct {
	P256dh string `json:"p256dh" binding:"required,min=16,max=256"`
	Auth   string `json:"auth" binding:"required,min=8,max=128"`
}

// PushSubscribeRequest mendaftarkan atau memperbarui langganan Web Push satu perangkat.
type PushSubscribeRequest struct {
	Endpoint    string          `json:"endpoint" binding:"required,max=2048"`
	Keys        PushKeysPayload `json:"keys" binding:"required"`
	DeviceLabel string          `json:"device_label" binding:"omitempty,max=120"`
	NotifyEntry *bool           `json:"notify_entry,omitempty"`
	NotifyExit  *bool           `json:"notify_exit,omitempty"`
}

// PushUpdatePreferencesRequest memperbarui saklar notifikasi per perangkat.
type PushUpdatePreferencesRequest struct {
	NotifyEntry *bool `json:"notify_entry"`
	NotifyExit  *bool `json:"notify_exit"`
}

// PushUnsubscribeRequest mencabut langganan berdasarkan endpoint browser aktif.
type PushUnsubscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required,max=2048"`
}

// PushSubscriptionItem adalah representasi aman satu perangkat tanpa kunci rahasia.
type PushSubscriptionItem struct {
	ID           string  `json:"id"`
	DeviceLabel  string  `json:"device_label"`
	NotifyEntry  bool    `json:"notify_entry"`
	NotifyExit   bool    `json:"notify_exit"`
	LastSentAt   *string `json:"last_sent_at"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	EndpointHost string  `json:"endpoint_host"`
}

// PushStatusResponse mengembalikan kunci publik VAPID beserta perangkat terdaftar.
type PushStatusResponse struct {
	Supported      bool                   `json:"supported"`
	VAPIDPublicKey string                 `json:"vapid_public_key"`
	Subscriptions  []PushSubscriptionItem `json:"subscriptions"`
}

// PushTestResponse melaporkan hasil pengiriman tes notifikasi.
type PushTestResponse struct {
	Delivered int `json:"delivered"`
	Expired   int `json:"expired"`
	Failed    int `json:"failed"`
}
