---
version: alpha
name: SIGAP
description: Pos Komando Nusantara untuk pemantauan dan pemeliharaan CCTV kawasan IKN.
colors:
  command-green: "#467235"
  command-green-hover: "#365b2b"
  signal-yellow: "#ffbf00"
  signal-soft: "#fff78d"
  archive-ivory: "#fff9e9"
  paper-card: "#fffef8"
  forest-ink: "#203f25"
  field-muted: "#62705a"
  sand-border: "#e4d6a7"           
  input-border: "#d4c58f"
  action-on-primary: "#ffffff"
  status-blue: "#3b82f6"
  warning-light: "#c88f00"
  error-light: "#b54432"
  night-canopy: "#162218"
  night-card: "#203122"
  night-input: "#263b28"
  night-border: "#3d553b"
  night-command-green: "#6d9a57"
  night-command-green-hover: "#7eaa68"
  night-foreground: "#fff9d7"
  night-muted: "#b6b99f"
  night-success: "#78a763"
  error-night: "#e06b56"
typography:
  display:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "30px"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "-0.025em"
  headline:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "20px"
    fontWeight: 700
    lineHeight: 1.4
    letterSpacing: "-0.01em"
  title:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "16px"
    fontWeight: 600
    lineHeight: 1.5
    letterSpacing: "-0.01em"
  body:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "-0.01em"
  body-sm:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.43
    letterSpacing: "-0.01em"
  label:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "12px"
    fontWeight: 700
    lineHeight: 1.33
    letterSpacing: "0.05em"
  technical:
    fontFamily: "Geist Mono, ui-monospace, monospace"
    fontSize: "12px"
    fontWeight: 600
    lineHeight: 1.33
    letterSpacing: "0.025em"
rounded:
  md: "8px"
  lg: "12px"
  xl: "16px"
  hero: "24px"
  full: "9999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  2xl: "24px"
  3xl: "32px"
  page-mobile: "24px"
  page-desktop: "40px"
  content-max: "1152px"
components:
  button-primary:
    backgroundColor: "{colors.command-green}"
    textColor: "{colors.action-on-primary}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.lg}"
    padding: "10px 20px"
  button-primary-hover:
    backgroundColor: "{colors.command-green-hover}"
    textColor: "{colors.action-on-primary}"
  button-secondary:
    backgroundColor: "{colors.paper-card}"
    textColor: "{colors.forest-ink}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.lg}"
    padding: "10px 16px"
  input:
    backgroundColor: "{colors.action-on-primary}"
    textColor: "{colors.forest-ink}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.lg}"
    padding: "10px 16px"
    height: "44px"
  card:
    backgroundColor: "{colors.paper-card}"
    textColor: "{colors.forest-ink}"
    rounded: "{rounded.xl}"
    padding: "20px"
  chip:
    backgroundColor: "{colors.signal-soft}"
    textColor: "{colors.forest-ink}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "4px 10px"
  navigation-active:
    backgroundColor: "{colors.command-green}"
    textColor: "{colors.action-on-primary}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.lg}"
    padding: "14px 16px"
---

# Design System: SIGAP

## Overview

**Creative North Star: "Pos Komando Nusantara"**

SIGAP terasa seperti ruang kendali operasional yang berpijak pada lingkungan Nusantara: terpercaya, sigap, teknis, dan tangguh. Antarmuka mengutamakan pembacaan kondisi, tindakan langsung, dan keterlacakan kerja. Warna hijau memegang kendali, kuning memberi sinyal, sementara permukaan gading atau hijau malam menjaga suasana tetap membumi.

Kepadatan berada di tingkat nyaman hingga padat: cukup lapang untuk membedakan kelompok informasi, tetapi tidak membuang ruang yang dibutuhkan admin dan teknisi. Sistem visual tidak mengejar kemewahan dekoratif. Otoritasnya datang dari hierarki yang tegas, status yang mudah dipindai, border yang konsisten, dan respons interaksi yang cepat.

**Key Characteristics:**

