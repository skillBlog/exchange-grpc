package main

import (
	"github.com/exchange-grpc/orderservice/pkg/apprunner"
	"github.com/exchange-grpc/orderservice/pkg/config"
	"github.com/exchange-grpc/orderservice/pkg/envloader"
)

func main() {
	envloader.LoadEnv()
	cfg := config.LoadConfig()
	apprunner.NewAppRunner(cfg).Run()
}
