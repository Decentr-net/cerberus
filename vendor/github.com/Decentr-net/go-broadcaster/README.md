# broadcaster
![img](https://img.shields.io/github/go-mod/go-version/Decentr-net/go-broadcaster) ![img](https://img.shields.io/github/v/tag/Decentr-net/go-broadcaster?label=version)

Package which simplifies broadcasting messages to decentr blockchain node

## Example

```
import (
    . "github.com/Decentr-net/decentr/testutil"
    communitytypes "github.com/Decentr-net/decentr/x/community/types"
    "github.com/Decentr-net/go-broadcaster"
)

func main() {
	b, err := broadcaster.New(Config{
		KeyringRootDir:     "~/.decentr",
		KeyringBackend:     "test",
		KeyringPromptInput: "",
		NodeURI:            "http://localhost:26657",
		BroadcastMode:      "sync",
		From:               "jack",
		ChainID:            "local",
	})
	if err != nil {
		panic(err)
	}

	if _, err := b.BroadcastMsg(&communitytypes.MsgFollow{
		Owner: b.From(),
		Whom:  NewAccAddress(),
	}, "follow me back"); err != nil {
		panic(err)
	}
}
```
