module github.com/Decentr-net/cerberus

go 1.15

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20190717161051-705d9623b7c1 // fix logrus for testcontainers

require (
	github.com/Decentr-net/decentr v1.2.2
	github.com/Decentr-net/go-api v0.0.3
	github.com/Decentr-net/go-broadcaster v0.0.1
	github.com/Decentr-net/logrus v0.7.2
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535
	github.com/cosmos/cosmos-sdk v0.39.2
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-chi/cors v1.1.1
	github.com/golang/mock v1.4.4
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jessevdk/go-flags v1.4.0
	github.com/minio/minio-go/v7 v7.0.5
	github.com/minio/sio v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/go-amino v0.15.1
	github.com/tendermint/tendermint v0.33.9
	github.com/testcontainers/testcontainers-go v0.8.0
	golang.org/x/net v0.0.0-20201010224723-4f7140c49acb
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	gopkg.in/yaml.v2 v2.3.0
)
