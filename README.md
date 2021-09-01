# goBridgeJune2021

A quick demo on using RabbitMQ for your vent based application.

# Components
- RabbitMQ for our message queuing
- Jaegar to export traces to
- Otel collector to collect data
- Three simple golang programs acting as producer and consumers using Rabbitmq

  ![Demo]()

  Helpful Commands: 
  - docker-compose up
  - docker-compose down
  - go run main.go 

  Sending messages: 
  - curl -si -X POST -d "message here" http://localhost:3333/fruits
  - curl -si -X POST -d "message here" http://localhost:3333/veggies

## RabbitMQ
Messaging queue running at: 

    http://localhost:15672

## Jaegar

http://localhost:16686/

## Otel
Opentelemetry Collector to capture telemetry data and export it to your choice of backend.

### zpages
 An extention helpful in debugging and troubleshooting.

    http://localhost:55679/debug/tracez
    http://localhost:55679/debug/rpcz
    http://localhost:55679/debug/servicez
    http://localhost:55679/debug/pipelinez
    http://localhost:55679/debug/extensionz

### Heathcheck
Check Otel health 

    http://localhost:13133

## Useful Resources

- RabbitMQ: https://www.rabbitmq.com/getstarted.html
- W3C Context: https://www.w3.org/TR/trace-context/
- Jaegar Examples: https://github.com/jaegertracing/jaeger/tree/master/examples
- Microsoft-Engineering-Playbook: https://github.com/microsoft/code-with-engineering-playbook
- Gremlin Blog: https://www.gremlin.com/blog/knowing-your-systems-and-how-they-can-fail-twilio-and-aws-talk-at-chaos-conf-2020/
- Otel Sample: https://github.com/open-telemetry/opentelemetry-go/blob/master/example/otel-collector/main.go
- Intro + Demo by Ted Young (Lightstep): https://youtu.be/yQpyIrdxmQc