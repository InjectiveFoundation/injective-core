<!--
order: 5
title: Run Node and Join Testnet
-->

# Join the Equinox Testnet
Validators should expect to provision one or more data center locations with redundant power, networking, firewalls, HSMs and servers. We expect that a modest level of hardware specifications will be needed initially and that they might rise as network use increases. Participating in the testnet is the best way to learn more.

For the Equinox testnet, we recommend following specs:

**Validator Node**
```
6 GB RAM
100GB SSD
x64 2.0 GHz
4 vCPU
```
**Sentry Node**
```
6 GB RAM
100GB SSD
x64 2.0 GHz
4 vCPU
```

### Step 1: Install injectived and peggo binaries
Make sure you have installed `injectived` and `peggo`. You can follow the guide [here](../tools/injectived/01_install.md).

### Step 2: Initialize a new Injective Chain node
Before actually running the Injective Chain node, we need to initialize the chain, and most importantly its genesis file.

```bash
# The argument <moniker> is the custom username of your node, it should be human-readable.
export MONIKER=<moniker>
# the Injective Chain has a chain-id of "injective-888"
injectived init $MONIKER
```

### Step 3: Prepare configuration to join the Equinox Testnet
You should now update the default configuration with the testnet's genesis file and application config file, as well as configure your persistent peers with a seed node.

```bash
git clone https://github.com/InjectiveLabs/network-config/

# copy genesis file to config directory
cp network-config/staking/40011/genesis.json ~/.injectived/config/genesis.json

# copy config file to config directory
cp network-config/staking/40011/app.toml  ~/.injectived/config/app.toml
```

You can also run verify the checksum of the genesis checksum - 4e8b598fd6943d898f452e0a431f90d98c5355d39337393ae384e454416e335e
```bash
sha256sum ~/.injectived/config/genesis.json
```

Then open update the persistent_peers field present in ~/.injectived/config/config.toml with the contents of network-config/staking/40011/seeds.txt and update the timeout_commit to 1000ms.
```bash
cat network-config/staking/40011/seeds.txt
nano ~/.injectived/config/config.toml

# timeout_commit = 1000ms
```

### Step 4: Start your node and sync the Injective Chain

```bash
injectived start
```

At this point, your node should start syncing blocks from the chain.

### Step 5: Use systemd service
Configure systemd service for injectived if not configured already.
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

At this point, your node should start syncing blocks from the chain.
