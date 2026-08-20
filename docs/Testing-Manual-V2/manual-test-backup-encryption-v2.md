# Manual Test — Backup Encryption (Task 7.3 Patch)

> **Kontrak uji formal — MT-13**
>
> Dokumen ini mendefinisikan prosedur pengujian enkripsi AES-256 untuk backup database & uploads SIGAP. Hasil sebelum **2026-08-21** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan environment Docker staging/production, jalankan backup dengan `ENCRYPT_BACKUPS=true`, verifikasi file `.gpg` ter-enkripsi dibuat, restore menggunakan `ENCRYPTION_KEY` yang sama, dan validasi data utuh setelah dekripsi.
>
> **Status suite:** `NOT RUN`

> **Supporting evidence Task 7.3-PATCH — 2026-08-21:** Backend & frontend build SUCCESS, skrip backup.sh/restore.sh sudah include enkripsi AES-256 support, dokumentasi BACKUP.md updated. Suite agregat MT-13 tetap `NOT RUN` karena testing E2E backup/restore dengan enkripsi belum dieksekusi. Evidence: script changes, git diff logs.

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-13 |
| Capability owner | QA Infrastructure + DevOps |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | Staging non-production (`docker-compose.dev.yml`) |
| Fixture prefix/data | `E2E-MT13-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Backup terenkripsi AES-256 berhasil, restore decrypt benar, data utuh tanpa korupsi |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |
| Status suite | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-13-S1 | YES | Setup 2 environments (encrypted vs raw), generate encryption key | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT13-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.

---

Dokumen ini memverifikasi fitur **Backup Encryption (AES-256)** untuk SIGAP, termasuk:
- Enkripsi backup database PostgreSQL (.sql.gz.gpg)
- Enkripsi backup uploads folder (.tar.gz.gpg)
- Dekripsi otomatis saat restore
- Backward compatibility (raw backups still work)
- Key management & security best practices

Env yang dipakai:
- Docker compose dev/staging environment
- PostgreSQL 15 DB instance
- Two test keys: `TEST_KEY_ENCRYPT_01` (valid), `WRONG_KEY_TEST_01` (invalid)
- Browser: Chrome dengan DevTools terbuka (untuk docker logs monitoring)
- Terminal: PowerShell/WSL with bash/GPG tools

## Prasyarat

1. **Environment Setup:**
   - Docker Compose staging mode aktif (`docker-compose.dev.yml`)
   - Database memiliki minimal fixture data (mock CCTV & maintenance records)
   - Uploads folder dengan minimal 5 dummy files
   - GPG installed (di container backup default ada `alpine + gpg`)

2. **Test Keys (generate baru):**
   ```bash
   # Generate secure test key
   openssl rand -base64 32 > test_encryption_key.txt
   
   # Read into env var
   $env:ENCRYPTION_KEY = Get-Content test_encryption_key.txt
   ```

3. **Manual Test Access:**
   - SSH/terminal access ke hosting/staging environment
   - Ability to stop/start Docker services during tests
   - Root/sudo permissions untuk backup operations

4. **Tools:**
   - Docker CLI (`docker compose exec`, `docker cp`)
   - GPG command-line tool (verify encryption manually)
   - `openssl` (key generation & verification)
   - Hex editor or text comparison tool

## Pengujian Otomatis

```bash
# 1. Verify script syntax
bash -n scripts/backup.sh && echo "✓ backup.sh syntax OK"
bash -n scripts/restore.sh && echo "✓ restore.sh syntax OK"

# 2. Check backward compatibility (should work without ENCRYPTION_KEY)
./scripts/backup.sh --help 2>&1 | Select-String "ENCRYPTION_KEY\|ENCRYPT_BACKUPS"

