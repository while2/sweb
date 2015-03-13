package server

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mijia/sweb/log"
	"github.com/stretchr/graceful"
	"golang.org/x/net/context"
)

type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context

const (
	kHrParamsKey     = "inter_ctx_key_hrparams"
	kGracefulTimeout = 10
)

type Server struct {
	Context context.Context
	wares   []Middleware
	router  *httprouter.Router
	debug   bool
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

func (s *Server) Middleware(ware Middleware) {
	s.wares = append(s.wares, ware)
}

func (s *Server) Handle(method, path string, handle Handler) {
	s.router.Handle(method, path, s.hrAdapt(handle))
}

func (s *Server) Get(path string, handle Handler) {
	s.Handle("GET", path, handle)
}

func (s *Server) Post(path string, handle Handler) {
	s.Handle("POST", path, handle)
}

func (s *Server) Put(path string, handle Handler) {
	s.Handle("PUT", path, handle)
}

func (s *Server) Patch(path string, handle Handler) {
	s.Handle("Patch", path, handle)
}

func (s *Server) Head(path string, handle Handler) {
	s.Handle("HEAD", path, handle)
}

func (s *Server) Delete(path string, handle Handler) {
	s.Handle("DELETE", path, handle)
}

func (s *Server) NotFound(handle Handler) {
	if handle != nil {
		h := s.hrAdapt(handle)
		s.router.NotFound = func(w http.ResponseWriter, r *http.Request) {
			h(w, r, nil)
		}
	}
}

func (s *Server) hrAdapt(fn Handler) httprouter.Handle {
	core := func(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context {
		// we are inside the onion core, so the next would be ignored
		return fn(ctx, w, r)
	}
	handler := buildOnion(append(s.wares, MiddleFn(core)))
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := s.Context
		if len(params) > 0 {
			ctx = newContextWithParams(ctx, params)
		}
		handler.ServeHTTP(ctx, NewResponseWriter(w), r)
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
	if isDebug {
		log.EnableDebug()
	}
	srv := &Server{
		Context: ctx,
		wares:   []Middleware{},
		router:  httprouter.New(),
		debug:   isDebug,
	}
	return srv
}

func newContextWithParams(ctx context.Context, params httprouter.Params) context.Context {
	return context.WithValue(ctx, kHrParamsKey, params)
}
