package envloader

import sharedenv "github.com/exchange-grpc/shared/envloader"

// LoadEnv загружает .env из рабочей директории (и на уровень выше при запуске из подпапки).
func LoadEnv() {
	sharedenv.Load(".env", "../.env")
}
