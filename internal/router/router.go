package router

import "context"

type HandlerFunc func(Context)
type MiddlewareFunc func(HandlerFunc) HandlerFunc

type Context interface {
	JSON(code int, data any)
	Param(string) string
	Query(string) string
	BindJSON(any) error
	Set(string, any)
	Get(string) (any, bool)
	GetHeader(key string) string
	Context() context.Context
}

type Router interface {
	GET(path string, handler HandlerFunc)
	POST(path string, handler HandlerFunc)
	Group(prefix string) Router
	Use(mw ...MiddlewareFunc)
	Run(addr string) error
}
