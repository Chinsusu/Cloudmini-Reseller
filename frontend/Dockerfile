FROM node:20-alpine AS builder
WORKDIR /app

# Build context = repo root, frontend files are in ./frontend/
COPY frontend/package.json ./
RUN npm install
COPY frontend/ ./

ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
# public directory may not exist — create it if needed
RUN mkdir -p ./public
COPY --from=builder /app/public* ./public/

EXPOSE 3000
CMD ["node", "server.js"]
