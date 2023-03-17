---
sidebar_position: 4
title: Join Mainnet
---

# Join Injective Mainnet

## Hardware Specification
Validators should expect to provision one or more data center locations with redundant power, networking, firewalls, HSMs and servers.

We initially recommend this minimum hardware specifications and they might rise as network usage increases.

```
4+ vCPU x64 2.0+ GHz
32+ GB RAM
1TB+ SSD
```

## Install injectived and peggo

```bash
wget https://github.com/InjectiveLabs/injective-chain-releases/releases/download/v1.9.0-1673640888/linux-amd64.zip
unzip linux-amd64.zip
sudo mv peggo /usr/bin
sudo mv injectived /usr/bin
sudo mv libwasmvm.x86_64.so /usr/lib 
```

## Initialize a new Injective Chain node

Before actually running the Injective Chain node, we need to initialize the chain, and most importantly its genesis file.

```
# The argument <moniker> is the custom username of your node, it should be human-readable.
export MONIKER=<moniker>
# the Injective Chain has a chain-id of "injective-1"
injectived init $MONIKER --chain-id injective-1
```

Running this command will create `injectived` default configuration files at `~/.injectived`.

## Prepare configuration to join Mainnet

You should now update the default configuration with the Mainnet's genesis file and application config file, as well as configure your persistent peers with a seed node.  

```bash
git clone https://github.com/InjectiveLabs/mainnet-config

# copy genesis file to config directory
cp mainnet-config/10001/genesis.json ~/.injectived/config/genesis.json

# copy config file to config directory
cp mainnet-config/10001/app.toml  ~/.injectived/config/app.toml
```

You can also run verify the checksum of the genesis checksum - 573b89727e42b41d43156cd6605c0c8ad4a1ce16d9aad1e1604b02864015d528
```bash
sha256sum ~/.injectived/config/genesis.json
```

Then open update the persistent_peers field present in ~/.injectived/config/config.toml with the contents of mainnet-config/10001/seeds.txt and update the `timeout_commit` to `300ms`.
```bash
cat mainnet-config/10001/seeds.txt
nano ~/.injectived/config/config.toml
```

## Configure systemd service for injectived

Edit the config at `/etc/systemd/system/injectived.service`:
```bash
[Unit]
  Description=injectived

[Service]
  WorkingDirectory=/usr/bin
  ExecStart=/bin/bash -c '/usr/bin/injectived --log-level=error start'
  Type=simple
  Restart=always
  RestartSec=5
  User=root

[Install]
  WantedBy=multi-user.target
```

Starting and restarting the systemd service
```bash
sudo systemctl daemon-reload
sudo systemctl restart injectived
sudo systemctl status injectived

# enable start on system boot
sudo systemctl enable injectived

# To check Logs
journalctl -u injectived -f
```

## Sync with the network

### Option 1. State-Sync

You can use state-sync to join the network by following the below instructions.

```bash
#!/bin/bash
sudo systemctl stop injectived
sudo injectived tendermint unsafe-reset-all --home ~/.injectived
CUR_HEIGHT=$(curl -sS https://tm.injective.network/block | jq .result.block.header.height | tr -d '"')
SNAPSHOT_INTERVAL=1000
RPC_SERVERS="23d0eea9bb42316ff5ea2f8b4cd8475ef3f35209\@65.109.36.70:11750,38c18461209694e1f667ff2c8636ba827cc01c86\@176.9.143.252:11750,4f9025feca44211eddc26cd983372114947b2e85\@176.9.140.49:11750,c98bb1b889ddb58b46e4ad3726c1382d37cd5609\@65.109.51.80:11750,f9ae40fb4a37b63bea573cc0509b4a63baa1a37a\@15.235.144.80:11750,7f3473ddab10322b63789acb4ac58647929111ba\@15.235.13.116:11750"
TRUST_HEIGHT=$(( CUR_HEIGHT - SNAPSHOT_INTERVAL ))
TRUSTED_HASH=$(curl -sS https://tm.injective.network/block?height=$TRUST_HEIGHT | jq .result.block_id.hash)
perl -i -pe 's|enable = false|enable = true|g' ~/.injectived/config/config.toml
perl -i -pe 's|rpc_servers = ".*?"|rpc_servers = "'$RPC_SERVERS'"|g' ~/.injectived/config/config.toml
perl -i -pe 's/^trust_height = \d+/trust_height = '$TRUST_HEIGHT'/' ~/.injectived/config/config.toml
perl -i -pe 's/^trust_hash = ".*?"/trust_hash = '$TRUSTED_HASH'/' ~/.injectived/config/config.toml
sudo systemctl start injectived
```

### Option 2. Snapshots

You can find pruned snapshots on:

1. [ChainLayer](https://quicksync.io/networks/injective.html).
2. [Polkachu](https://polkachu.com/tendermint_snapshots/injective).
3. [HighStakes](https://tools.highstakes.ch/files/injective.tar.gz).
4. [AutoStake](http://snapshots.autostake.net/injective-1/).

**Archival** (>7TB)
```bash
aws s3 sync --acl public-read --delete --no-sign-request s3://injective-snapshots/mainnet/injectived/daily/data $HOME/.injectived/data
aws s3 sync --acl public-read --delete --no-sign-request s3://injective-snapshots/testnet/injectived/daily/wasm $HOME/.injectived/wasm
```


### Support

For any further questions, you can always connect with the Injective Team via Discord, Telegram, and email.

[Discord](https://discord.gg/injective)

[Telegram](https://t.me/joininjective)

[E-mail](mailto: contact@injectivelabs.org)
