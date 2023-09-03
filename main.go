package seni

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/valyala/fasthttp"
)

type (
	Seni struct {
		server *fasthttp.Server
		stack  [][]*route
		pool   sync.Pool
	}
	Group struct {
		app    *Seni
		prefix string
	}
	routeParser struct {
		Segs   []paramSeg
		Params []string
	}
	paramSeg struct {
		Const      string
		Param      string
		IsParam    bool
		IsOptional bool
		IsLast     bool
	}
	route struct {
		Method      string
		Path        string
		Params      []string
		Handlers    []Handler
		routeParser routeParser
		root        bool
		use         bool
	}
	Ctx struct {
		app          *Seni
		Fasthttp     *fasthttp.RequestCtx
		path         string
		method       string
		route        *route
		values       []string
		indexRoute   int
		indexHandler int
	}
	Handler = func(*Ctx)
)

var (
	methodINT = map[string]int{
		MethodGet:     0,
		MethodHead:    1,
		MethodPost:    2,
		MethodPut:     3,
		MethodDelete:  4,
		MethodConnect: 5,
		MethodOptions: 6,
		MethodTrace:   7,
		MethodPatch:   8,
	}
)

// HTTP methods were copied from net/http.
const (
	MethodGet     = "GET"     // RFC 7231, 4.3.1
	MethodHead    = "HEAD"    // RFC 7231, 4.3.2
	MethodPost    = "POST"    // RFC 7231, 4.3.3
	MethodPut     = "PUT"     // RFC 7231, 4.3.4
	MethodPatch   = "PATCH"   // RFC 5789
	MethodDelete  = "DELETE"  // RFC 7231, 4.3.5
	MethodConnect = "CONNECT" // RFC 7231, 4.3.6
	MethodOptions = "OPTIONS" // RFC 7231, 4.3.7
	MethodTrace   = "TRACE"   // RFC 7231, 4.3.8
)

// App ///////////////////////////

func New() *Seni {
	s := &Seni{
		stack: make([][]*route, len(methodINT)),
		pool: sync.Pool{
			New: func() any {
				return new(Ctx)
			},
		},
	}
	s.init()
	return s
}

func (s *Seni) Listen(address string) error {
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		panic("listen error: " + err.Error())
	}
	return s.server.Serve(listener)
}

func (s *Seni) Shutdown() error {
	if s.server == nil {
		return fmt.Errorf("Server is not running")
	}
	return s.server.Shutdown()
}

func (s *Seni) init() *Seni {
	s.server = &fasthttp.Server{
		Handler: s.handler,
	}
	return s
}

// methods ///////////////////////////

func (s *Seni) Get(path string, handlers ...Handler) {
	s.register("GET", path, handlers...)
}

func (s *Seni) Post(path string, handlers ...Handler) {
	s.register("POST", path, handlers...)
}

func (s *Seni) Put(path string, handlers ...Handler) {
	s.register("PUT", path, handlers...)
}

func (s *Seni) Delete(path string, handlers ...Handler) {
	s.register("DELETE", path, handlers...)
}

func (s *Seni) Use(handlers ...Handler) {
	s.register("USE", "", handlers...)
}

func (s *Seni) Group(prefix string, handlers ...Handler) *Group {
	if len(handlers) > 0 {
		s.register("USE", prefix, handlers...)
	}
	return &Group{prefix: prefix, app: s}
}

// group ///////////////////////////

func (g *Group) Get(path string, handlers ...Handler) {
	g.app.Get(
		getGroupPath(g.prefix, path),
		handlers...,
	)
}

func (g *Group) Post(path string, handlers ...Handler) {
	g.app.Post(
		getGroupPath(g.prefix, path),
		handlers...,
	)
}

func (g *Group) Put(path string, handlers ...Handler) {
	g.app.Put(
		getGroupPath(g.prefix, path),
		handlers...,
	)
}

func (g *Group) Delete(path string, handlers ...Handler) {
	g.app.Delete(
		getGroupPath(g.prefix, path),
		handlers...,
	)
}

func (g *Group) Group(prefix string, handlers ...Handler) *Group {
	// prefix = getGroupPath(g.prefix, prefix)
	// if len(handlers) > 0 {
	// 	g.app.register("USE", prefix, handlers...)
	// }
	// return &Group{prefix: prefix, app: g.app}
	return g.app.Group(
		getGroupPath(g.prefix, prefix),
		handlers...,
	)
}

func getGroupPath(prefix, path string) string {
	if path == "/" {
		return prefix
	}
	return TrimRight(prefix, '/') + path
}

// router ///////////////////////////

func (s *Seni) register(method string, path string, handlers ...Handler) {
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	pr := parseRoute(path)
	r := &route{
		Method:      method,
		Path:        path,
		Params:      pr.Params,
		Handlers:    handlers,
		routeParser: pr,
		root:        path == "/",
		use:         method == "USE",
	}

	if method == "USE" {
		for _, i := range methodINT {
			s.stack[i] = append(s.stack[i], r)
		}
		return
	}

	i := methodINT[method]
	s.stack[i] = append(s.stack[i], r)
}

func (s *Seni) handler(fctx *fasthttp.RequestCtx) {
	ctx := s.acquireCtx(fctx)
	s.next(ctx)
	s.releaseCtx(ctx)
}

