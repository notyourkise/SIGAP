# Manual Test - Fix BB-13 Dynamic Checklist Completion

## Informasi Umum

| Field | Value |
|-------|-------|
| **Issue** | BB-13 - Hasil pemeliharaan seluruh OK masih "Perlu Tindak Lanjut" |
| **Branch** | bugfix/bb-13-maintenance-checklist-completion |
| **Version** | 0.64.5 (development) |
| **Timestamp** | 2026-08-20 20:00 UTC+8 |
| **Tester** | AI Agent |
| **Environment** | Staging (Docker Compose) |

---

## Deskripsi Masalah

**Before Fix:**
- Backend hanya men-set status `completed` bila semua checklist item `ok` **DAN** jumlahnya persis **9 items**
- Jika ada 8 atau 10 item (atau jumlah dinamis lainnya), semuanya `ok` tapi tetap gagal → status `failed`
- Ini karena hardcode konstanta `numItems = 9` yang tidak fleksibel

**After Fix:**
- Backend menerima array checklist dinamis (jumlah berapa saja sesuai template aktif)
- Status ditentukan oleh: jika SEMUA item wajib `ok` → completed ✅; jika ada satu `not_ok` → attention ❌
- Validasi payload integrity: jika kurang/duplikat item → HTTP 400 error

---

## Prasyarat

1. Environment Docker compose berjalan
2. Database seeded dengan minimal 1 CCTV dengan status "aktif"
3. User teknisi terautentikasi dengan valid JWT token
4. Template checklist aktif tersedia di database (dari migrasi 000017_checklist_templates)

---

## Skenario Pengujian

### S1: Pengecekan dengan 8 Items Semua OK

**Step-by-Step:**

1. Login sebagai teknisi (`/api/auth/login`)
2. Pilih CCTV manapun dari daftar
3. Klik tombol "Pengecekan Lapangan"
4. Isikan form checklist dengan **8 items**, semua hasil `ok`:

```json
{
  "cctv_id": "<uuid-cctv>",
  "notes": "Testing 8 items",
  "checks": [
    {"name": "Pengecekan jaringan", "result": "ok", "note": ""},
    {"name": "Pengecekan listrik", "result": "ok", "note": ""},
    {"name": "Pengecekan fisik", "result": "ok", "note": ""},
    {"name": "Pengecekan log", "result": "ok", "note": ""},
    {"name": "Pengecekan backup", "result": "ok", "note": ""},
    {"name": "Pengecekan security", "result": "ok", "note": ""},
    {"name": "Pengecekan monitoring", "result": "ok", "note": ""},
    {"name": "Pengecekan dokumentasi", "result": "ok", "note": ""}
  ],
  "images": [{"image_url": "/api/uploads/test.jpg"}]
}
```

5. Submit form

**Expected Result:**
- ✅ HTTP 201 Created
- ✅ Response data `overall_status`: `"ok"`
- ✅ CCTV tidak berubah status (tetap "aktif")
- ✅ Notifikasi tidak dikirim (karena tidak ada issue)
- ✅ Health score tetap sama

**Actual Result:** [Tulis hasil nyata]

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

### S2: Pengecekan dengan 10 Items Semua OK

**Step-by-Step:**

1. Gunakan sesi teknisi yang sama
2. Pilih CCTV berbeda atau yang sama
3. Klik "Pengecekan Lapangan"
4. Isikan form dengan **10 items**, semua hasil `ok`:

```json
{
  "cctv_id": "<uuid-cctv>",
  "notes": "Testing 10 items",
  "checks": [
    {"name": "Pengecekan jaringan", "result": "ok", "note": ""},
    {"name": "Pengecekan listrik", "result": "ok", "note": ""},
    {"name": "Pengecekan fisik", "result": "ok", "note": ""},
    {"name": "Pengecekan log", "result": "ok", "note": ""},
    {"name": "Pengecekan backup", "result": "ok", "note": ""},
    {"name": "Pengecekan security", "result": "ok", "note": ""},
    {"name": "Pengecekan monitoring", "result": "ok", "note": ""},
    {"name": "Pengecekan dokumentasi", "result": "ok", "note": ""},
    {"name": "Pengecekan patch", "result": "ok", "note": ""},
    {"name": "Pengecekan performance", "result": "ok", "note": ""}
  ],
  "images": [{"image_url": "/api/uploads/test.jpg"}]
}
```

