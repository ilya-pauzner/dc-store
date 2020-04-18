FROM    golang:latest
WORKDIR /Users/ilpauzner/go/src
#/dc-store/store
COPY    . .
RUN go build main.go
RUN chmod 755 wait-for-it.sh
CMD  ["./wait-for-it.sh", "db:6379", "--", "./wait-for-it.sh", "auth:8081", "--", "./main"]