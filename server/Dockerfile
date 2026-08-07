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
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/whatamidoing-server"]
