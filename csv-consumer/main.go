// +build !solution

package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v7"
	"github.com/streadway/amqp"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Stock struct {
	Name       string   `json:"name"`
	Code       uint64   `json:"code,omitempty"`
	Categories []string `json:"categories"`
}

var (
	ch       *amqp.Channel
	csvQueue amqp.Queue

	stockClient *redis.Client
)

func main() {
	stockClient = redis.NewClient(&redis.Options{Addr: "db:6379", DB: 0})

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

	msgs, err := ch.Consume(
		csvQueue.Name, // queue
		"",            // consumer
		true,          // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		log.Fatalf("%s: %s", "Failed to register a consumer", err)
	}

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			err := processMessage(d.Body)
			if err != nil {
				log.Printf("%s: %s", "Failed to process message", err)
				err = sendMessageToQueue(d.Body)
				if err != nil {
					log.Printf("%s: %s", "Failed to send message back to queue", err)
				}
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func processMessage(message []byte) error {
	log.Printf("Got from queue: %s\n", message)

	filename := filepath.Join("/tmp", string(message))
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	reader := csv.NewReader(f)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		} else {
			log.Print(record)
		}

		if len(record) != 3 {
			return errors.New("format should be code,name,cat1&cat2&cat3")
		}

		codeString := record[0]
		name := record[1]
		categoriesString := record[2]

		code, err := strconv.ParseUint(codeString, 10, 64)
		if err != nil {
			return err
		}

		stock := Stock{
			Name:       name,
			Code:       code,
			Categories: strings.Split(categoriesString, "&"),
		}

		contents, err := json.Marshal(stock)
		if err != nil {
			return err
		}

		_, err = stockClient.Get(codeString).Result()
		if err == nil {
			log.Printf("stock with %s code already exists", codeString)
			continue
		} else if !errors.Is(err, redis.Nil) {
			return err
		}

		_, err = stockClient.Set(codeString, contents, 0).Result()
		if err != nil {
			return err
		}

		log.Printf("successfully written stock %v", string(contents))
	}

	err = os.Remove(filename)
	if err != nil {
		return err
	}

	return nil
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
