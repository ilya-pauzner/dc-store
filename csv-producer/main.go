// +build !solution

package main

import (
	"bufio"
	"github.com/gorilla/mux"
	"github.com/ilya-pauzner/dc-store/util"
	"github.com/streadway/amqp"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

var (
	ch       *amqp.Channel
	csvQueue amqp.Queue
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("%s: %s", "Failed to connect to RabbitMQ", err)
	}
	defer func() { _ = conn.Close() }()

	ch, err = conn.Channel()
	if err != nil {
		log.Fatalf("%s: %s", "Failed to open a channel", err)
	}
	defer func() { _ = ch.Close() }()

	csvQueue, err = ch.QueueDeclare(
		"csv", // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("%s: %s", "Failed to declare a queue", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/upload", upload)

	log.Fatal(http.ListenAndServe(":8083", r))
}

func sendMessageToQueue(message []byte) error {
	return ch.Publish(
		"",            // exchange
		csvQueue.Name, // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		})
}

func upload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		util.ErrorAsJson(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer func() { _ = file.Close() }()

	err = sendMessageToQueue([]byte(header.Filename))
	if err != nil {
		util.ErrorAsJson(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("File name %s\n", header.Filename)

	newFileName := strconv.FormatUint(util.RandomUint64(), 10)
	newFile, err := os.Create(newFileName)
	if err != nil {
		util.ErrorAsJson(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() { _ = newFile.Close() }()

	// Copy the file data to my buffer
	_, err = io.Copy(newFile, file)
	if err != nil {
		util.ErrorAsJson(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go func() {
		_ = toQueue(newFileName)
	}()
}

func toQueue(newFileName string) error {
	f, err := os.Open(newFileName)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		err = sendMessageToQueue(scanner.Bytes())
		if err != nil {
			return err
		}
	}

	return nil
}
