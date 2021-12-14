module github.com/Decentr-net/go-api

go 1.15

require (
	github.com/Decentr-net/logrus v0.7.2-0.20210316223658-7a9b48625189
	github.com/cosmos/cosmos-sdk v0.44.3
	github.com/davecgh/go-spew v1.1.1
	github.com/gofrs/uuid v4.1.0+incompatible
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.14
	github.com/tomasen/realip v0.0.0-20180522021738-f0c99a92ddce
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/genproto v0.0.0-20210903162649-d08c68adba83 // indirect
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
