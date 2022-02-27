FROM golang:1.16-alpine as build

WORKDIR /workspace
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOARCH=amd64 go build -a -installsuffix cgo -trimpath -o consulat .

FROM alpine:3.15
COPY --from=build /workspace/consulat /usr/local/bin/consulat
CMD [ "consulat" ]