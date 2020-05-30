// +build !solution

package main

import (
	"bufio"
	"github.com/streadway/amqp"
	"log"
	"os"
	"path/filepath"
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

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		log.Print(scanner.Text())
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
