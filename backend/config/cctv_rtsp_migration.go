package config

import (
	"strings"

	"sigap-backend/models"
	"sigap-backend/utils"

	"gorm.io/gorm"
)

// EncryptLegacyRTSPURLs mengenkripsi rtsp_url lama yang masih plaintext.
// Migration ini idempoten: nilai yang sudah berprefix enc:v1: dilewati.
// Tidak ada data yang dihapus; hanya representasi at-rest yang diubah.
func EncryptLegacyRTSPURLs(db *gorm.DB) {
	if db == nil {
		return
	}

	var cctvs []models.CCTV
	if err := db.Select("id", "rtsp_url").
		Where("rtsp_url IS NOT NULL AND rtsp_url <> ''").
		Find(&cctvs).Error; err != nil {
		utils.Log.Warnf("⚠️ Gagal memuat URL RTSP legacy untuk enkripsi: %v", err)
		return
	}

	updated := 0
	for _, cctv := range cctvs {
		if utils.IsEncryptedRTSPURL(cctv.RTSPURL) {
			continue
		}
		if strings.TrimSpace(cctv.RTSPURL) == "" {
			continue
		}

		encrypted, err := utils.EncryptRTSPURL(cctv.RTSPURL)
		if err != nil {
			utils.Log.Warnf("⚠️ Gagal mengenkripsi URL RTSP CCTV %s: %v", cctv.ID, err)
			continue
		}
		if err := db.Model(&models.CCTV{}).
			Where("id = ?", cctv.ID).
			Update("rtsp_url", encrypted).Error; err != nil {
			utils.Log.Warnf("⚠️ Gagal menyimpan URL RTSP terenkripsi CCTV %s: %v", cctv.ID, err)
			continue
		}
		updated++
	}

	if updated > 0 {
		utils.Log.Infof("🔐 %d URL RTSP legacy berhasil dienkripsi at-rest", updated)
	}
}
