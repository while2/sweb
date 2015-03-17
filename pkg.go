/*
Package sweb provides a in-house go web server skeleton and implementation, supports:

  * Server Graceful shutdown
  * Wrap httprouter for the named parameter support, e.g. "/hello/:name"
  * Middlware, and two basic middleware included: RecoveryWare, StatWare
  * Render for composable html templates
  * Single context object for each request derived from base context
  * Server named routes and reverse helper funcs

Basic Usage:

	// init a parent context for the server
	ctx := context.WithValue(context.Background(), "dbConn", db)
	srv := server.New(ctx, false)

	// hook some middlewares
	srv.Middleware(server.NewRecoveryWare(false))
	srv.Middleware(server.NewStatWare())

	// add the routings
	srv.Get("/hello/:name", "Hello", HelloHandler)

	// Run the server supporting the graceful shutdown by default
	srv.Run(":9000")

A sweb handler would be like:

	func Hello(ctx context.Context, w http.ResponseWriter, request *http.Request) context.Context {
		name := server.Params(ctx, "name")
		userId := ctx.Value("userId").(int)
		fmt.Fprintf(w, "Hello, %q, userId = %d", name, userId)
		return ctx
	}

A basic middleware would be like:

	type IncrMiddleware struct {
	}

	func (m *IncrMiddleware) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next server.Handler) context.Context {
		userId, ok := ctx.Value("userId").(int)
		if !ok {
			userId = 1
		}
		newCtx := context.WithValue(ctx, "userId", userId+1)
		return next(newCtx, w, r)
		// Also can do some hooking staff after the handler processed the request
	}

Please checkout the detail godoc for the subpackages:
  * http://godoc.org/github.com/mijia/sweb/server
  * http://godoc.org/github.com/mijia/sweb/render
*/
package sweb
