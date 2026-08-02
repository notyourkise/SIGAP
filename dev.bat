@echo off
echo Menjalankan Backend dan Frontend SIGAP secara bersamaan...

:: Buka terminal baru dan jalankan Backend Go
start "Backend Go" cmd /c "cd backend && go run main.go"

:: Buka terminal baru dan jalankan Frontend Next.js
start "Frontend Next.js" cmd /c "cd frontend && npm run dev"

echo Selesai! Cek dua jendela terminal baru yang terbuka ya.