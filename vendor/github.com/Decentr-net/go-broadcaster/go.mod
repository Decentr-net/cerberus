module github.com/Decentr-net/go-broadcaster

go 1.16

require (
	github.com/Decentr-net/decentr v1.5.0
	github.com/cosmos/cosmos-sdk v0.44.3
	github.com/golang/mock v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/spm v0.1.8-0.20211026072440-6f215802f3ec // wait fix for v0.44.3 in tag
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
