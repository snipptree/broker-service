package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const webPort = "8080"

type Config struct {
	// Rabbit *amqp.Connection
}

func main() {
	// try to connect to rabbitMq
	// 	rabbitConn, err := connect()
	// 	if err != nil {
	// 		log.Printf("error connecting to rabbitmq: %v", err)
	// 		os.Exit(1)
	// 	}
	// 	defer rabbitConn.Close()
	//
	app := Config{
		// Rabbit: rabbitConn,
	}

	log.Printf("Starting broker service on PORT %s", webPort)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.Routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func connect() (*amqp.Connection, error) {
	var counts int32
	var backOff = 1 * time.Second
	var connection *amqp.Connection

	// dont continue until our rabbit is ready
	for {
		c, err := amqp.Dial("amqp://guest:guest@rabbitmq")
		if err != nil {
			log.Println("rabbitmq not yet ready...")
			counts++
		} else {
			connection = c
			log.Println("Connected to Rabbit MQ!")
			break
		}

		if counts > 5 {
			log.Printf("error connecting to rabbitmq: %v", err)
		}

		backOff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		log.Println("backing off...")
		time.Sleep(backOff)
		continue
	}
	return connection, nil
}
