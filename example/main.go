package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mijia/sweb/server"
	"golang.org/x/net/context"
)

func Hello(ctx context.Context, w http.ResponseWriter, request *http.Request) {
	name := server.Params(ctx, "name")
	fmt.Fprintf(w, "Hello, %q", name)
}

func main() {
	ctx := context.WithValue(context.Background(), "userId", 1)
	srv := server.New(ctx, true)
	srv.Get("/hello/:name", Hello)
	log.Fatal(srv.Run(":9000"))
}
