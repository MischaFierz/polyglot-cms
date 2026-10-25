# Stage 1: compile the TypeScript admin panel
FROM node:22-alpine AS admin
WORKDIR /admin
COPY admin/package.json ./
RUN npm install
COPY admin/ ./
RUN npm run build

# Stage 2: compile the Go API (pure Go SQLite driver, so no CGO needed)
FROM golang:1.23-alpine AS api
WORKDIR /src
COPY api/ ./
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /cms-api .

# Stage 3: tiny runtime image
FROM alpine:3.20
RUN adduser -D -u 1000 cms && mkdir -p /data/uploads && chown -R cms /data
COPY --from=api /cms-api /usr/local/bin/cms-api
COPY --from=admin /admin/index.html /admin/style.css /app/admin/
COPY --from=admin /admin/dist /app/admin/dist
USER cms
ENV CMS_ADDR=:8080 \
    CMS_DB=/data/cms.db \
    CMS_UPLOAD_DIR=/data/uploads \
    CMS_ADMIN_DIR=/app/admin
EXPOSE 8080
CMD ["cms-api"]
