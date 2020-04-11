FROM    golang:latest
WORKDIR /Users/ilpauzner/go/src
#/dc-store/auth
COPY    . .
RUN go build main.go
CMD  ["./main"]