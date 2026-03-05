# AMQP in Go: A Complete Tutorial

AMQP (Advanced Message Queuing Protocol) is an open standard for message-oriented middleware. In Go, the most popular library for AMQP is [`amqp091-go`](https://github.com/rabbitmq/amqp091-go) (maintained by RabbitMQ). This tutorial walks you through everything from installation to advanced patterns.

---

## Prerequisites

- Go 1.18+
- A running RabbitMQ instance (or Docker)

**Start RabbitMQ with Docker:**
```bash
docker run -d \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  rabbitmq:3-management
```

The management UI will be available at `http://localhost:15672` (guest/guest).

---

## Installation

```bash
go get github.com/rabbitmq/amqp091-go
```

---

## 1. Connecting to RabbitMQ

```go
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
```

The connection URL format is `amqp://user:password@host:port/vhost`. The default virtual host is `/`.

---

## 2. Creating a Channel

Almost all AMQP operations happen over a **channel**, which is a multiplexed, lightweight connection on top of the TCP connection.

```go
ch, err := conn.Channel()
if err != nil {
    log.Fatalf("failed to open channel: %v", err)
}
defer ch.Close()
```

> **Best practice:** Create one channel per goroutine. Channels are not safe to share across goroutines.

---

## 3. Declaring a Queue

Before publishing or consuming, you need to declare a queue. Declaration is idempotent — calling it multiple times with the same settings is safe.

```go
q, err := ch.QueueDeclare(
    "hello",  // name
    false,    // durable (survives broker restart)
    false,    // auto-delete when unused
    false,    // exclusive (only this connection can use it)
    false,    // no-wait
    nil,      // arguments
)
if err != nil {
    log.Fatalf("failed to declare queue: %v", err)
}
```

**Key parameters:**

| Parameter    | Description |
|--------------|-------------|
| `durable`    | Queue survives a RabbitMQ restart |
| `autoDelete` | Queue is deleted when the last consumer disconnects |
| `exclusive`  | Only accessible by the declaring connection |

---

## 4. Publishing a Message

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

body := "Hello, AMQP!"
err = ch.PublishWithContext(
    ctx,
    "",      // exchange (empty = default exchange)
    q.Name,  // routing key (queue name for default exchange)
    false,   // mandatory
    false,   // immediate
    amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte(body),
    },
)
if err != nil {
    log.Fatalf("failed to publish: %v", err)
}
log.Printf("Sent: %s", body)
```

---

## 5. Consuming Messages

```go
msgs, err := ch.Consume(
    q.Name,  // queue
    "",      // consumer tag (empty = auto-generated)
    true,    // auto-ack
    false,   // exclusive
    false,   // no-local
    false,   // no-wait
    nil,     // args
)
if err != nil {
    log.Fatalf("failed to register consumer: %v", err)
}

var forever chan struct{}

go func() {
    for d := range msgs {
        log.Printf("Received: %s", d.Body)
    }
}()

log.Println("Waiting for messages. Press CTRL+C to exit.")
<-forever
```

---

## 6. Manual Acknowledgement

In production, you should **not** use `auto-ack`. Instead, acknowledge messages manually after successful processing. This ensures messages are not lost if a consumer crashes.

```go
msgs, err := ch.Consume(
    q.Name,
    "",
    false, // auto-ack = false
    false,
    false,
    false,
    nil,
)

go func() {
    for d := range msgs {
        log.Printf("Processing: %s", d.Body)

        // Simulate work
        err := processMessage(d.Body)
        if err != nil {
            // Nack and requeue
            d.Nack(false, true)
            continue
        }

        // Acknowledge success
        d.Ack(false) // false = single message (not batch)
    }
}()
```

**Acknowledgement methods:**

| Method       | Description |
|--------------|-------------|
| `Ack(multiple)`  | Mark as successfully processed |
| `Nack(multiple, requeue)` | Negative ack; optionally requeue |
| `Reject(requeue)` | Reject a single message |

---

## 7. Exchanges and Routing

Exchanges route messages to queues based on **routing keys** and **binding rules**.

### Exchange Types

| Type      | Routing Behavior |
|-----------|-----------------|
| `direct`  | Routes to queues whose binding key exactly matches the routing key |
| `fanout`  | Broadcasts to all bound queues (ignores routing key) |
| `topic`   | Pattern matching with `*` (one word) and `#` (zero or more words) |
| `headers` | Routes based on message header attributes |

### Direct Exchange Example

```go
// Declare the exchange
err = ch.ExchangeDeclare(
    "logs",   // name
    "direct", // type
    true,     // durable
    false,    // auto-deleted
    false,    // internal
    false,    // no-wait
    nil,
)

// Declare a queue and bind it
q, _ := ch.QueueDeclare("", false, true, true, false, nil)
ch.QueueBind(
    q.Name,   // queue name
    "error",  // routing key
    "logs",   // exchange
    false,
    nil,
)

// Publish to the exchange with a routing key
ch.PublishWithContext(ctx, "logs", "error", false, false,
    amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte("Something went wrong"),
    },
)
```

### Topic Exchange Example

```go
ch.ExchangeDeclare("topic_logs", "topic", true, false, false, false, nil)

// Bind with patterns
ch.QueueBind(q.Name, "kern.*", "topic_logs", false, nil)   // matches kern.info, kern.error
ch.QueueBind(q.Name, "#.error", "topic_logs", false, nil)  // matches any *.error

// Publish
ch.PublishWithContext(ctx, "topic_logs", "kern.error", false, false,
    amqp.Publishing{Body: []byte("Kernel error")},
)
```

