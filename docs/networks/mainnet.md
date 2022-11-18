<!--
order: 6
title: Run Node and Join Mainnet
-->

# Join the Mainnet

## Hardware Specification
Validators should expect to provision one or more data center locations with redundant power, networking, firewalls, HSMs and servers.

We initially recommend this minimum hardware specifications and they might rise as network usage increases.

**Validator Node**
```
4+ vCPU x64 2.0+ GHz
32+ GB RAM
3TB+ SSD
```
**Sentry Node**
```
4+ vCPU x64 2.0+ GHz
32+ GB RAM
3TB+ SSD (no pruning, archival node)
```


## Sync Node From Scratch
Because of breaking changes we made with release 10001 rc7, the node should be synced with injectived version 10001 rc6, until halt height block 2045750. Then the injectived binary should be updated to version 10001 rc7 to the latest block.

Here is a simple 6-step guide on how to sync from scratch.

### Step 1: Backup your configuration
```bash
mkdir $HOME/injectived-backup/
cp -rf $HOME/.injectived/config $HOME/.injectived/keyring-file $HOME/injectived-backup/
```

### Step 2: 
Get the previous release version 10001 rc6
```bash
# Make sure that no running injectived processes
killall injectived &>/dev/null || true

# Remove current binary that you use 
rm -rf /usr/bin/injectived 

# Download the Injective Chain Mainnet-10001 rc6 binaries from the official injective-chain-releases.
wget https://github.com/InjectiveLabs/injective-chain-releases/releases/download/v1.0.1-1629427973/linux-amd64.zip

unzip linux-amd64.zip

mv -f injectived /usr/bin

#check version
injectived version 

# Should output
#Version dev (48356b9)
```

### Step 3: Initialize a new Injective Chain node

Before actually running the Injective Chain node, we need to initialize the chain, and most importantly its genesis file.

```bash
# The argument <moniker> is the custom username of your node, it should be human-readable.
export MONIKER=<moniker>
# the Injective Chain has a chain-id of "injective-1"
injectived init $MONIKER --chain-id injective-1
```

Running this command will create `injectived` default configuration files at `~/.injectived`.

### Step 4: Prepare configuration to join Mainnet

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

Then open update the persistent_peers field present in ~/.injectived/config/config.toml with the contents of mainnet-config/10001/seeds.txt and update the `timeout_commit` to `1000ms`.

```bash
cat mainnet-config/10001/seeds.txt
nano ~/.injectived/config/config.toml

# timeout_commit = 1000ms
```

### Step 5: Start your node using CLI or systemd service till halt height block
You can start your node simply by running `injectived start` which should start syncing the Injective Chain.

```bash
injectived start --halt-height 2045750
```

NOTE: The sync process takes approximately two days until it reaches halt height. If you want to run sync as a background process, follow the below guide using systemd service. 
Make sure you update ExecStart commannd bypassing --halt-height 2045750 param. Otherwise, you will overshoot the halt height and will have to resync from scratch again.

`ExecStart=/bin/bash -c '/usr/bin/injectived --halt-height 2045750 --log-level=error start'`

### Using systemd service
Configure systemd service for injectived if not configured already.

Edit the config at `/etc/systemd/system/injectived.service`:

```
[Unit]
  Description=injectived

[Service]
  WorkingDirectory=/usr/bin
  ExecStart=/bin/bash -c '/usr/bin/injectived --log-level=error start --halt-height 2045750'
  Type=simple
  Restart=always
  RestartSec=5
  User=ec2-user

[Install]
  WantedBy=multi-user.target
```

Starting and restarting systemd service

```bash
sudo systemctl daemon-reload
sudo systemctl restart injectived
sudo systemctl status injectived

# enable start on system boot
sudo systemctl enable injectived

# To check Logs
journalctl -u injectived -f
```

At this point, your node should start syncing blocks from the chain. Wait for the node to sync till above halt height is reached.

### Step 6
From this step, given that you've now reached the halt height, you can now switch binary to rc7 version.

```bash
# Make sure that no running injectived processes
killall injectived &>/dev/null || true

# If you use systemd service, stop it, otherwise skip this step
sudo systemctl stop injectived && sudo systemctl disable injectived

# Remove old binary, you don't need it anymore
rm -rf /usr/bin/injectived 

# Get the latest one
wget https://github.com/InjectiveLabs/injective-chain-releases/releases/download/v1.0.1-1635956190/linux-amd64.zip

unzip linux-amd64.zip

sudo mv injectived peggo injective-exchange /usr/bin
#check version
injectived version 
 
# Should output
#Version dev (b174465)
```

### Step 7
Continue syncing until chain halts automatically.

```bash
injectived start
```

NOTE: If you want to continue sync as a background process, follow above guide using systemd service. Make sure you update back ExecStart commannd by removing --halt-height 2045750 param.

`ExecStart=/bin/bash -c '/usr/bin/injectived --log-level=error start'`

Start systemd service

```bash
sudo systemctl start injectived && sudo systemctl enable injectived
```

Once your node completes syncing the blocks, proceed to the next step [Canonical Chain Upgrade](https://chain.injective.network/guides/mainnet/canonical-chain-upgrade.html).
