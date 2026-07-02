module github.com/exchange-grpc

go 1.25.0

require (
	github.com/exchange-grpc/orderservice v0.0.0
	github.com/exchange-grpc/proto v0.0.0
	github.com/exchange-grpc/shared v0.0.0
	github.com/exchange-grpc/spotservice v0.0.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/exchange-grpc/orderservice => ./orderservice
	github.com/exchange-grpc/proto => ./proto
	github.com/exchange-grpc/shared => ./shared
	github.com/exchange-grpc/spotservice => ./spotservice
)
