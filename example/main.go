package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mijia/sweb/server"
	"golang.org/x/net/context"
)

type IncrMiddleware struct {
}

func (m *IncrMiddleware) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next server.Handler) context.Context {
	userId, ok := ctx.Value("userId").(int)
	if !ok {
		userId = 1
	}
	newCtx := context.WithValue(ctx, "userId", userId+1)
	return next(newCtx, w, r)
}

func AuthHandler(handler server.Handler) server.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
		userId, ok := ctx.Value("userId").(int)
		if !ok {
			userId = 1
		}
		ctx = context.WithValue(ctx, "userId", userId+1)
		authed := true
		if authed {
			handler(ctx, w, r)
		}
		return ctx
	}
}

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
	srv.Middleware(&IncrMiddleware{})
	srv.Get("/hello/:name", "Hello", Hello)
	srv.Get("/auth/:name", "Hello", AuthHandler(Hello))

	log.Fatal(srv.Run(":9000"))
}
