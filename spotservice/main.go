package main

import (
	"github.com/exchange-grpc/spotservice/pkg/apprunner"
	"github.com/exchange-grpc/spotservice/pkg/config"
	"github.com/exchange-grpc/spotservice/pkg/envloader"
)

func main() {
	envloader.LoadEnv()
	cfg := config.LoadConfig()
	apprunner.NewAppRunner(cfg).Run()
}
