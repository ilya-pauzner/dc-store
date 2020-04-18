FROM    golang:latest
WORKDIR /Users/ilpauzner/go/src
#/dc-store/sender
COPY    . .
RUN go build main.go
RUN chmod 755 wait-for-it.sh
CMD  ["./wait-for-it.sh", "--timeout=30", "rabbitmq:5672", "--", "./main"]