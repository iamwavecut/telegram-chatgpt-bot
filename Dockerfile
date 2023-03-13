FROM golang:1.19-alpine as base
WORKDIR /build
COPY . .
RUN go mod download && go mod verify && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s -extldflags "-static"' -o /chatgptbot ./cmd/main.go

FROM gcr.io/distroless/static
ARG BOT_TOKEN
ARG OPENAI_TOKEN
ARG LANG=en

ENV BOT_TOKEN=${BOT_TOKEN} \
    OPENAI_TOKEN=${OPENAI_TOKEN} \
    LANG=${LANG}
COPY --from=base /chatgptbot .
ENTRYPOINT ["./chatgptbot"]