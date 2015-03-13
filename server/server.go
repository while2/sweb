package server

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mijia/sweb/log"
	"github.com/stretchr/graceful"
	"golang.org/x/net/context"
)

type Handler func(context.Context, http.ResponseWriter, *http.Request)

const (
	kHrParamsKey     = "inter_ctx_key_hrparams"
	kGracefulTimeout = 10
)

type Server struct {
	Context context.Context

	router *httprouter.Router
	debug  bool
}

func (s *Server) Run(addr string) error {
	srv := &graceful.Server{
		Timeout: kGracefulTimeout * time.Second,
		Server: &http.Server{
			Addr:    addr,
			Handler: s.router,
		},
	}
	log.Infof("Server is listening on %s", addr)
	return srv.ListenAndServe()
}

func (s *Server) Handle(method, path string, handle Handler) {
	s.router.Handle(method, path, s.hrAdapt(handle))
}

func (s *Server) Get(path string, handle Handler) {
	s.Handle("GET", path, handle)
}

func (s *Server) hrAdapt(fn Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := s.Context
		if len(params) > 0 {
			ctx = newContextWithParams(ctx, params)
		}
		// TODO: going through the middlewares
		fn(ctx, w, r)
	}
}

func Params(ctx context.Context, key string) string {
	if params, ok := ctx.Value(kHrParamsKey).(httprouter.Params); !ok {
		return ""
	} else {
		return params.ByName(key)
	}
}

func New(ctx context.Context, isDebug bool) *Server {
	return &Server{
		Context: ctx,
		router:  httprouter.New(),
		debug:   isDebug,
	}
}

func newContextWithParams(ctx context.Context, params httprouter.Params) context.Context {
	return context.WithValue(ctx, kHrParamsKey, params)
}
