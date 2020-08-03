# Cerberus
![img](https://img.shields.io/docker/build/decentr/cerberus)

The Cerberus encrypts data and pushes it into [ipfs](https://ipfs.io) 

## Run
### Docker
#### Local image
```
make image
docker run -it --rm -e "HOST=0.0.0.0" -e "PORT=7070" -e "LOG_LEVEL=debug" -p "7070:7070" cerberus-local
```
### From source
```
go run cmd/cerberus/main.go \
    --host=0.0.0.0 \
    --port=8080 \
    --log.level=debug
```
### Run Cerberus and [ipfs](https://ipfs.io) with docker-compose
#### Local image
```
make image
docker-compose -f scripts/docker-compose.yml up
```
## Parameters
| CLI param         | Environment var          | Default | Description
|---------------|------------------|---------------|---------------------------------
| host         | HOST         | 0.0.0.0  | host to bind server
| port    | PORT    | 8080  | port to listen
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
