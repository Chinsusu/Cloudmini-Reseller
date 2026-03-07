FROM node:20-alpine AS builder
WORKDIR /app

# Build context = repo root, frontend files are in ./frontend/
COPY frontend/package.json ./
RUN npm install
COPY frontend/ ./

# Pass API URL at build time so Next.js standalone rewrites are configured correctly
ARG NEXT_PUBLIC_API_URL=http://pvp-api-gateway:8080
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
RUN mkdir -p ./public

EXPOSE 3000
CMD ["node", "server.js"]
