package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/knightfall22/Phylax/config"
)

func main() {
	conf := config.LoadConfigurations()
	ctx := context.Background()

	app := Run(ctx, conf)
	defer app.Close()

	var sigs chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
}
