package main

import (
	"context"
	"edbtest/supervisor/message"
	"edbtest/supervisor/supervisor"
	"log"
	"time"
)

func main() {
	worker := func(ctx context.Context, message chan message.Message) error {
		log.Println("Worker started")
		time.Sleep(1 * time.Second)

		panic("Worker panicked")
	}

	supervisor := supervisor.NewSupervisor(worker, 3, 10*time.Second, 5*time.Second)
	supervisor.Start(context.Background())
	time.Sleep(30 * time.Second)
	supervisor.Stop()

	log.Println("Program finished")
}
