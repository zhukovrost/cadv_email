package main

import (
	"github.com/zhukovrost/cadv_email/internal/app"
	"github.com/zhukovrost/cadv_email/internal/config"
	"log"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	app.Run(cfg)
}