- Dua tema yang mempertahankan hubungan warna dan hierarki yang sama.
- Hijau untuk navigasi, tindakan utama, dan kondisi sehat.
- Kuning sebagai sinyal terbatas untuk perhatian dan status.
- Kontainer berlapis melalui warna permukaan dan border.
- Komponen tegas, siap kerja, dan berorientasi tugas.
- Data teknis menggunakan angka, label ringkas, serta mono seperlunya.

## Colors

Palet menggabungkan **Hijau Komando** sebagai warna otoritas operasional dan **Kuning Sinyal** sebagai aksen perhatian, di atas netral bernuansa tanah yang konsisten pada tema terang maupun gelap.

### Primary

- **Hijau Komando:** warna utama untuk tombol primer, navigasi aktif, identitas SIGAP, focus treatment, dan kondisi sehat. Versi malam memiliki lightness lebih tinggi agar tetap terbaca di atas Kanopi Malam.
- **Hijau Komando Hover:** penguatan interaksi untuk tombol dan tautan primer; tidak dipakai sebagai warna permukaan statis.

### Secondary

- **Kuning Sinyal:** menandai perhatian, proses, peran admin, dan momen yang perlu segera dipindai. Gunakan dalam bidang kecil, indikator, atau aksen; bukan sebagai latar halaman.
- **Kuning Sinyal Lembut:** lapisan ringan untuk badge, glow, dan transisi aksen pada tema terang.

### Tertiary

- **Biru Data:** membedakan informasi teknis netral seperti status terjadwal, kode, dan indikator data yang tidak bersifat berhasil, peringatan, atau gagal.

### Neutral

- **Gading Arsip:** dasar tema terang yang terasa lebih membumi daripada putih murni.
- **Kertas Pos:** permukaan kartu terang, modal, dan navigasi.
- **Tinta Rimba:** teks utama tema terang dan warna fisik label QR.
- **Catatan Lapangan:** teks sekunder, keterangan, dan metadata tema terang.
- **Garis Pasir:** pemisah dan border tema terang.
- **Kanopi Malam:** dasar tema gelap dan tema awal aplikasi.
- **Panel Malam:** permukaan kartu, sidebar, serta modal gelap.
- **Bidang Input Malam:** bidang kontrol yang perlu dibedakan dari Panel Malam.
- **Garis Malam:** border struktural tema gelap.
- **Gading Malam:** teks utama tema gelap.

**The Signal Discipline Rule.** Hijau menunjukkan kendali atau keberhasilan; kuning menunjukkan perhatian. Jangan memakai keduanya sebagai dekorasi tanpa makna.

**The Theme Continuity Rule.** Tema gelap bukan identitas kedua. Pertahankan urutan peran, hierarki, dan kepadatan yang sama; hanya nilai permukaan dan kontras yang bergeser.

## Typography

**Display Font:** Geist (dengan fallback system-ui dan sans-serif)  
**Body Font:** Geist (dengan fallback system-ui dan sans-serif)  
**Label/Mono Font:** Geist Mono (dengan fallback ui-monospace dan monospace)

**Character:** Geist memberi suara yang teknis, lugas, dan kontemporer tanpa terasa seperti konsol mesin. Geist Mono dibatasi untuk kode CCTV, alamat IP, koordinat, dan payload agar data teknis dapat dibedakan tanpa mengubah seluruh produk menjadi antarmuka terminal.

### Hierarchy

- **Display:** judul halaman utama dan sapaan dashboard; tebal, rapat, dan langsung terbaca.
- **Headline:** judul modal atau kelompok kerja penting.
- **Title:** judul kartu, seksi, dan nama entitas.
- **Body:** teks antarmuka standar dan penjelasan singkat.
- **Body Small:** isi tabel, kontrol, tombol, dan metadata operasional.
- **Label:** label status atau kategori; umumnya tebal, ringkas, uppercase, dan memiliki tracking positif.
- **Technical:** kode CCTV, alamat IP, koordinat, payload, dan nilai teknis lain yang membutuhkan pemindaian karakter per karakter.

