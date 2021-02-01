# Cerberus
![img](https://img.shields.io/docker/cloud/build/decentr/cerberus.svg) ![img](https://img.shields.io/github/go-mod/go-version/Decentr-net/cerberus) ![img](https://img.shields.io/github/v/tag/Decentr-net/cerberus?label=version)

Cerberus is a Decentr oracle. Cerberus stores and validates PDV (private data value).

## Run
### Docker
#### Local image
```
make image
docker run -it --rm -e "HTTP_HOST=0.0.0.0" -e "HTTP_PORT=7070" -e "LOG_LEVEL=debug" -e "S3_ENDPOINT=localhost:9000" -e "S3_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE" -e "S3_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" -e "S3_USE_SSL=false" -e "S3_BUCKET=cerberus" -e "ENCRYPT_KEY=0102030405060708090a0b0c0d0e0f10201f1e1d1c1b1a191817161514131211" -p "7070:7070" cerberus-local
```
#### Docker Compose
```
make image
docker-compose -f scripts/docker-compose.yml up -d
```
### From source
```
go run cmd/cerberus/main.go \
    --http.host=0.0.0.0 \
    --http.port=8080 \
    --s3.endpoint=localhost:9000 \
    --s3.region=us-east-2 \
    --s3.access-key-id=AKIAIOSFODNN7EXAMPLE \
    --s3.secret-access-key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
    --s3.use-ssl=false \
    --s3.bucket=cerberus \
    --encrypt-key=0102030405060708090a0b0c0d0e0f10201f1e1d1c1b1a191817161514131211 \
    --log.level=debug
```

## Parameters
| CLI param         | Environment var          | Default | Description
|---------------|------------------|---------------|---------------------------------
| http.host         | HTTP_HOST         | 0.0.0.0  | host to bind server
| http.port    | HTTP_PORT    | 8080  | port to listen
| http.max-body-size    | HTTP_MAX_BODY_SIZE    | 8000000  | max requests' body size in bytes
| sentry.dsn    | SENTRY_DSN    |  | sentry dsn
| s3.endpoint    | S3_ENDPOINT    | localhost:9000  | s3 storage endpoint
| s3.region    | S3_REGION    |  | s3 storage region
| s3.access-key-id    | S3_ACCESS_KEY_ID    |  | Access KeyID for S3 storage
| s3.secret-access-key    | S3_SECRET_ACCESS_KEY    |   | Secret Key for S3 storage
| s3.use-ssl    | S3_USE_SSL    | false  | do use ssl for S3 storage connection?
| s3.bucket    | S3_BUCKET    | cerberus  | bucket name for S3 storage
| reward-map-config | REWARD_MAP_CONFIG | configs/rewards.yml | path to yaml [config](configs/rewards.yml) with pdv rewards
| min-pdv-count | MIN_PDV_COUNT | 100 | minimal count of pdv to save
| max-pdv-count | MAX_PDV_COUNT | 100 | maximal count of pdv to save
| encrypt-key    | ENCRYPT_KEY    |   | private key for data encryption in hex
| log.level   | LOG_LEVEL   | info  | level of logger (debug,info,warn,error)


## Development
### Makefile
#### Update vendors
Use `make vendor`
#### Install required for development tools
You can check all tools existence with `make check-all` or force installing them with `make install-all` 
##### golangci-lint 1.29.0
Use `make install-linter`
##### swagger v0.25.0
Use `make install-swagger`
##### gomock v1.4.3
Use `make install-mockgen`
#### Build docker image
Use `make image` to build local docker image named `cerberus-local`
#### Build binary
Use `make build` to build for your OS or use `make linux` to build for linux(used in `make image`) 
#### Run tests
Use `make test` to run tests. Also you can run tests with `integration` tag with `make fulltest`