# 3. Validate new env vars documented
grep -i "encryption\|gpg\|aes" docs/BACKUP.md | Measure-Object | Select-Object Count
```

## Skenario Uji

### **S1: Setup Test Environment & Key Generation**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S1-1 | Generate test encryption key using OpenSSL | File `test_key.txt` created with random base64 string (32+ chars) | NOT RUN | |
| S1-2 | Set environment variables `ENCRYPTION_KEY` and `ENCRYPT_BACKUPS=true` | Both vars exported successfully, `echo $ENCRYPTION_KEY` shows value | NOT RUN | |
| S1-3 | Verify Docker compose has backup service running | `docker compose ps | Select-String "backup"` returns healthy container | NOT RUN | |
| S1-4 | Create test data in database (insert mock CCTV records) | 10+ CCTV records inserted via seeders/manual INSERT | NOT RUN | |
| S1-5 | Populate uploads folder with 5+ dummy files | Files present in `/app/uploads/` directory | NOT RUN | |
| S1-6 | Ensure backup volume exists and is writable | Volume `backups_data` mounted, `ls /backups` works | NOT RUN | |

**Acceptance:** All setup steps completed successfully before encryption testing begins.

---

### **S2: Encrypt Backup Execution**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S2-1 | Execute backup with encryption enabled | Command completes without errors, log shows "Enkripsi dilakukan" | NOT RUN | |
| S2-2 | Verify both raw AND encrypted files created | Files: `sigap_<TS>.sql.gz` AND `sigap_<TS>.sql.gz.gpg` present | NOT RUN | |
| S2-3 | Verify uploads also have .gpg version | Files: `uploads_<TS>.tar.gz` AND `uploads_<TS>.tar.gz.gpg` present | NOT RUN | |
| S2-4 | Check file sizes (GPG adds ~27 bytes header, slightly larger) | `.gpg` files should be ~27 bytes larger than raw versions | NOT RUN | |
| S2-5 | Attempt to read .gpg file as plain text | Should show garbage/encrypted content, NOT readable SQL | NOT RUN | |
| S2-6 | Verify backup timestamps match expected window | All files created within last minute of backup execution | NOT RUN | |
| S2-7 | Log output contains encryption status messages | Console/log shows: "Mengenkripsi database backup", "Uploads backup terenkripsi" | NOT RUN | |

**Acceptance:** Backup process creates BOTH raw AND encrypted versions when `ENCRYPT_BACKUPS=true`.

---

### **S3: Decrypt/Restore Process Validation**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S3-1 | Stop API service before restore (prevent write conflicts) | `docker compose stop api` successful | NOT RUN | |
| S3-2 | Run restore command pointing to .gpg files | Command starts decryption process | NOT RUN | |
| S3-3 | Provide correct ENCRYPTION_KEY during restore | Decryption succeeds, no error shown | NOT RUN | |
| S3-4 | Verify database restored correctly | Query `SELECT COUNT(*) FROM cctvs` matches pre-backup count | NOT RUN | |
| S3-5 | Verify uploads folder restored completely | All original files present after restore | NOT RUN | |
| S3-6 | API service can start successfully after restore | `docker compose start api` works, logs show normal startup | NOT RUN | |
| S3-7 | Fresh login to application works post-restore | Can login admin user, data visible in UI | NOT RUN | |

**Acceptance:** Restore from encrypted backup completely successful with data integrity preserved.

---

### **S4: Wrong Key Rejection Testing**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S4-1 | Attempt restore with WRONG encryption key | Command fails immediately with "ERROR: Dekripsi gagal" message | NOT RUN | |
| S4-2 | Verify no partial restore occurs | Database remains unchanged (original data not corrupted) | NOT RUN | |
| S4-3 | Check error code is non-zero (failure exit) | Script exits with code 1 (or error code) | NOT RUN | |
| S4-4 | Temporary decrypted file cleaned up | No temp `.sql.gz` file left in `/backups/db/` after failed attempt | NOT RUN | |
| S4-5 | Error message clearly indicates wrong key issue | Message includes: "Periksa ENCRYPTION_KEY" or similar | NOT RUN | |

**Acceptance:** Incorrect keys properly rejected, system safe from accidental corruption.

---

### **S5: Raw Backup Compatibility (Backward Compatibility)**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S5-1 | Unset `ENCRYPTION_KEY` variable | `echo $ENCRYPTION_KEY` returns empty | NOT RUN | |
| S5-2 | Run backup WITHOUT `ENCRYPT_BACKUPS=true` | Command completes successfully | NOT RUN | |
| S5-3 | Verify only raw .gz files created (NO .gpg) | Only `sigap_<TS>.sql.gz`, NO `.sql.gz.gpg` present | NOT RUN | |
| S5-4 | Restore raw backup using old method | Old restore script works on .gz files | NOT RUN | |
| S5-5 | Data integrity verified after raw restore | Counts match pre-backup values exactly | NOT RUN | |
| S5-6 | Legacy systems can still consume backups | External processes reading .sql.gz work normally | NOT RUN | |

**Acceptance:** System maintains full backward compatibility with unencrypted backups.

---

### **S6: Security Best Practices Verification**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S6-1 | Verify ENCRYPTION_KEY stored securely (not in repo) | File `test_key.txt` or similar in .gitignore | NOT RUN | |
| S6-2 | Check Docker secrets usage (optional advanced) | If configured, key loaded from `/run/secrets/encryption_key` | NOT RUN | |
| S6-3 | Verify key rotation capability | New key generated, can encrypt fresh backup successfully | NOT RUN | |
| S6-4 | Documented key storage procedures exist | Section 5.3.4 in BACKUP.md covers best practices | NOT RUN | |
| S6-5 | No key leakage in logs/error messages | Logs show `***` or `[REDACTED]` instead of actual key | NOT RUN | |
| S6-6 | Temp files removed after encryption success | No sensitive temp files left in accessible locations | NOT RUN | |

**Acceptance:** Security best practices implemented and documented.

---

### **S7: Rotasi Backup dengan Enkripsi**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S7-1 | Configure short retention (e.g., `RETENTION_DAYS=1`) for testing | Variable set, backup script will delete old files next run | NOT RUN | |
| S7-2 | Create multiple backups over time (simulate multi-day) | At least 3 separate backups created | NOT RUN | |
| S7-3 | Wait for retention policy to trigger | Files older than 1 day deleted automatically | NOT RUN | |
| S7-4 | Verify BOTH raw AND .gpg files deleted together | Old `sigap_X.sql.gz` AND `sigap_X.sql.gz.gpg` both removed | NOT RUN | |
| S7-5 | Recent backups preserved correctly | Newest backup(s) retained according to schedule | NOT RUN | |
| S7-6 | Rotasi operation doesn't corrupt remaining backups | Remaining backups still restorable successfully | NOT RUN | |

**Acceptance:** Retention policies handle both encrypted and raw files equally.

---

## Kriteria Penerimaan

Semua skenario S1–S7 harus berstatus `PASS`:

- [ ] ✅ Test environment setup complete with valid fixtures
- [ ] ✅ Encrypted backups created alongside raw versions
- [ ] ✅ GPG files contain valid AES-256 encrypted content
- [ ] ✅ Correct key allows successful decryption/restore
- [ ] ✅ Wrong key properly rejected with clear error message
- [ ] ✅ No data corruption during encryption/decryption cycle
- [ ] ✅ Raw backup format still supported (backward compatible)
- [ ] ✅ Key management documented with security best practices
- [ ] ✅ Retention policies work for both encrypted and raw files
- [ ] ✅ Production-ready deployment instructions available

---

## Catatan & Gotchas

- **GPG Performance:** Enkripsi/dekripsi bisa memakan waktu 10-30 detik tergantung ukuran backup. Jangan interrupt proses!

- **Key Persistence:** Key yang hilang = backup yang tidak bisa dipulihkan. Selalu backup key secara terpisah!

- **Alpine Limitations:** Container backup (postgres:15-alpine) includes limited GPG. Pastikan package ada (`apt-get install gnupg`).

- **File Permissions:** Encrypted files akan owned oleh root/user backup container. Adjust permissions jika perlu akses human-readable.

- **Network Storage:** Jika backup offsite (S3, Azure, GCS), pastikan transfer meng-copy `.gpg` files, bukan `.gz` originals.

- **Monitoring Integration:** Metrics/alerting untuk backup size perlu update untuk recognize .gpg extension.

- **CI/CD Pipeline:** Tambahkan step encryption test di CI pipeline sebelum production deployment.

---

## Bukti Tambahan yang Wajib Dikumpulkan

Sebelum acceptance finalisasi, kumpulkan:

1. **Screenshot/Log Evidence:**
   - Backup execution log showing "Enkripsi dilakukan" messages
   - `ls -lhR /backups` output showing .gpg files
   - GPG manual verify attempt (should fail without key)
   - Successful restore confirmation messages

2. **Database Verification:**
   ```sql
   SELECT COUNT(*) as total_cctvs FROM cctvs;
   SELECT COUNT(*) as total_maintenance_logs FROM maintenance_logs;
   -- Compare these counts before and after backup/restore
   ```

3. **Encryption Verification:**
   ```bash
   # Try to identify file type (should be ASCII armor or binary cipher)
   file backups/db/sigap_*_encrypted.sql.gz.gpg
   
   # Attempt manual decode (will fail without key, proves it's encrypted)
   echo "wrong_key" | gpg --batch --yes --passphrase-fd 0 --decrypt \
     --output /tmp/test.sql backups/db/sigap_test.sql.gz.gpg 2>&1 | Select-String "Failed"
   ```

4. **Performance Metrics:**
   - Time to create encrypted backup (vs raw)
   - Time to decrypt/restore backup
   - File size difference (percentage increase due to encryption)

5. **Security Audit Trail:**
   - Git commit hash showing backup.sh changes
   - Documentation link to section 5.3.4 best practices
   - Key rotation procedure documented

**Evidence path/link:** Store in shared folder with naming convention `MT-13-[YYYYMMDD]-[scenario]-[type].log/png/sql`

---

## Version History

| Tanggal | Perubahan | Author | Reviewer |
|---------|-----------|--------|----------|
| 2026-08-21 | Initial creation for Task 7.3 Backup Encryption patch | AI Agent | Pending |
| TBD | Update with actual test execution results | To Be Assigned | QA Lead |

**Branch terkait:** `bugfix/backup-encryption`  
**Issue tracker:** BB-19 (related security enhancement)  
**Related PR:** #XXX (pending merge)

---

## Referensi Teknis

### Enkripsi Format Details

- **Algorithm:** AES-256 (via GPG symmetric encryption)
- **Cipher Mode:** CFB (Cipher Feedback)
- **Key Derivation:** Salted PBKDF2
- **File Extension:** `.gpg` (standard GNU Privacy Guard format)

### Recovery Procedures

1. **Lost Encryption Key:**
   - ❌ Cannot recover encrypted backups
   - ✅ Use previous backup from before key change
   - 🔒 Implement key vault/KMS for production

2. **Corrupted Encrypted Backup:**
   - Retry GPG decompression with verbose mode
   - Verify checksum if available
   - Contact infrastructure team if pattern emerges

3. **Partial Restore Needed:**
   - Use GPG `--output` to extract specific tables via pg_dump filter
   - Consider point-in-time recovery from transaction logs

---

> [!NOTE]
> Dokumen ini mengikuti kontrak testing formal dari `docs/TESTING.md`. Sebelum eksekusi, pastikan semua prasyarat terpenuhi. Status `NOT RUN` berarti belum ada bukti runtime execution yang valid.