func (s *Seni) next(ctx *Ctx) bool {
	i := methodINT[ctx.method]
	lens := len(s.stack[i]) - 1
	for ctx.indexRoute < lens {
		ctx.indexRoute++

		route := s.stack[i][ctx.indexRoute]

		match, values := route.match(ctx.path)
		if !match {
			continue
		}
		ctx.route = route
		ctx.values = values
		// Execute first handler of route
		ctx.indexHandler = 0
		route.Handlers[ctx.indexHandler](ctx)
		return true
	}

	ctx.Status(404).SendString("Not Found")
	return false
}

func (r *route) match(path string) (bool, []string) {
	values := []string{}
	if r.root && path == "" {
		return true, values
	}
	if len(r.Params) > 0 {
		params, ok := r.routeParser.getMatch(path, r.use)
		if ok {
			return true, params
		}
	}
	if r.use {
		if r.root {
			return true, values
		}
		if strings.HasPrefix(path, r.Path) {
			return true, values
		}
	}
	return r.Path == path, values
}

// params ///////////////////////////

func parseRoute(pattern string) routeParser {
	var patternCount int
	splitted := []string{""}
	if pattern != "" {
		splitted = strings.Split(pattern, "/")[1:]
	}
	patternCount = len(splitted)

	var out = make([]paramSeg, patternCount)
	var params []string
	var segIndex int
	for i := 0; i < patternCount; i++ {
		paramLen := len(splitted[i])
		if paramLen == 0 {
			continue
		}

		if splitted[i][0] == ':' {
			out[segIndex] = paramSeg{
				Param:      paramTrimer(splitted[i]),
				IsParam:    true,
				IsOptional: splitted[i][paramLen-1] == '?',
			}
			params = append(params, out[segIndex].Param)
		} else {
			if segIndex > 0 && out[segIndex-1].IsParam == false {
				// combine const seg
				segIndex--
				out[segIndex].Const += "/" + splitted[i]
			} else {
				out[segIndex] = paramSeg{
					Const:   splitted[i],
					IsParam: false,
				}
			}
		}
		segIndex++
	}

	if segIndex == 0 {
		out[segIndex] = paramSeg{}
		segIndex++
	}
	out[segIndex-1].IsLast = true

	return routeParser{
		Segs:   out[:segIndex:segIndex],
		Params: params,
	}
}

func paramTrimer(param string) string {
	end := len(param)
	if param[0] != ':' { // is not a param
		return param
	}
	if param[end-1] == '?' {
		end--
	}
	return param[1:end]
}

func (p *routeParser) getMatch(path string, partialCheck bool) ([]string, bool) {
	params := []string{}
	if len(path) > 0 {
		path = path[1:]
	}
	for _, segment := range p.Segs {
		partLen := len(path)
		var i int
		if segment.IsParam {
			i = strings.IndexByte(path, '/')
			if i == -1 {
				i = partLen
			}
			params = append(params, path[:i])
		} else {
			i = len(segment.Const)
			if partLen < i {
				return nil, false
			} else if i == 0 && partLen > 0 {
				return nil, false
			} else if path[:i] != segment.Const {
				return nil, false
			} else if partLen > i && path[i] != '/' {
				return nil, false
			}
		}

		if partLen > 0 {
			j := i + 1
			if segment.IsLast || partLen < j {
				j = i
			}
			path = path[j:]
		}
	}

	if len(path) != 0 && !partialCheck {
		return nil, false
	}

	return params, true
}

// context ///////////////////////////

func (s *Seni) acquireCtx(fctx *fasthttp.RequestCtx) *Ctx {
	ctx := s.pool.Get().(*Ctx)
	ctx.app = s
	ctx.Fasthttp = fctx
	ctx.path = string(fctx.URI().Path())
	ctx.method = string(fctx.Request.Header.Method())
	ctx.indexRoute = -1
	ctx.indexHandler = 0
	ctx.prettifyPath()
	return ctx
}

func (s *Seni) releaseCtx(ctx *Ctx) {
	ctx.app = nil
	ctx.values = nil
	ctx.route = nil
	ctx.Fasthttp = nil
	s.pool.Put((ctx))
}

func (ctx *Ctx) Next() {
	ctx.indexHandler++
	if ctx.indexHandler < len(ctx.route.Handlers) {
		ctx.route.Handlers[ctx.indexHandler](ctx)
	} else {
		ctx.app.next(ctx)
	}
}

func (ctx *Ctx) Write(bodies ...string) *Ctx {
	for i := range bodies {
		body := bodies[i]
		ctx.Fasthttp.Response.AppendBodyString(body)
	}
	return ctx
}

func (ctx *Ctx) SendString(body string) {
	ctx.Fasthttp.Response.SetBodyString(body)
}

func (ctx *Ctx) Status(status int) *Ctx {
	ctx.Fasthttp.Response.SetStatusCode(status)
	return ctx
}

func (ctx *Ctx) Params(key string) string {
	if ctx.route.Params == nil {
		return ""
	}
	for i := range ctx.route.Params {
		if ctx.route.Params[i] == key {
			return ctx.values[i]
		}
	}
	return ""
}

func (ctx *Ctx) Query(key string, defaultValue string) string {
	v := string(ctx.Fasthttp.QueryArgs().Peek(key))
	if len(v) == 0 {
		v = defaultValue
	}
	return v
}

func (ctx *Ctx) FormValue(key string, defaultValue string) string {
	v := string(ctx.Fasthttp.FormValue(key))
	if len(v) == 0 {
		v = defaultValue
	}
	return v
}

func (ctx *Ctx) prettifyPath() {
	ctx.path = TrimRight(ctx.path, '/')
}
