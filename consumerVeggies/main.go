package main

import (
	"context"
	"log"

	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	trace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	ctx := context.Background()
	driver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint("localhost:55680"),
		otlpgrpc.WithDialOption(grpc.WithBlock()), // useful for testing
	)
	exporter, err := otlp.NewExporter(ctx, driver)

	if err != nil {
		log.Fatalf("failed to initialize stdout export pipeline: %v", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(bsp))

	// Handle this error in a sensible manner where possible
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)
	propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)

	//////////////////// Rabbit Setup /////////////////////////
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"food",   // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare(
		"grassy_greens", // name
		false,           // durable
		false,           // delete when unused
		true,            // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	failOnError(err, "Failed to declare a queue")

	log.Printf("Binding queue %s to exchange %s with routing key green", q.Name, "colours")
	err = ch.QueueBind(
		q.Name,   // queue name
		"greens", // routing key
		"food",   // exchange
		false,
		nil)
	failOnError(err, "Failed to bind a queue")

	/////////////////////////////////////////////////////////////////

	msgs, err := ch.Consume(
		q.Name,         // queue
		"grass_greens", // consumer
		true,           // auto ack
		false,          // exclusive
		false,          // no local
		false,          // no wait
		nil,            // args
	)
	failOnError(err, "Failed to register a consumer")

	tracer := otel.Tracer("consumer_veggies")
	var span trace.Span

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			ctx, span = tracer.Start(ctx, "Vegetables", trace.WithAttributes(
				label.String("Green", "Grass")))
			span.AddEvent("green veggie", trace.WithAttributes(label.String("message", string(d.Body))))

			log.Printf("Message Received: %s", d.Body)
			span.End()
		}
	}()

	log.Printf("Waiting for messages. To exit press CTRL+C")
	<-forever
}
