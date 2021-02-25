FROM golang:1.15-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0

# Download dependencies
# COPY go.mod go.sum ./
# RUN go mod download || true

# Copy source and build
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /out/default-backend-operator . && ls /out

FROM scratch AS bin
USER 1001
ENTRYPOINT [ "/default-backend-operator" ]
COPY error_pages /error_pages/
COPY --from=0 /out/default-backend-operator /default-backend-operator
