package main

import (
	"context"
	"log"
	"time"

	"github.com/cooldogedev/spectral"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	conn, err := spectral.Dial(ctx, "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.CloseWithError(0, "")

	stream, err := conn.OpenStream(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	message := "Hello, World!"
	_, err = stream.Write([]byte(message))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Sent: %s", message)

	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Received echo: %s", string(buf[:n]))
	select {
	case <-conn.Context().Done():
	}
}