---

## 8. Durable Queues and Persistent Messages

To survive a RabbitMQ restart, you need **both** a durable queue and persistent messages.

```go
// Durable queue
q, err := ch.QueueDeclare("task_queue", true, false, false, false, nil)

// Persistent message
ch.PublishWithContext(ctx, "", q.Name, false, false,
    amqp.Publishing{
        DeliveryMode: amqp.Persistent, // <-- key setting
        ContentType:  "text/plain",
        Body:         []byte("important task"),
    },
)
```

> **Note:** `amqp.Persistent` sets `DeliveryMode` to 2. The default (`amqp.Transient`) is 1.

---

## 9. Quality of Service (Prefetch)

By default, RabbitMQ dispatches messages round-robin without regard to how many unacknowledged messages a consumer has. Use `Qos` to limit this.

```go
err = ch.Qos(
    1,     // prefetchCount: max unacknowledged messages
    0,     // prefetchSize: 0 = no limit
    false, // global: false = per-consumer, true = per-channel
)
```

This ensures that a slow consumer doesn't get overwhelmed and that work is distributed fairly among multiple consumers.

---

## 10. Dead Letter Queues (DLQ)

Messages that are rejected, expire, or overflow a queue can be routed to a **dead letter exchange (DLX)**.

```go
// 1. Declare the DLX and DLQ
ch.ExchangeDeclare("dlx", "direct", true, false, false, false, nil)
ch.QueueDeclare("dead_letters", true, false, false, false, nil)
ch.QueueBind("dead_letters", "dead", "dlx", false, nil)

// 2. Declare the main queue with DLX settings
args := amqp.Table{
    "x-dead-letter-exchange":    "dlx",
    "x-dead-letter-routing-key": "dead",
    "x-message-ttl":             int32(30000), // 30 second TTL
}
ch.QueueDeclare("main_queue", true, false, false, false, args)
```

---

## 11. RPC Pattern (Request/Reply)

AMQP supports synchronous RPC via a reply-to queue and correlation IDs.

**Client:**
```go
// Create a temporary reply queue
replyQ, _ := ch.QueueDeclare("", false, false, true, false, nil)

msgs, _ := ch.Consume(replyQ.Name, "", true, false, false, false, nil)

corrID := uuid.New().String()

ch.PublishWithContext(ctx, "", "rpc_queue", false, false,
    amqp.Publishing{
        ContentType:   "text/plain",
        CorrelationId: corrID,
        ReplyTo:       replyQ.Name,
        Body:          []byte("35"), // input
    },
)

// Wait for response
for d := range msgs {
    if d.CorrelationId == corrID {
        log.Printf("RPC result: %s", d.Body)
        break
    }
}
```

**Server:**
```go
msgs, _ := ch.Consume("rpc_queue", "", false, false, false, false, nil)

for d := range msgs {
    n, _ := strconv.Atoi(string(d.Body))
    result := fibonacci(n) // your computation

    ch.PublishWithContext(ctx, "", d.ReplyTo, false, false,
        amqp.Publishing{
            ContentType:   "text/plain",
            CorrelationId: d.CorrelationId,
            Body:          []byte(strconv.Itoa(result)),
        },
    )
    d.Ack(false)
}
```

---

## 12. Connection Recovery

Network failures are inevitable. Implement reconnection logic to build resilient consumers.

```go
func connect(url string) (*amqp.Connection, *amqp.Channel, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, nil, err
    }
    ch, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, nil, err
    }
    return conn, ch, nil
}

func startConsumerWithRecovery(url string) {
    for {
        conn, ch, err := connect(url)
        if err != nil {
            log.Printf("Connection failed: %v. Retrying in 5s...", err)
            time.Sleep(5 * time.Second)
            continue
        }

        // Register for connection close notifications
        connClose := conn.NotifyClose(make(chan *amqp.Error, 1))

        // Start consuming (your logic here)
        go consume(ch)

        // Block until connection drops
        err = <-connClose
        log.Printf("Connection closed: %v. Reconnecting...", err)
        ch.Close()
        conn.Close()
    }
}
```

---

## 13. Publisher Confirms

For guaranteed delivery, enable **publisher confirms** so the broker acknowledges each published message.

```go
err = ch.Confirm(false) // false = no-wait
if err != nil {
    log.Fatalf("failed to enable confirms: %v", err)
}

confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

ch.PublishWithContext(ctx, "", q.Name, false, false,
    amqp.Publishing{Body: []byte("important message")},
)

// Wait for broker ack
if confirm := <-confirms; confirm.Ack {
    log.Println("Message confirmed by broker")
} else {
    log.Println("Message NOT confirmed — republish or alert")
}
```

---

## 14. Complete Working Example

Below is a self-contained producer/consumer example using all recommended best practices.

```go
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
```

---

## Summary

| Concept | Key Takeaway |
|---|---|
| **Connection** | One per application; expensive to create |
| **Channel** | One per goroutine; lightweight |
| **Queue** | Declaration is idempotent |
| **Exchange** | Routes messages; `direct`, `fanout`, `topic`, `headers` |
| **Acknowledgement** | Always use manual ack in production |
| **Durability** | Durable queue + Persistent message = survives restart |
| **Prefetch (QoS)** | Prevents consumer overload; enables fair dispatch |
| **Dead Letter Queue** | Catches rejected/expired messages |
| **Publisher Confirms** | Guarantees broker received your message |
| **Reconnection** | Always implement; network failures are normal |
