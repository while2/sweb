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

Please checkout the detail godoc for the subpackages:
  * http://godoc.org/github.com/mijia/sweb/server
  * http://godoc.org/github.com/mijia/sweb/render
*/
package sweb
