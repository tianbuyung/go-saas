package router

func registerHealth(r Router) {
	r.GET("/health", func(c Context) {
		c.JSON(200, map[string]string{
			"status": "ok",
		})
	})
}
