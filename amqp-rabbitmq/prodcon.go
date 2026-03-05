package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    amqp "github.com/rabbitmq/amqp091-go"
)

const amqpURL = "amqp://guest:guest@localhost:5672/"
const queueName = "example"

func main() {
    conn, err := amqp.Dial(amqpURL)
    if err != nil {
        log.Fatalf("dial: %v", err)
    }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil {
        log.Fatalf("channel: %v", err)
    }
    defer ch.Close()

    q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
    if err != nil {
        log.Fatalf("queue declare: %v", err)
    }

    if err := ch.Qos(5, 0, false); err != nil {
        log.Fatalf("qos: %v", err)
    }

    // Producer goroutine
    go func() {
        for i := 0; ; i++ {
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            err := ch.PublishWithContext(ctx, "", q.Name, false, false,
                amqp.Publishing{
                    DeliveryMode: amqp.Persistent,
                    ContentType:  "text/plain",
                    Body:         []byte(time.Now().String()),
                },
            )
            cancel()
            if err != nil {
                log.Printf("publish error: %v", err)
            } else {
                log.Printf("published message #%d", i)
            }
            time.Sleep(time.Second)
        }
    }()

    // Consumer
    msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
    if err != nil {
        log.Fatalf("consume: %v", err)
    }

    go func() {
        for d := range msgs {
            log.Printf("received: %s", d.Body)
            time.Sleep(500 * time.Millisecond) // simulate work
            d.Ack(false)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Shutting down...")
}
