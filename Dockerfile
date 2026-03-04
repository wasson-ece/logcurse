FROM golang:1.24 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o /logcurse

FROM scratch
COPY --from=builder /logcurse /logcurse
EXPOSE 8080
ENTRYPOINT ["/logcurse", "--serve"]
