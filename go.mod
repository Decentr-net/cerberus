module github.com/Decentr-net/cerberus

go 1.15

replace github.com/docker/docker => github.com/docker/engine v0.0.0-20190717161051-705d9623b7c1 // fix logrus for testcontainers

require (
	github.com/Decentr-net/decentr v1.2.2
	github.com/Decentr-net/logrus v0.7.1
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535
	github.com/cosmos/cosmos-sdk v0.39.2
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/getsentry/sentry-go v0.9.0 // indirect
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/go-cmp v0.5.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jessevdk/go-flags v1.4.0
	github.com/minio/minio-go/v7 v7.0.5
	github.com/minio/sio v0.2.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/tendermint/go-amino v0.15.1
	github.com/tendermint/tendermint v0.33.9
	github.com/testcontainers/testcontainers-go v0.8.0
	github.com/tomasen/realip v0.0.0-20180522021738-f0c99a92ddce
	golang.org/x/net v0.0.0-20201010224723-4f7140c49acb
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/genproto v0.0.0-20200726014623-da3ae01ef02d // indirect
	gopkg.in/yaml.v2 v2.3.0
)
