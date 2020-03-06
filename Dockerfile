FROM    golang:latest
WORKDIR /Users/ilpauzner/go/src
#/dc-store
COPY    . .
RUN go build main.go
CMD  ["./main"]