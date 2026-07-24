package models

// CCTV mewakili master data perangkat CCTV tanpa menyimpan kredensial
type CCTV struct {
	Base
	KodeCCTV      string `gorm:"type:varchar(50);uniqueIndex;not null" json:"kode_cctv"`
	IDChannel     string `gorm:"type:varchar(50)" json:"id_channel"`
	Nama          string `gorm:"type:varchar(100);not null" json:"nama"`
	LokasiArea    string `gorm:"type:varchar(255)" json:"lokasi_area"`
	Longitude     string `gorm:"type:varchar(50)" json:"longitude"`
	Latitude      string `gorm:"type:varchar(50)" json:"latitude"`
	JenisCCTV     string `gorm:"type:varchar(50)" json:"jenis_cctv"`
	SumberListrik string `gorm:"type:varchar(50)" json:"sumber_listrik"`
	TipeKoneksi   string `gorm:"type:varchar(50)" json:"tipe_koneksi"`
	IndoorOutdoor string `gorm:"type:varchar(20)" json:"indoor_outdoor"`
	Pemilik       string `gorm:"type:varchar(100)" json:"pemilik"`
	Status        string `gorm:"type:varchar(20);not null" json:"status"`
	IPNVR         string `gorm:"type:varchar(45)" json:"ip_nvr"`
	IPCamera      string `gorm:"type:varchar(45)" json:"ip_camera"`
	Keterangan    string `gorm:"type:text" json:"keterangan"`
	QrPayload     string `gorm:"type:text;unique" json:"qr_payload"`
}
