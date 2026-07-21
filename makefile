# Makefile untuk Monorepo SIGAP

.PHONY: frontend backend dev

# Menjalankan backend Go Fiber
backend:
	cd backend && go run main.go

# Menjalankan frontend Next.js
frontend:
	cd frontend && npm run dev

# Menjalankan keduanya secara bersamaan (Butuh tools seperti 'make')
dev:
	@echo "Menjalankan Backend dan Frontend secara bersamaan..."
	make -j 2 backend frontend