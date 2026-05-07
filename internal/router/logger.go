package router

import (
	"time"

	"go.uber.org/zap"
)

func ZapLogger(log *zap.Logger) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) {
			start := time.Now()
			next(c)
			gc, ok := c.(*ginContext)
			if !ok {
				log.Warn("ZapLogger: unexpected context type, request not logged")
				return
			}
			fields := []zap.Field{
				zap.String("method", gc.ctx.Request.Method),
				zap.String("path", gc.ctx.Request.URL.Path),
				zap.Int("status", gc.ctx.Writer.Status()),
				zap.Duration("latency", time.Since(start)),
				zap.String("ip", gc.ctx.ClientIP()),
			}
			if val, exists := gc.ctx.Get(ContextUserIDKey); exists {
				if id, ok := val.(string); ok && id != "" {
					fields = append(fields, zap.String("public_id", id))
				}
			}
			log.Info("request", fields...)
		}
	}
}
