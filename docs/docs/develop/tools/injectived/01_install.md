---
sidebar_position: 1
title: Install Injectived
---


# Install `injectived` 

`injectived` is the command-line interface and daemon that connects to Injective and enables you to interact with the Injective blockchain. Injective core is the official Golang reference implementation of the Injective node software.

## Requirements

- Golang v1.16.1 - go1.17.1 linux/amd64
- Ensure your GOPATH and GOBIN environment variables are [set up](https://go.dev/wiki/SettingGOPATH).
- Linux users: install build-essential.
- 8 vCPU (4 physical core), x86_64 architecture processor
- 64 GB RAM (or equivalent swap file set up)
- 1 TB of storage space

## Option 1: From binary

The easiest way to install `injectived` and Injective core is by downloading a pre-built binary for your operating system. Download the most recent Injective binaries from the official [injective-chain-releases repo](https://github.com/InjectiveLabs/injective-chain-releases).

:::tip

Make sure to check the releases repo above for the most recent version!

:::

```bash
wget https://github.com/InjectiveLabs/injective-chain-releases/releases/download/v1.x.x-x/linux-amd64.zip
```

This zip file will contain three binaries and a virtual machine:
- **`injectived`** - Injective daemon
- **`peggo`** - Injective ERC-20 bridge relayer daemon
- **`injective-exchange`** - the Injective Exchange daemon
- **`libwasmvm.x86_64.so`** - the wasm virtual machine which is needed to execute smart contracts.

Unzip and add `injectived`, `injective-exchange` and `peggo` to your `/usr/bin`. Also add `libwasmvm.x86_64.so` to user library path `/usr/lib`.

```bash
unzip linux-amd64.zip
sudo mv injectived peggo injective-exchange /usr/bin
sudo mv libwasmvm.x86_64.so /usr/lib
```

Check your binary version by running following commands.

```bash
injectived version
peggo version
injective-exchange version
```

Confirm your version matches the output below

```bash
injectived version
Version dev (f32e524)

peggo version
Version dev (b5c188c)

injective-exchange version
Version dev (ca1da5e)
```

## Option 2: From source

Note: you will only install `injectived` but not `injective-exchange`, `peggo` or `libwasmvm.x86_64.so` using this option. If you are using MacOS you can only install `injectived` from source.

### Get the Injective core source code

Use git to retrieve [Injective core](https://github.com/InjectiveFoundation/injective-core).

Clone the injective repo:

```bash
git clone https://github.com/InjectiveFoundation/injective-core
```

### Build Injective core from source

Build Injective core, and install the `injectived` executable to your GOPATH environment variable.

```bash
cd injective-core
make install
```

### Verify your Injective core installation

Verify that Injective core is installed correctly.

```bash
injectived version
```

