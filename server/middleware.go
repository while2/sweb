package server

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/mijia/sweb/log"
	"golang.org/x/net/context"
)

type Middleware interface {
	ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context
}

type MiddleFn func(context.Context, http.ResponseWriter, *http.Request, Handler) context.Context

func (m MiddleFn) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context {
	return m(ctx, w, r, next)
}

type RecoveryWare struct {
	printStack bool
	stackAll   bool
	stackSize  int
}

func (m *RecoveryWare) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			stack := make([]byte, m.stackSize)
			stack = stack[:runtime.Stack(stack, m.stackAll)]
			log.Errorf("PANIC: %s\n%s", err, stack)
			if m.printStack {
				fmt.Fprintf(w, "PANIC: %s\n%s", err, stack)
			}
		}
	}()

	return next(ctx, w, r)
}

func NewRecoveryWare(printStack bool) Middleware {
	return &RecoveryWare{
		printStack: printStack,
		stackAll:   printStack,
		stackSize:  1024 * 8,
	}
}

type StatWare struct {
	ignoredPrefixes []string
}

func (m *StatWare) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context {
	start := time.Now()
	newCtx := next(ctx, w, r)
	res := w.(ResponseWriter)
	urlPath := r.URL.Path
	if res.Status() >= 400 {
		log.Warnf("Request %q %q, status=%v, size=%d, duration=%v",
			r.Method, r.URL.Path, res.Status(), res.Size(), time.Since(start))
	} else {
		ignored := false
		for _, prefix := range m.ignoredPrefixes {
			if strings.HasPrefix(urlPath, prefix) {
				ignored = true
				break
			}
		}
		if !ignored {
			log.Infof("Request %q %q, status=%v, size=%d, duration=%v",
				r.Method, r.URL.Path, res.Status(), res.Size(), time.Since(start))
		}
	}
	return newCtx
}

func NewStatWare(prefixes ...string) Middleware {
	return &StatWare{prefixes}
}

type _OnionLayer struct {
	handler Middleware
	next    *_OnionLayer
}

func (m _OnionLayer) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	return m.handler.ServeHTTP(ctx, w, r, m.next.ServeHTTP)
}

func buildOnion(wares []Middleware) _OnionLayer {
	var next _OnionLayer
	if len(wares) == 0 {
		return hollowOnion()
	} else if len(wares) > 1 {
		next = buildOnion(wares[1:])
	} else {
		next = hollowOnion()
	}
	return _OnionLayer{wares[0], &next}
}

func hollowOnion() _OnionLayer {
	fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context {
		return ctx
	}
	return _OnionLayer{MiddleFn(fn), &_OnionLayer{}}
}
