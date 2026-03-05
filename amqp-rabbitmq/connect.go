package main

import (
    "log"

    amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatalf("failed to connect: %v", err)
    }
    defer conn.Close()

    log.Println("Connected to RabbitMQ")
}

