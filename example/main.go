package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mijia/sweb/server"
	"golang.org/x/net/context"
)

func Hello(ctx context.Context, w http.ResponseWriter, request *http.Request) context.Context {
	name := server.Params(ctx, "name")
	userId := ctx.Value("userId").(int)
	fmt.Fprintf(w, "Hello, %q, userId = %d", name, userId)
	return ctx
}

func main() {
	ctx := context.WithValue(context.Background(), "userId", 1)
	srv := server.New(ctx, true)

	srv.Middleware(server.NewRecoveryWare(true))
	srv.Middleware(server.NewStatWare())
	srv.Get("/hello/:name", "Hello", Hello)

	log.Fatal(srv.Run(":9000"))
}
