package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

var (
	ch *amqp.Channel
	// valueRecorder
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

	////////////// Rabbit Setup //////////////////////////////////////////
	// conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err = conn.Channel()
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
	//////////////////////////////////////////////////////////////////////

	setupRoutes()
	log.Print("Routes Setup, starting server...")
	err = http.ListenAndServe(":3333", nil)
	if err != nil {
		panic(err)
	}

}

func setupRoutes() {
	http.HandleFunc("/fruits", fruit)
	http.HandleFunc("/greens", green)
}

func green(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = baggage.ContextWithValues(ctx,
		label.String("producer", "green"),
	)
	tracer := otel.Tracer("producer")
	var span trace.Span
	ctx, span = tracer.Start(ctx, "Producer")
	span.SetAttributes(label.String("type", "veggies"))
	defer span.End()

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	publishMessage(ctx, "greens", string(reqBody), ch)
}

func fruit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = baggage.ContextWithValues(ctx,
		label.String("producer", "fruit"),
	)
	tracer := otel.Tracer("producer")
	var span trace.Span
	ctx, span = tracer.Start(ctx, "event received")
	span.SetAttributes(label.String("type", "fruit"))
	defer span.End()

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	publishMessage(ctx, "fruit", string(reqBody), ch)
}

func publishMessage(ctx context.Context, routingKey string, body string, ch *amqp.Channel) {
	// span := trace.SpanFromContext(ctx)
	tracer := otel.Tracer("producer")
	var span trace.Span
	ctx, span = tracer.Start(ctx, "publish mesg...")
	defer span.End()
	span.AddEvent("Sending message to Rabbit", trace.WithAttributes(label.String("routingKey", routingKey), label.String("message", body)))

	err := ch.Publish(
		"food",     // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			Headers:       injectContext(ctx, make(amqp.Table)),
			CorrelationId: "abc",
			ContentType:   "text/plain",
			Body:          []byte(body),
		})
	failOnError(err, "Failed to publish a message")

	log.Printf("Sent Message: %s", body)
}

// injects tracing context into headers.
func injectContext(ctx context.Context, headers map[string]interface{}) map[string]interface{} {
	otel.GetTextMapPropagator().Inject(ctx, &headerSupplier{
		headers: headers,
	})
	log.Printf("Headers: %s", headers)
	return headers
}

type headerSupplier struct {
	headers map[string]interface{}
}

func (s *headerSupplier) Get(key string) string {
	value, ok := s.headers[key]
	if !ok {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

func (s *headerSupplier) Set(key string, value string) {
	s.headers[key] = value
}
