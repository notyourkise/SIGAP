# ==========================================
# Stage 1: Builder
# ==========================================
FROM node:20-alpine AS builder

WORKDIR /app

# Salin package.json dan package-lock.json
COPY package*.json ./

# Install dependencies (termasuk devDependencies untuk build)
RUN npm ci

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

# Expose port 3001
EXPOSE 3001

# Jalankan server Next.js pada port 3001
CMD ["npm", "run", "start", "--", "-p", "3001"]
