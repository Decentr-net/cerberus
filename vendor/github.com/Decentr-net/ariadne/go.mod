module github.com/Decentr-net/ariadne

go 1.16

require (
	github.com/Decentr-net/decentr v1.5.0
	github.com/cosmos/cosmos-sdk v0.44.3
	github.com/gin-gonic/gin v1.7.0 // indirect; <1.7.0 is vulnerable verison
	github.com/golang/mock v1.6.0
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/spm v0.1.8-0.20211026072440-6f215802f3ec
	github.com/tendermint/tendermint v0.34.14
	google.golang.org/grpc v1.40.0
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
