package main

import (
	"context"
	"log"

	"github.com/cooldogedev/spectral"
)

func main() {
	listener, err := spectral.Listen("127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Echo server started on 127.0.0.1:8080")
	conn, err := listener.Accept(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.CloseWithError(0, "")

	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Received: %s", string(buf[:n]))
	if _, err = stream.Write(buf[:n]); err != nil {
		log.Fatal(err)
	}
	<-conn.Context().Done()
}
