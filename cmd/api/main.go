package main

import "saas/internal/bootstrap"

func main() {
	app := bootstrap.NewApp()
	app.Run()
}
