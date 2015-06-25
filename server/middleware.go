package server

import (
	"expvar"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/mijia/sweb/log"
	"github.com/paulbellamy/ratecounter"
	"golang.org/x/net/context"
)

// Middleware is an interface defining the middleware for the server, the middleware should call the next handler to pass the
// request down, or just return a HttpRedirect request and etc.
type Middleware interface {
	ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context
}

// MiddleFn is an adapter to adapt a function to a Middleware interface
type MiddleFn func(context.Context, http.ResponseWriter, *http.Request, Handler) context.Context

// ServeHTTP adapts the Middleware interface.
func (m MiddleFn) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context {
	return m(ctx, w, r, next)
}

// RecoveryWare is the recovery middleware which can cover the panic situation.
type RecoveryWare struct {
	printStack bool
	stackAll   bool
	stackSize  int
}

// ServeHTTP implements the Middleware interface, just recover from the panic. Would provide information on the web page
// if in debug mode.
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

// NewRecoveryWare returns a new recovery middleware. Would log the full stack if enable the printStack.
func NewRecoveryWare(flags ...bool) Middleware {
	stackFlags := []bool{false, false}
	for i := range flags {
		if i >= len(stackFlags) {
			break
		}
		stackFlags[i] = flags[i]
	}
	return &RecoveryWare{
		printStack: stackFlags[0],
		stackAll:   stackFlags[1],
		stackSize:  1024 * 8,
	}
}

// StatWare is the statistics middleware which would log all the access and performation information.
type StatWare struct {
	ignoredPrefixes []string
}

// ServeHTTP implements the Middleware interface. Would log all the access, status and performance information.
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

// NewStatWare returns a new StatWare, some ignored urls can be specified with prefixes which would not be logged.
func NewStatWare(prefixes ...string) Middleware {
	return &StatWare{prefixes}
}

// RuntimeWare is the statictics middleware which would collect some basic qps, 4xx, 5xx data information
type RuntimeWare struct {
	ignoredUrls []string
	latency     [20]string

	cQps *ratecounter.RateCounter
	c4xx *ratecounter.RateCounter
	c5xx *ratecounter.RateCounter

	hitsQps      *expvar.Int
	hits4xx      *expvar.Int
	hits5xx      *expvar.Int
	hitsServed   *expvar.String
	numGoroutine *expvar.Int
}

func (m *RuntimeWare) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next Handler) context.Context {
	start := time.Now()
	newCtx := next(ctx, w, r)

	urlPath := r.URL.Path
	statusCode := w.(ResponseWriter).Status()
	if statusCode >= 500 {
		m.c5xx.Incr(1)
		m.hits5xx.Set(m.c5xx.Rate())
	} else if statusCode >= 400 {
		m.c4xx.Incr(1)
		m.hits4xx.Set(m.c4xx.Rate())
	}

	ignoreQps := false
	for _, prefix := range m.ignoredUrls {
		if strings.HasPrefix(urlPath, prefix) {
			ignoreQps = true
			break
		}
	}
	if !ignoreQps {
		m.cQps.Incr(1)
		rate := m.cQps.Rate()
		m.hitsQps.Set(rate)
		index := (rate - 1) % int64(len(m.latency))
		m.latency[int(index)] = time.Since(start).String()
		m.hitsServed.Set(strings.Join(m.latency[:], ", "))
	}
	m.numGoroutine.Set(int64(runtime.NumGoroutine()))
	return newCtx
}

func NewRuntimeWare(prefixes []string) Middleware {
	expvar.NewString("at_server_start").Set(time.Now().Format("2006-01-02 15:04:05"))
	return &RuntimeWare{
		ignoredUrls:  prefixes,
		cQps:         ratecounter.NewRateCounter(time.Minute),
		c4xx:         ratecounter.NewRateCounter(5 * time.Minute),
		c5xx:         ratecounter.NewRateCounter(5 * time.Minute),
		hitsQps:      expvar.NewInt("hits_per_minute"),
		hits4xx:      expvar.NewInt("hits_4xx_per_5min"),
		hits5xx:      expvar.NewInt("hits_5xx_per_5min"),
		hitsServed:   expvar.NewString("latency"),
		numGoroutine: expvar.NewInt("goroutine_count"),
	}
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