**The Operational Hierarchy Rule.** Satu layar memiliki satu judul dominan. Kartu dan tabel menggunakan title atau label; jangan mengangkat semua kelompok informasi menjadi headline.

**The Mono Is Evidence Rule.** Mono menandai data teknis, bukan gaya. Jangan gunakan Geist Mono pada narasi, tombol umum, atau judul halaman.

## Layout

SIGAP memakai shell aplikasi dengan sidebar tetap pada desktop dan header serta drawer pada mobile. Sidebar berukuran lebar saat terbuka dan menciut menjadi jalur ikon; area konten menggunakan wadah terpusat dengan lebar maksimum 1152 px.

Ritme ruang mengikuti kenaikan 4 px dengan langkah dominan 8, 12, 16, 20, 24, dan 32 px. Padding halaman adalah 24 px pada layar kecil dan 40 px pada desktop. Kartu operasional biasanya memakai padding 20–24 px, sedangkan kontrol memakai tinggi sekitar 40–44 px.

Grid menyesuaikan kepadatan data: satu kolom di mobile, dua kolom mulai layar kecil untuk ringkasan, empat kolom pada layar lebar untuk statistik, dan pembagian asimetris untuk area utama versus panel kesehatan. Tabel boleh menggulir horizontal; jangan memampatkan kolom teknis sampai isi tidak dapat dibaca.

**The Command-Zone Rule.** Kelompokkan filter, aksi utama, dan status dalam zona yang jelas. Aksi destruktif harus terpisah secara warna dan posisi dari alur primer.

## Elevation & Depth

Kedalaman dibangun terutama melalui **lapisan warna dan border**. Gading Arsip atau Kanopi Malam menjadi dasar; Kertas Pos atau Panel Malam menjadi permukaan kerja; bidang input mendapat lapisan ketiga. Shadow bukan bahasa struktur utama.

### Shadow Vocabulary

- **Subtle Rest:** shadow sangat ringan untuk tabel, kartu statistik, atau kontrol yang perlu lepas tipis dari permukaan.
- **Command Action:** shadow berwarna Hijau Komando untuk tombol primer atau navigasi aktif yang sedang memegang fokus tindakan.
- **Modal Lift:** shadow besar untuk dialog yang berada di atas overlay gelap.
- **Login Ambient:** bayangan hijau lembut dan lebar khusus kartu autentikasi.

**The Borders Before Shadows Rule.** Gunakan perubahan warna permukaan dan border untuk struktur sehari-hari. Shadow hanya untuk modal, aksi penting, navigasi aktif, atau keadaan sementara yang benar-benar terangkat.

## Shapes

Bahasa bentuk memakai sudut membulat yang tegas, bukan gelembung dekoratif. Kontrol kecil menggunakan radius 8 px, kontrol utama 12 px, kartu dan modal 16 px, dan kartu autentikasi 24 px. Pill penuh dibatasi untuk status, filter ringkas, atau indikator.

Border tipis menjadi rangka kontainer dan kontrol. Border atas yang lebih tebal dapat membawa kode status pada kartu statistik. Bentuk lingkaran dipakai untuk titik status atau kontrol ikon yang memang setara pada semua arah.

**The Radius Ladder Rule.** Ukuran bentuk mengikuti hierarki: kontrol < kartu < permukaan hero. Jangan memakai radius 24 px pada setiap elemen.

## Components

Komponen SIGAP terasa **tegas dan siap kerja**: area klik jelas, label langsung, state dapat dibaca, dan respons berlangsung cepat tanpa gerak elastis.

### Buttons

- **Shape:** tombol umum memakai sudut 12 px; tombol kompak dan aksi tabel memakai 8 px.
- **Primary:** Hijau Komando dengan teks putih, padding horizontal 16–20 px, serta bobot semibold.
- **Hover / Focus:** hover bergeser ke Hijau Komando Hover; focus memakai ring hijau transparan; active mengecil tipis untuk umpan balik taktil.
- **Secondary:** permukaan kartu atau latar dengan border; hover menguatkan border atau menggeser warna permukaan.
- **Destructive:** merah hanya untuk hapus, gagal, atau bahaya; jangan memakai gaya destruktif untuk aksi pembatalan biasa.

