FROM    golang:latest
WORKDIR /Users/ilpauzner/go/src
#/dc-store/store
COPY    . .
RUN go build main.go
RUN chmod 755 wait-for-it.sh
CMD  ["./wait-for-it.sh", "--timeout=60", "db:6379", "--", "./wait-for-it.sh", "--timeout=60", "auth:8081", "--", "./main"]