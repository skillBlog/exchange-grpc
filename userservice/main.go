package main

import (
	"github.com/exchange-grpc/userservice/pkg/apprunner"
	"github.com/exchange-grpc/userservice/pkg/config"
	"github.com/exchange-grpc/userservice/pkg/envloader"
)

func main() {
	envloader.LoadEnv()
	cfg := config.LoadConfig()
	apprunner.NewAppRunner(cfg).Run()
}