5. Submit form

**Expected Result:**
- ✅ HTTP 201 Created
- ✅ Response data `overall_status`: `"ok"`
- ✅ CCTV update status dari "pemeliharaan" ke "aktif" (jika sebelumnya rusak)
- ✅ Health score recalculated
- ✅ Audit log tercatat

**Actual Result:** [Tulis hasil nyata]

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

### S3: Pengecekan dengan 9 Items Ada Satu NOT_OK

**Step-by-Step:**

1. Login sebagai teknisi
2. Pilih CCTV
3. Klik "Pengecekan Lapangan"
4. Isikan **9 items** seperti default template, tapi satu item `not_ok`:

```json
{
  "cctv_id": "<uuid-cctv>",
  "notes": "Ada masalah di item 3",
  "checks": [
    {"name": "Pengecekan jaringan", "result": "ok", "note": ""},
    {"name": "Pengecekan listrik", "result": "ok", "note": ""},
    {"name": "Pengecekan fisik", "result": "not_ok", "note": "Kabel longgar"},
    {"name": "Pengecekan log", "result": "ok", "note": ""},
    {"name": "Pengecekan backup", "result": "ok", "note": ""},
    {"name": "Pengecekan security", "result": "ok", "note": ""},
    {"name": "Pengecekan monitoring", "result": "ok", "note": ""},
    {"name": "Pengecekan dokumentasi", "result": "ok", "note": ""},
    {"name": "Pengecekan patch", "result": "ok", "note": ""}
  ],
  "images": [{"image_url": "/api/uploads/test.jpg"}]
}
```

5. Submit form

**Expected Result:**
- ✅ HTTP 201 Created
- ✅ Response data `overall_status`: `"attention"`
- ✅ CCTV update status jadi "pemeliharaan" (jika sebelumnya "aktif")
- ✅ Admin receives notification in-app
- ✅ Alert badge muncul di dashboard admin

**Actual Result:** [Tulis hasil nyata]

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

### S4: Payload Kurang Item Dari Template Aktif

**Step-by-Step:**

1. Login sebagai teknisi
2. Cek template checklist aktif (harusnya punya X items)
3. Submit request dengan hanya Y items (Y < X):

```json
{
  "cctv_id": "<uuid-cctv>",
  "notes": "Missing items",
  "checks": [
    {"name": "Hanya item pertama", "result": "ok", "note": ""}
  ],
  "images": [{"image_url": "/api/uploads/test.jpg"}]
}
```

4. Submit form

**Expected Result:**
- ❌ HTTP 400 Bad Request
- ✅ Error message: "Payload checklist tidak lengkap" atau similar
- ✅ Tidak ada record dibuat di database
- ✅ No side effects (CCTV status unchanged)

**Actual Result:** [Tulis hasil nyata]

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

## Kriteria Lulus

Test suite dianggap **PASS** bila:
1. ✅ S1: 8 items semua OK → overall_status "ok" 
2. ✅ S2: 10 items semua OK → overall_status "ok"
3. ✅ S3: 9 items + 1 not_ok → overall_status "attention"
4. ✅ S4: Payload kurang item → HTTP 400 error

**Overall Suite Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

## Catatan Tambahan

### Bukti Evidence

- [ ] Screenshot respons HTTP 201 untuk S1, S2, S3
- [ ] Screenshot respons HTTP 400 untuk S4
- [ ] Database query result: verify field_maintenances table records
- [ ] Log backend (INFO level) untuk audit trail

### Build SHA & Version Info

- Git Commit SHA: `[isi setelah commit]`
- Backend Version: `0.64.5-dev`
- Frontend Version: `[isi]`
- Database Schema: `[cek migrations]`

### Cleanup Procedure

Setelah testing selesai:
1. Hapus semua record `field_maintenances` yang dibuat saat testing
2. Reset status CCTV ke kondisi awal
3. Clear notification inbox untuk admin

---

## Referensi

- **Original Bug**: BLACKBOX-FINDINGS-2026-08-19.md - BB-13
- **Backend Changelog**: See backend/changelog.md entry for BB-13 fix
- **Frontend UI**: `/field-maintenance` wizard component
- **API Contract**: Swagger docs generated after fix

---

_Ditulis: 2026-08-20 20:00 UTC+8_  
_Diperbarui: Saat eksekusi manual_