### Chips

- **Style:** pill kecil dengan bidang transparan/tonal, teks semibold, dan kode warna status.
- **State:** hijau untuk aktif/selesai, kuning untuk proses/perhatian, merah untuk gagal/nonaktif, biru untuk status informasional.

### Cards / Containers

- **Corner Style:** sudut 16 px sebagai standar kartu.
- **Background:** Kertas Pos pada tema terang dan Panel Malam pada tema gelap.
- **Shadow Strategy:** datar secara default; lihat aturan Elevation & Depth.
- **Border:** satu garis struktural yang mengikuti tema.
- **Internal Padding:** 20–24 px untuk kartu utama; 12–16 px untuk kontainer di dalam kartu.

### Inputs / Fields

- **Style:** bidang input tersendiri, border jelas, sudut 12 px, dan tinggi minimal sekitar 44 px.
- **Focus:** border berubah menjadi Hijau Komando dengan ring hijau transparan.
- **Error / Disabled:** error memakai merah semantik dan pesan bahasa Indonesia; disabled menurunkan opacity serta mempertahankan keterbacaan label.

### Navigation

Sidebar adalah navigasi utama desktop; header dan drawer menggantikannya pada mobile. Item aktif memakai bidang Hijau Komando, teks putih, dan shadow ringan. Item tidak aktif tetap netral dan memperoleh latar tipis saat hover. Label boleh disembunyikan hanya ketika sidebar sengaja diciutkan; ikon dan tooltip harus mempertahankan orientasi.

### Tables

Tabel memakai header berkontras tonal, separator horizontal, hover baris yang tenang, dan kolom aksi kompak. Identitas perangkat memadukan nama, kode mono, lokasi, status, dan thumbnail QR tanpa menghilangkan urutan baca.

### Modals

Modal berada di atas overlay hitam transparan dengan blur ringan, memakai radius 16 px dan shadow besar. Judul serta tombol tutup selalu berada di bagian atas; aksi primer/destruktif dikelompokkan di bagian bawah setelah separator.

**The State Must Speak Rule.** Jangan bergantung pada warna saja. Status penting juga membutuhkan label, ikon, bentuk, atau teks yang menjelaskan kondisinya.

## Do's and Don'ts

### Do:

- **Do** gunakan Hijau Komando untuk navigasi aktif, aksi primer, focus, dan kondisi sehat.
- **Do** gunakan Kuning Sinyal secara terbatas untuk perhatian, proses, atau status yang harus cepat ditemukan.
- **Do** bangun hierarki dengan perbedaan permukaan dan border sebelum menambahkan shadow.
- **Do** pertahankan bahasa Indonesia yang singkat, operasional, dan konsisten.
- **Do** gunakan Geist Mono hanya pada kode, alamat, koordinat, dan payload teknis.
- **Do** pertahankan Logo Nusantara pada konteks identitas fisik atau produk yang relevan, termasuk label QR.
- **Do** sediakan mode mobile yang mempertahankan seluruh fungsi penting admin dan teknisi.

### Don't:

- **Don't** mengubah SIGAP menjadi dashboard SaaS generik dengan kartu bertumpuk tanpa hierarki tugas.
- **Don't** memakai neon futuristik, glow berlebihan, atau efek terminal sebagai identitas utama.
- **Don't** memberi shadow pada setiap kartu; gunakan hanya untuk modal, aksi penting, navigasi aktif, atau elevation sementara.
- **Don't** memakai kuning sebagai bidang besar atau dekorasi yang tidak membawa sinyal.
- **Don't** mencampur radius secara acak atau membuat semua komponen berbentuk pill.
- **Don't** menyembunyikan status operasional hanya dalam warna tanpa label yang dapat dibaca.
