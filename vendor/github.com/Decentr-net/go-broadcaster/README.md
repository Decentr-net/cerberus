# broadcaster
![img](https://img.shields.io/github/go-mod/go-version/Decentr-net/go-broadcaster) ![img](https://img.shields.io/github/v/tag/Decentr-net/go-broadcaster?label=version)

Package which simply broadcasting messages to cosmos based blockchain node

## Example

```
import (
    "github.com/Decentr-net/go-broadcaster"
    "github.com/Decentr-net/decentr/app"
    "github.com/Decentr-net/decentr/community"
)

func main() {
    b, err := broadcaster.New(app.MakeCodec(), broadcaster.Config{
        CLIHome:            "~/.decentrcli",
        KeyringBackend:     "test",
        KeyringPromptInput: "",
        NodeURI:            "zeus.testnet.decentr.xyz:26656",
        BroadcastMode:      "sync",
        From:               "hera",
        ChainID:            "testnet",
        GenesisKeyPass:     "12345678",
    })

    if err != nil {
        panic(err)
    }

    if err := b.BroadcastMsg(community.MsgFollow{
        Owner: b.From(),
        Whom:  zeus,
    }, "follow me back"); err != nil {
        panic(err)
    }
}
```
