# `p2pcp` - Peer to Peer Copy

[CI Badge]: https://img.shields.io/github/actions/workflow/status/gdlol/p2pcp/.github%2Fworkflows%2Fmain.yml
[CI URL]: https://github.com/gdlol/p2pcp/actions/workflows/main.yml
[Codecov Badge]: https://img.shields.io/codecov/c/github/gdlol/p2pcp
[Codecov URL]: https://codecov.io/gh/gdlol/p2pcp
[License Badge]: https://img.shields.io/github/license/gdlol/p2pcp

[![CI Badge][CI Badge]][CI URL]
[![Codecov Badge][Codecov Badge]][Codecov URL]
[![License Badge][License Badge]](LICENSE)

Simple & Secure command line peer-to-peer data transfer tool based on
[libp2p](https://github.com/libp2p/go-libp2p), with peer discovery through DHT/mDNS,
NAT traversal, and no setup.

## Table of Contents

- [Example](#example)
- [Installation (or not)](#installation-or-not)
  - [With Docker/regctl](#with-dockerregctl)
  - [Docker Image](#docker-image)
- [Usage](#usage)
- [Design](#design)
  - [Comparison to `pcp`](#comparison-to-pcp)
  - [Interactive Mode (Default)](#interactive-mode-default)
  - [Strict Mode (--strict)](#strict-mode---strict)
- [Features](#features)
- [Notes](#notes)
- [Acknowledgements](#acknowledgements)
- [References](#references)

## Example

Send:

```
> p2pcp send
Node ID: 5TPXvuyV78S4dnnoUzsVv1Tb3fEXmQpL2E6juf2P9Em9
+-----------------+
|     +B..        |
|   .+o.=         |
|  ..o+o          |
|  o+ +..         |
| E.o+ o.S        |
|...o...+++       |
|. o.*o=ooo.      |
|   *+*o+o        |
|   o++Bo         |
+-----------------+
Please run the following command on the receiver's side:

p2pcp receive f2P9Em9
PIN: 654028

Sending...
Done.
```

Receive:

```
> p2pcp receive f2P9Em9
Enter PIN/token: 900137
Sender ID: 5TPXvuyV78S4dnnoUzsVv1Tb3fEXmQpL2E6juf2P9Em9
Please verify that the following random art matches the one displayed on the sender's side.
+-----------------+
|     +B..        |
|   .+o.=         |
|  ..o+o          |
|  o+ +..         |
| E.o+ o.S        |
|...o...+++       |
|. o.*o=ooo.      |
|   *+*o+o        |
|   o++Bo         |
+-----------------+
Are you sure you want to connect to this sender? [y/N]
y
Receiving...
Done.
```

## Installation (or not)

### With Docker/regctl:

```sh
docker run --rm \
    ghcr.io/regclient/regctl image get-file \
    ghcr.io/gdlol/p2pcp \
    --platform linux/amd64 /p2pcp > p2pcp
chmod +x p2pcp
```

For PowerShell:

```ps1
docker run --rm `
    ghcr.io/regclient/regctl image get-file `
    ghcr.io/gdlol/p2pcp `
    --platform windows/amd64 /p2pcp.exe > p2pcp.exe
```

The list of published platforms can be found at [ghcr.io/gdlol/p2pcp](https://ghcr.io/gdlol/p2pcp).

### Docker Image

You can also use the Docker image directly without installing the binary:

```sh
alias p2pcp='docker run --rm -it \
    --network host \
    -u $(id -u):$(id -g) \
    -v ${PWD}:/data \
    ghcr.io/gdlol/p2pcp'
```

For PowerShell:

```ps1
function p2pcp {
    docker run --rm -it `
    --network host `
    -v ${PWD}:/data `
    ghcr.io/gdlol/p2pcp @args
}
```

## Usage

See [Usage](docs/Usage.md)

## Design

`p2pcp` is forked from [pcp](https://github.com/dennis-tra/pcp) with enhanced libp2p integration and an
updated security model.

### Comparison to `pcp`

Communication security in `pcp` was protected by an extra layer of encryption over the libp2p
stream, with a shared secret consisting of `3` random words from
[BIP39](https://github.com/bitcoin/bips/blob/master/bip-0039/bip-0039-wordlists.md) (`2048^3` combinations).
The strength would be equivalent to a random alphanumeric password of length between `5` and `6`.

The receiver command for `pcp` would look like

```
pcp receive four-random-english-words
```

The first random word (`2048` combinations) will be used for DHT advertisement & peer discovery.

The receiver command for `p2pcp` would look like

```
p2pcp receive id
Enter PIN/token: secret
```

The way `id` and `secret` are generated & used depends on the 2 modes of operation.

### Interactive Mode (Default)

By default, `id` will be the `7` character suffix of the sender's hashed, Base58 encoded libp2p node ID
(`~2 trillion` combinations). The chance of collision (2 nodes sharing the same suffix) should be negligible,
however, deliberate collision would still be possible by a motivated attacker.

To verify the sender's identity with confidence, the receiver is expected to compare the sender's Random Art
(visual representation of the sender's node ID) with the one displayed on the sender's side.

Random Art[[1](#r1)] was popularized by OpenSSH and is commonly seen during the usage of
[ssh-keygen](https://man.openbsd.org/ssh-keygen#l).

`secret` will be a random `6` digits (`1 million` combinations) short passcode and used to authenticate the receiver.
The sender will abort upon any failed attempt of authentication.

### Strict Mode (--strict)

`p2pcp send` can be invoked with a `--strict` flag, in which case the security model becomes cryptographically sound:

`id` will be the full Base58 encoded hash of the sender's libp2p node ID. Since the libp2p node ID is a
cryptographic public key and will be verified by peers, forging the sender's identity would require
breaking the public key algorithm or generating a collision to a cryptographic hash.

`secret` will be a random string with at least `128` bits of entropy.

In strict mode, authentication becomes non-interactive (no need to confirm the Random Art).

On the downside, `id` and `secret` are much longer and less feasible for manual typing.

## Features

- Peer discovery through DHT/mDNS
- NAT traversal with hole-punching and auto relays
- Cross platform (Linux, macOS, Windows)
- Proper handling of special file types
- Sustain across network interruptions

## Notes

- `p2pcp send` will be ready after it successfully advertises itself to the DHT, this is a process that may take
  variable time depending on network conditions. If both sender and receiver are on the same local network,
  the `--private` flag can be used to skip this step.
- Sender and receiver will try to establish a direct connection via hole-punching, if this is unsuccessful,
  the connection will be relayed by other nodes found through DHT and likely heavily rate limited.

## Acknowledgements

- [`go-libp2p`](https://github.com/libp2p/go-libp2p) - The Go implementation of the libp2p Networking Stack.
- [`progressbar`](https://github.com/schollz/progressbar) - A really basic thread-safe progress bar for Golang applications
- [`drunken-bishop`](https://github.com/moul/drunken-bishop) - Drunken Bishop algorithm for Ascii-Art representation of Fingerprint

## References

<!-- spell-checker: ignore Perrig -->

<a id="r1"></a>

[1]. A. Perrig and D. Song, "Hash visualization: A new technique to improve real-world security," in International Workshop on Cryptographic Techniques and E-Commerce, 1999.
