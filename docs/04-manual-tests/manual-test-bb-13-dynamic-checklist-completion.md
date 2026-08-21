# Manual Test - Bug BB-13: Dynamic Checklist Status

| Item | Detail |
|------|--------|
| **Task ID** | BB-13 |
| **Branch** | `bugfix/bb-13-dynamic-checklist-status` |
| **Versi Backend** | 0.64.5 (setelah fix) |
| **Tanggal** | 2026-08-21 |

---

## 1. Prasyarat

- [ ] Docker-compose running (`db`, `redis`, `api`, `frontend`)
- [ ] Login sebagai **teknisi** yang memiliki tugas terjadwal dengan status `ongoing`
- [ ] Pastikan template checklist yang aktif BISA bervariasi (tidak harus 9 items)
  - Bisa lewat admin dashboard: `/schedules` → edit jadwal → ubah template
  - Atau langsung di database table `checklist_templates`

---

## 2. Pengujian Otomatis (Build & Lint)

### Backend Build
```bash
cd backend
go build ./...          # Should pass without errors
go test ./... -count=1   # All tests should pass
```

### Frontend Build
```bash
cd frontend
npm run lint            # No critical lint errors
npm run build           # Production build should succeed
```

### Swagger Regeneration
```bash
cd backend
swag init              # Should regenerate docs.go without errors
```

---

## 3. Skenario Uji Manual

### **Test Case 1: Completion with Fewer than 9 Items (e.g., 5 items)**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Admin membuat/mengedit jadwal pemeliharaan dengan template berisi **5 item checklist** | Jadwal tersimpan dengan template baru | ✅ / ❌ |
| 2 | Generator menghasilkan maintenance log dari jadwal tersebut | Log status: `scheduled` → `ongoing` | ✅ / ❌ |
| 3 | Teknisi membuka halaman `/maintenance` → klik "Selesaikan" pada task dengan status `ongoing` | Modal complete checklist terbuka, menampilkan **5 item** | ✅ / ❌ |
| 4 | Teknisi mengisi semua 5 item dengan status **"Normal / OK"** + minimal 1 foto bukti | Validasi berhasil, tombol Submit aktif | ✅ / ❌ |
| 5 | Teknisi klik **"Selesaikan Tugas"** | API response: `{"status": "success", "data": {...}}` | ✅ / ❌ |
| 6 | Verifikasi status maintenance log di tabel / halaman | Status berubah menjadi **`completed`** (BUKAN `failed`) | ✅ / ❌ |
| 7 | Cek notifikasi ke admin | Notifikasi muncul: "Tugas selesai: Selesai" | ✅ / ❌ |
| 8 | Cek status CCTV terkait | CCTV status: `aktif` (karena completed) | ✅ / ❌ |

**Kriteria Lulus:** Semua langkah berhasil, status = `completed` meskipun hanya 5 items.

---

### **Test Case 2: Completion with Exactly 9 Items (Legacy Behavior)**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Setup template dengan **9 item checklist** (template default) | Template tersimpan | ✅ / ❌ |
| 2 | Generator menghasilkan maintenance log | Log status: `ongoing` | ✅ / ❌ |
| 3 | Teknisi menyelesaikan semua 9 item dengan status **"OK"** + foto | Form valid, submit berhasil | ✅ / ❌ |
| 4 | Verifikasi hasil | Status = **`completed`**, CCTV → `aktif` | ✅ / ❌ |

**Kriteria Lulus:** Legacy behavior tetap berfungsi (backward compatibility).

---

### **Test Case 3: Failure with Mixed Results (Any Count)**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Setup template dengan **7 item checklist** | Template siap | ✅ / ❌ |
| 2 | Teknisi menyelesaikan task dengan kondisi: <br>- 5 items = "OK" <br>- 2 items = "Bermasalah" | Form valid (semua terisi, ada catatan untuk bermasalah) | ✅ / ❌ |
| 3 | Submit form | ✅ / ❌ |
| 4 | Verifikasi status maintenance log | Status = **`failed`** | ✅ / ❌ |
| 5 | Cek notifikasi admin | Notifikasi: "Tugas selesai: Perlu Tindak Lanjut" | ✅ / ❌ |
| 6 | Cek status CCTV | Status = `pemeliharaan` | ✅ / ❌ |

**Kriteria Lulus:** Ada item "not_ok" → status = `failed`, berapa pun jumlah total items.

---

### **Test Case 4: Edge Case - Single Item Template**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Buat template dengan **1 item checklist** | Template valid | ✅ / ❌ |
| 2 | Teknisi menyelesaikan dengan status "OK" | Submit berhasil | ✅ / ❌ |
| 3 | Verifikasi | Status = **`completed`** | ✅ / ❌ |

**Kriteria Lulus:** Sistem mendukung template minimal 1 item.

---

## 4. Kriteria Lulus Umum

✅ **PASS** jika:
- Status maintenance log benar (`completed` vs `failed`) berdasarkan kelengkapan dan hasil checklist
- Status CCTV sinkron otomatis dengan hasil penyelesaian
- Tidak ada error di backend logs
- Frontend menampilkan status yang sesuai
- Notifikasi dikirim dengan label yang tepat ("Selesai" atau "Perlu Tindak Lanjut")

❌ **FAIL** jika:
- Status masih terpaku pada hardcoded "9/9 OK"
- Template dengan jumlah berbeda tidak bekerja sesuai ekspektasi
- Error 500 atau validation error saat submit

---

## 5. Catatan Tambahan

### Backend Logs
Periksa `backend/logs/app.log` setelah test:
- Harus ada log `COMPLETE_MAINTENANCE: Menyelesaikan tugas pemeliharaan... (Status: completed)`
- Tidak ada error `gorm`, `uuid.Parse`, atau transaction rollback

### Database Verification
```sql
-- Cek status terakhir maintenance log
SELECT id, action_taken, status, completed_at 
FROM maintenance_logs 
ORDER BY created_at DESC 
LIMIT 5;

-- Cek sinkronisasi status CCTV
SELECT c.kode_cctv, c.status, ml.status as maintenance_status
FROM cctvs c
JOIN maintenance_logs ml ON c.id = ml.cctv_id
WHERE ml.status IN ('completed', 'failed')
ORDER BY ml.created_at DESC
LIMIT 5;
```

### Webhook Event (Jika Enabled)
Verifikasi event `maintenance.completed` dikirim ke configured webhooks dengan payload:
```json
{
  "event": "maintenance.completed",
  "payload": {
    "log_id": "...",
    "cctv_id": "...",
    "action_taken": "...",
    "status": "completed",
    "completed_at": "..."
  }
}
```

---

## 6. Rollback Plan

Jika ditemukan regressi atau issue kritis:

1. Revert commit:  
   ```bash
   git checkout main
   git merge --no-edit bugfix/bb-13-dynamic-checklist-status^
   ```

2. Restore backup branch:
   ```bash
   git checkout -b rollback/bb-13-backup
   ```

3. Deploy versi sebelumnya:
   ```bash
   git cherry-pick <previous-stable-commit>
   ```

---

**Tester**: ________________  
**Tanggal Testing**: ________________  
**Status**: ⬜ Pass / ⬜ Fail / ⬜ Blocked  
**Catatan**: _______________________________________________________


