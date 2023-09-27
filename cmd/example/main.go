package main

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		panic(err)
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: Handler(redisClient),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("server unexpected stopped: %s", err)
		}
	}()

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Kill, os.Interrupt)
	<-stopSig
}
