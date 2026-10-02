package main

import (
	"fmt"
	"os"

	"github.com/exchange-grpc/orderservice/pkg/apprunner"
	"github.com/exchange-grpc/orderservice/pkg/config"
	"github.com/exchange-grpc/orderservice/pkg/envloader"
)

func main() {
	envloader.LoadEnv()
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	apprunner.NewAppRunner(cfg).Run()
}
