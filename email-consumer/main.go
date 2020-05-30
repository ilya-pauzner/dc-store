// +build !solution

package main

import (
	"github.com/streadway/amqp"
	"log"
)

var (
	ch         *amqp.Channel
	emailQueue amqp.Queue
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

	emailQueue, err = ch.QueueDeclare(
		"email", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		log.Fatalf("%s: %s", "Failed to declare a queue", err)
	}

	msgs, err := ch.Consume(
		emailQueue.Name, // queue
		"",              // consumer
		true,            // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		log.Fatalf("%s: %s", "Failed to register a consumer", err)
	}

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			err := sendMessageToConsole([]byte("Received a message: " + string(d.Body)))
			if err != nil {
				log.Printf("%s: %s", "Failed to send message to console", err)
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

func sendMessageToConsole(message []byte) error {
	log.Println(string(message))
	return nil
}

func sendMessageToQueue(message []byte) error {
	return ch.Publish(
		"",              // exchange
		emailQueue.Name, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		})
}
