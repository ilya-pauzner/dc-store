FROM    golang:latest
WORKDIR /Users/ilpauzner/go/src
#/dc-store/csv-producer
COPY    . .
RUN go build main.go
RUN chmod 755 wait-for-it.sh
CMD  ["./wait-for-it.sh", "--timeout=60", "rabbitmq:5672", "--", "./main"]
