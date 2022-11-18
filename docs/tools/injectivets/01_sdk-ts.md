<!--
order: 1
title: sdk-ts
-->

# sdk-ts

Within this 

## Requirements

- Golang v1.16.1 - go1.17.1 linux/amd64
- Ensure your GOPATH and GOBIN environment variables are set up correctly.
- Linux users: install build-essential.
- 8-core (4 physical core), x86_64 architecture processor
- 32 GB RAM (or equivalent swap file set up)
- 1 TB of storage space

## Option 1: From binary

The easiest way to install `injectived` and Injective core is by downloading a pre-built binary for your operating system. Download the Injective Chain Staking-40021-1652947015 binaries from the official injective-chain-releases.

```
wget https://github.com/InjectiveLabs/injective-chain-releases/releases/download/v0.4.19-1652947015/linux-amd64.zip
```

This zip file will contain three binaries and a virtual machine:
- **`injectived`** - the Injective Chain daemon
- **`peggo`** - the Injective Chain ERC-20 bridge relayer daemon
- **`injective-exchange`** - the Injective Exchange daemon
- **`libwasmvm.x86_64.so`** - the wasm virtual machine which is needed to execute smart contracts.

Unzip and add `injectived`, `injective-exchange` and `peggo` to your `/usr/bin`. Also add `libwasmvm.x86_64.so` to user library path `/usr/lib`.

```
unzip linux-amd64.zip
sudo mv injectived peggo injective-exchange /usr/bin
sudo mv libwasmvm.x86_64.so /usr/lib
```

Check your binary version by running following commands.

```
injectived version
peggo version
injective-exchange version
```

Confirm your version matches the output below
```
injectived version
Version dev (f32e524)

peggo version
Version dev (b5c188c)

injective-exchange version
Version dev (ca1da5e)
```

## Option 2: From source

Note: you will only install `injectived` but not `injective-exchange`, `peggo` or `libwasmvm.x86_64.so` using this option.

### Get the Injective core source code

Use git to retrieve [Injective core](https://github.com/InjectiveLabs/injective-core).

Clone the injective repo:

```protobuf
git clone https://github.com/InjectiveLabs/injective-core
```

### Build Injective core from source

Build Injective core, and install the `injectived` executable to your GOPATH environment variable.

```
cd injective-core
make install
```

### Verify your Injective core installation

Verify that Injective core is installed correctly.

```
injectived version
```

