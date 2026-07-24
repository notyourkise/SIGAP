# ==========================================
# Stage 1: Builder
# ==========================================
FROM node:20-alpine AS builder

WORKDIR /app

# Salin package.json dan package-lock.json
COPY package*.json ./

# Install dependencies menggunakan npm install sesuai request
RUN npm install

# Salin seluruh source code frontend
COPY . .

# Build Next.js
RUN npm run build

# ==========================================
# Stage 2: Runner
# ==========================================
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

# Salin file-file penting yang dibutuhkan Next.js saat runtime
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/node_modules ./node_modules

# Expose port 3000 (Next.js berjalan di port 3000 di dalam container)
EXPOSE 3000

# Jalankan server Next.js pada port 3000
CMD ["npm", "run", "start", "--", "-p", "3000"]
