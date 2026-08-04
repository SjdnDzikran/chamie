# ---- Build stage ----
FROM oven/bun:1-alpine AS build
WORKDIR /app

# Install dependencies first for layer caching
COPY bun.lock package.json ./
RUN bun install --frozen-lockfile

# Build-time args for Vite env vars (baked into JS at build time)
ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

COPY . .
RUN bun run build

# ---- Production stage ----
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
