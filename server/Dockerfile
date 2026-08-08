# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /whatamidoing-server ./cmd/server

# ---- runtime stage ----
FROM scratch
COPY --from=build /whatamidoing-server /whatamidoing-server
# 持久化数据目录（管理员/设备/token），用 -v ./data:/data 挂载
RUN mkdir -p /data
EXPOSE 8080
ENV PORT=8080 DATA_FILE=/data/data.json
ENTRYPOINT ["/whatamidoing-server"]
