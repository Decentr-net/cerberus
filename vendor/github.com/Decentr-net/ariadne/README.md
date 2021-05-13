# Ariadne
![img](https://img.shields.io/github/license/Decentr-net/ariadne) ![img](https://img.shields.io/github/go-mod/go-version/Decentr-net/ariadne) ![img](https://img.shields.io/github/v/tag/Decentr-net/ariadne?label=version)

Ariadne is a library for fetching blocks from cosmos based blockchain node. The library is helpful at off-chain services building.

## Install
```
go get -u github.com/Decentr-net/ariadne
```

## Usage

Short example:

```go
f, err := ariadne.New(nodeAddr, cdc, time.Minute)
if err != nil { panic(err) }

_ = f.FetchBlocks(context.Background(), 1, func (b Block) error {
    fmt.Sprintf("%+v\n", b)
    return nil
})
````

You have to look at detailed example [here](example/example.go) 

## Contributing

Feel free to create issues and send pull requests!

## License

This project is under Apache-2.0 License