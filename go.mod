module github.com/Decentr-net/cerberus

go 1.16

require (
	github.com/Decentr-net/ariadne v1.1.1
	github.com/Decentr-net/decentr v1.5.0
	github.com/Decentr-net/go-api v0.1.0
	github.com/Decentr-net/go-broadcaster v0.1.0
	github.com/Decentr-net/logrus v0.7.2
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535
	github.com/aws/aws-sdk-go v1.36.30
	github.com/cosmos/cosmos-sdk v0.44.3
	github.com/disintegration/imaging v1.6.2
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-chi/cors v1.1.1
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.2.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/jmoiron/sqlx v1.3.3
	github.com/lib/pq v1.10.4
	github.com/minio/minio-go/v7 v7.0.5
	github.com/minio/sio v0.2.0
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.14
	github.com/testcontainers/testcontainers-go v0.11.1
	golang.org/x/net v0.0.0-20210903162142-ad29c8ab022f
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
