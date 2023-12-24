package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	args := os.Args

	kafkaEndpoint := os.Getenv("KAFKA_ENDPOINT")
	if kafkaEndpoint == "" {
		log.Fatal("please specify KAFKA_ENDPOINT environment variable")
	}
	log.Printf("KAFKA_ENDPOINT - %s\n", kafkaEndpoint)

	consumerGroup := os.Getenv("KAFKA_CONSUMER_GROUP")
	if consumerGroup == "" {
		log.Fatal("please specify KAFKA_CONSUMER_GROUP environment variable")
	}
	log.Printf("KAFKA_CONSUMER_GROUP - %s\n", consumerGroup)

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		log.Fatal("please specify KAFKA_TOPIC environment variable")
	}
	log.Printf("KAFKA_TOPIC - %s\n", topic)

	if args[1] != "consumer" {
		consumer(kafkaEndpoint, consumerGroup, topic)
	} else if args[1] != "producer" {
		producer(kafkaEndpoint, consumerGroup, topic)
	} else {
		fmt.Println("I am not sure I understand that. I am limited to just 'consumer' & 'producer' commands.")
		os.Exit(1)
	}

}

func consumer(kafkaEndpoint, consumerGroup, topic string) {

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaEndpoint,
		"group.id":          consumerGroup,
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		log.Fatalf("unable to create consumer %v", err)
	}
	defer consumer.Close()

	err = consumer.Subscribe(topic, nil)
	if err != nil {
		log.Fatalf("failed to subscribe to topic %v", err)
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)
	var closed bool
	consumerChannel := make(chan int)

	go func() {
		log.Println("waiting for messages...")
		for !closed {
			select {
			case <-exit:
				log.Println("shutdown request..")

				closed = true
				consumerChannel <- 1
			default:
				log.Println("reading message ..")
				msg, _ := consumer.ReadMessage(3 * time.Second)
				if msg != nil {
					log.Printf("message: %s topic: %d\n", string(msg.Value), msg.TopicPartition.Partition)
				}
				sleepSec := 20
				// trigger nuclei
				log.Printf("Sleeping for %d sec\n", sleepSec)
				time.Sleep(time.Duration(sleepSec) * time.Second)

			}
		}

	}()

	log.Println("press ctrl+c to exit")
	<-consumerChannel
	consumer.Close()
	log.Println("exited...")

}

func producer(kafkaEndpoint, consumerGroup, topic string) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaEndpoint,
		"group.id":          consumerGroup,
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		fmt.Printf("Failed to create producer: %s", err)
		os.Exit(1)
	}

	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Failed to deliver message: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Produced event to topic %s: key = %-10s value = %s\n",
						*ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))
				}
			}
		}
	}()

	dataKey := [...]string{"key1", "key2", "key3", "key4", "key5", "key6"}
	dataValue := [...]string{"value1", "value2", "value3", "value4", "value5"}

	for n := 0; n < 1; n++ {
		for n := 0; n < 100; n++ {
			key := dataKey[rand.Intn(len(dataKey))]
			data := dataValue[rand.Intn(len(dataValue))]
			producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: int32(n)},
				Key:            []byte(key),
				Value:          []byte(data),
			}, nil)
		}
		// Wait for all messages to be delivered
		sleepSec := 10
		log.Printf("Sleeping for %d sec\n", sleepSec)
		time.Sleep(time.Duration(sleepSec) * time.Second)
		producer.Flush(15 * 1000)
	}

	producer.Close()
}
