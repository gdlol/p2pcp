# Usage

- [p2pcp](#p2pcp)
  - [p2pcp send](#p2pcp-send)
  - [p2pcp receive](#p2pcp-receive)

## `p2pcp`

```
Usage:
  p2pcp [command]

Available Commands:
  receive     Receives file/directory from remote peer to specified directory
  send        Sends the specified file/directory to remote peer

Flags:
  -d, --debug     show debug logs
  -p, --private   only connect to private networks

Use "p2pcp [command] --help" for more information about a command.
```

## `p2pcp send`

```
Sends the specified file/directory to remote peer

Usage:
  p2pcp send [path] [flags]

Flags:
  -s, --strict   use strict mode, this will generate a long secret for authentication

Global Flags:
  -d, --debug     show debug logs
  -p, --private   only connect to private networks
```

## `p2pcp receive`

```
Receives file/directory from remote peer to specified directory

Usage:
  p2pcp receive id [path] [flags]

Global Flags:
  -d, --debug     show debug logs
  -p, --private   only connect to private networks
```
