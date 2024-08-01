---
sidebar_position: 3
---

# Configure Peggo

## Equinox Testnet
### Step 1: Configure your Peggo relayer

```bash
mkdir ~/.peggo
cp testnet-config/staking/40014/peggo-config.env ~/.peggo/.env
cd ~/.peggo
```

First, update the `PEGGO_ETH_RPC` in the `.env` file with a valid Sepolia EVM RPC Endpoint.

To set up your own Sepolia full node, follow the instructions [here](https://ethereum.org/en/developers/docs/nodes-and-clients/run-a-node/). It's possible to use an Alchemy or Infura RPC, but keep in mind that the Peggo bridge is still under development, and the request amount it makes to the RPC is not optimized. Ensure it does not incur high costs on your account.

Peggo also requires access to your validator's Cosmos and Ethereum credentials to sign transactions for the corresponding networks.

#### Cosmos Keys

There are two ways to provide the credential access - a keyring with encrypted keys, or just private key in plaintext.

**1. Cosmos Keyring**

Update the `PEGGO_COSMOS_FROM` to your validator key name (or account address) and `PEGGO_COSMOS_FROM_PASSPHRASE` to your Cosmos Keyring passphrase. Please note that the default keyring backend is `file` and it will try to locate keys on disk.

Keyring path must be pointing to homedir of your injectived node, if you want reuse the keys from there.

Learn more about Cosmos Keyring setup [here](https://docs.cosmos.network/master/run-node/keyring.html).

**2. Cosmos Private Key (Unsafe)**  

Simply update the `PEGGO_COSMOS_PK` with your Validator's Account private key.

To obtain your validator's Cosmos private key, run `injectived keys unsafe-export-eth-key $VALIDATOR_KEY_NAME`.

This method is insecure and is not recommended.

#### Ethereum Keys

There are two ways to provide the credential access - a Geth keystore with encrypted keys, or just private key in plaintext.

**1. Geth Keystore**

Simply create a new private key store and update the following env variables:
* `PEGGO_ETH_KEYSTORE_DIR`
* `PEGGO_ETH_FROM`
* `PEGGO_ETH_PASSPHRASE`

You can find instructions for securely creating a new Ethereum account using a keystore in the Geth Documentation [here](https://geth.ethereum.org/docs/interface/managing-your-accounts).  

Example is provided below.

```bash
geth account new --datadir=/home/ec2-user/.peggo/data/

INFO [03-23|18:18:36.407] Maximum peer count                       ETH=50 LES=0 total=50
Your new account is locked with a password. Please give a password. Do not forget this password.
Password:
Repeat password:

Your new key was generated

Public address of the key:   0x9782dc957DaE6aDc394294954B27e2118D05176C
Path of the secret key file: /home/ec2-user/.peggo/data/keystore/UTC--2021-03-23T15-18-44.284118000Z--9782dc957dae6adc394294954b27e2118d05176c

- You can share your public address with anyone. Others need it to interact with you.
- You must NEVER share the secret key with anyone! The key controls access to your funds!
- You must BACKUP your key file! Without the key, it's impossible to access account funds!
- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!
```

Now you can set env variables like this:

```ini
PEGGO_ETH_KEYSTORE_DIR=/home/ec2-user/.peggo/data/keystore
PEGGO_ETH_FROM=0x9782dc957DaE6aDc394294954B27e2118D05176C
PEGGO_ETH_PASSPHRASE=12345678
```

Next, ensure that your Ethereum addresss has Sepolia ETH. You can request Sepolia ETH from the public faucet [here](https://www.alchemy.com/faucets/ethereum-sepolia).  

**2. Ethereum Private Key (Unsafe)**

Simply update the `PEGGO_ETH_PK` with a new Ethereum Private Key from a new account.

Next, ensure that your Ethereum addresss has Sepolia ETH. You can request Sepolia ETH from the public faucet [here](https://www.alchemy.com/faucets/ethereum-sepolia).  

### Step 2: Register Your Ethereum Address

```bash
cd ~/.peggo
peggo tx register-eth-key
```

You can verify successful registration by checking for your Validator's mapped Ethereum address on https://staking-lcd-testnet.injective.network/peggy/v1/valset/current.

### Step 3: Start the Relayer

```bash
peggo orchestrator
```

This starts the Peggo bridge (relayer / orchestrator).

### Step 4: Create a Peggo systemd service

Add `peggo.service` file with below content under `/etc/systemd/system/peggo.service`

```ini
[Unit]
  Description=peggo

[Service]
  WorkingDirectory=/home/ec2-user/.peggo
  ExecStart=/bin/bash -c 'peggo orchestrator '
  Type=simple
  Restart=always
  RestartSec=1
  User=ec2-user

[Install]
  WantedBy=multi-user.target
```

Then run the following commands to configure Environment variables, start and stop the peggo relayer.

```bash
sudo systemctl start peggo
sudo systemctl stop peggo
sudo systemctl restart peggo
sudo systemctl status peggo

# enable start on system boot
sudo systemctl enable peggo

# To check Logs
journalctl -f -u peggo
```

### Step 5: (Optional) Protect Cosmos Keyring from unauthorized access

:::important
This is an advanced DevOps topic, consult with your sysadmin.
:::

Learn more about Cosmos Keyring setup [here](https://docs.cosmos.network/master/run-node/keyring.html). Once you've launched your node, the default keyring will have the validator operator key stored on disk in the encrypted form. Usually the keyring is located within node's homedir, i.e. `~/.injectived/keyring-file`.

Some sections of the Injective Staking documentation will guide you through using this key for governance purposes, i.e. submitting transactions and setting up an Ethereum bridge. In order to protect the keys from unauthorized access, even when the keyring passphrase is leaked via configs, you can set OS permissions to allow disk access to `injectived` / `peggo` processes only.

In Linux systems like Debian, Ubuntu and RHEL, this can be achieved using POSIX Access Control Lists (ACLs). Before beginning to work with ACLs, the file system must be mounted with ACLs turned on. There are some official guides for each distro:

* [Ubuntu](https://help.ubuntu.com/community/FilePermissionsACLs)
* [Debian](https://wiki.debian.org/Permissions)
* [Amazon Linux (RHEL)](https://www.redhat.com/sysadmin/linux-access-control-lists)

## Testnet


### Step 1: Configure your Peggo relayer

```bash
mkdir ~/.peggo
cp testnet-config/40014/peggo-config.env ~/.peggo/.env
cd ~/.peggo
```

First, update the `PEGGO_ETH_RPC` in the `.env` file with a valid Ethereum EVM RPC Endpoint.

To create your own Ethereum full node, you can follow our instructions [here](https://ethereum.org/en/developers/docs/nodes-and-clients/run-a-node/). It's possible to use an external Ethereum RPC provider such as Alchemy or Infura, but keep in mind that the Peggo bridge relayer uses heavy use of `eth_getLogs` calls which may increase your cost burden depending on your provider.

Peggo also requires access to your validator's delegated Injective Chain account and Ethereum key credentials to sign transactions for the corresponding networks.

#### Creating your delegated Cosmos Key for sending Injective transactions

Your peggo relayer can either
 - Use an explicitly delegated account key specific for sending validator specific Peggy transactions (i.e. `ValsetConfirm`, `BatchConfirm`, and `SendToCosmos` transactions)
 or
  - Simply use your validator's account key.

For isolation purposes, we recommend creating a delegated Cosmos key to send Injective transactions instead of using your validator account key.

To create a new key, run
```bash
injectived keys add $ORCHESTRATOR_KEY_NAME
```

Then ensure that your orchestrator inj address has INJ balance.

To obtain your orchestrators's inj address, run
```bash
injectived keys list $ORCHESTRATOR_KEY_NAME
```

You can transfer INJ from your validator account to orchestrator address using this command
```bash
injectived tx bank send $VALIDATOR_KEY_NAME  $ORCHESTRATOR_INJ_ADDRESS <amount-in-inj> --chain-id=injective-888 --keyring-backend=file --yes --node=tcp://localhost:26657 --gas-prices=500000000inj
```

Example
```bash
injectived tx bank send genesis inj1u3eyz8nkvym0p42h79aqgf37gckf7szreacy9e 20000000000000000000inj --chain-id=injective-888  --keyring-backend=file --yes --node=tcp://localhost:26657 --gas-prices=500000000inj
```

You can then verify that your orchestrator account has INJ balances by running
```bash
injectived q bank balances $ORCHESTRATOR_INJ_ADDRESS
```

#### Managing Cosmos account keys for `peggo`

Peggo supports two options to provide Cosmos signing key credentials - using the Cosmos keyring (recommended) or by providing a plaintext private key.

**Option 1. Cosmos Keyring**

In the `.env` file, first specify the `PEGGO_COSMOS_FROM` and `PEGGO_COSMOS_FROM_PASSPHRASE` corresponding to your peggo account signing key.

If you are using a delegated account key configuration as recommended above, this will be your `$ORCHESTRATOR_KEY_NAME` and passphrase respectively. Otherwise, this should be your `$VALIDATOR_KEY_NAME` and associated validator passphrase.

Please note that the default keyring backend is `file` and that as such peggo will try to locate keys on disk by default.

To use the default injectived key configuration, you should set the keyring path to the home directory of your injectived node, e.g. `~/.injectived`.

You can also read more about the Cosmos Keyring setup [here](https://docs.cosmos.network/master/run-node/keyring.html).

**Option 2. Cosmos Private Key (Unsafe)**  

In the `.env` file, specify the `PEGGO_COSMOS_PK` corresponding to your peggo account signing key.

If you are using a delegated account key configuration as recommended above, this will be your orchestrator account's private key. Otherwise, this should be your validator's account private key.

To obtain your orchestrator's Cosmos private key (if applicable), run
```bash
injectived keys unsafe-export-eth-key $ORCHESTRATOR_KEY_NAME
```

To obtain your validator's Cosmos private key (if applicable), run
```bash
injectived keys unsafe-export-eth-key $VALIDATOR_KEY_NAME
````

Again, this method is less secure and is not recommended.

#### Managing Ethereum keys for `peggo`

Peggo supports two options to provide signing key credentials - using the Geth keystore (recommended) or by providing a plaintext Ethereum private key.

**Option 1. Geth Keystore**

Simply create a new private key store and update the following env variables:
* `PEGGO_ETH_KEYSTORE_DIR`
* `PEGGO_ETH_FROM`
* `PEGGO_ETH_PASSPHRASE`

You can find instructions for securely creating a new Ethereum account using a keystore in the Geth Documentation [here](https://geth.ethereum.org/docs/interface/managing-your-accounts).  

For convience, an example is provided below.

```bash
geth account new --datadir=/home/ec2-user/.peggo/data/

INFO [03-23|18:18:36.407] Maximum peer count                       ETH=50 LES=0 total=50
Your new account is locked with a password. Please give a password. Do not forget this password.
Password:
Repeat password:

Your new key was generated

Public address of the key:   0x9782dc957DaE6aDc394294954B27e2118D05176C
Path of the secret key file: /home/ec2-user/.peggo/data/keystore/UTC--2021-03-23T15-18-44.284118000Z--9782dc957dae6adc394294954b27e2118d05176c

- You can share your public address with anyone. Others need it to interact with you.
- You must NEVER share the secret key with anyone! The key controls access to your funds!
- You must BACKUP your key file! Without the key, it's impossible to access account funds!
- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!
```

Make sure you heed the warnings that geth provides, particularly in backing up your key file so that you don't lose your keys by mistake. We also recommend not using any quote or backtick characters in your passphrase for peggo compatibility purposes.

You should now set the following env variables:

```bash
# example values, replace with your own
PEGGO_ETH_KEYSTORE_DIR=/home/ec2-user/.peggo/data/keystore
PEGGO_ETH_FROM=0x9782dc957DaE6aDc394294954B27e2118D05176C
PEGGO_ETH_PASSPHRASE=12345678
```

Then ensure that your Ethereum address has enough ETH.   

**Option 2. Ethereum Private Key (Unsafe)**

Simply update the `PEGGO_ETH_PK` with a new Ethereum Private Key from a new account.

Then ensure that your Ethereum address has ETH.
### Step 2: Register Your Orchestrator and Ethereum Address

You can register orchestrator and ethereum address only once. It **CANNOT** be updated later.
So Check twice before running below command.
```bash
injectived tx peggy set-orchestrator-address $VALIDATOR_INJ_ADDRESS $ORCHESTRATOR_INJ_ADDRESS $ETHEREUM_ADDRESS --from $VALIDATOR_KEY_NAME --chain-id=injective-888 --keyring-backend=file --yes --node=tcp://localhost:26657 --gas-prices=500000000inj

```
- To obtain your validator's inj address, run, `injectived keys list $VALIDATOR_KEY_NAME`
- To obtain your orchestrators's inj address, `injectived keys list $ORCHESTRATOR_KEY_NAME`

Example:
```bash
injectived tx peggy set-orchestrator-address inj10m247khat0esnl0x66vu9mhlanfftnvww67j9n inj1x7kvxlz2epqx3hpq6v8j8w859t29pgca4z92l2 0xf79D16a79130a07e77eE36e8067AeA783aBdA3b6 --from validator-key-name --chain-id=injective-888 --keyring-backend=file --yes --node=tcp://localhost:26657 --gas-prices=500000000inj
```


You can verify successful registration by checking for your Validator's mapped Ethereum address on https://testnet.lcd.injective.dev/peggy/v1/valset/current.

### Step 3: Start the Relayer

```bash
cd ~/.peggo
peggo orchestrator
```

This starts the Peggo bridge (relayer / orchestrator).

### Step 4: Create a Peggo systemd service

Add `peggo.service` file with below content under `/etc/systemd/system/peggo.service`

```ini
[Unit]
  Description=peggo

[Service]
  WorkingDirectory=/home/ec2-user/.peggo
  ExecStart=/bin/bash -c 'peggo orchestrator '
  Type=simple
  Restart=always
  RestartSec=1
  User=ec2-user

[Install]
  WantedBy=multi-user.target
```

Then run the following commands to configure Environment variables, start and stop the peggo relayer.

```bash
sudo systemctl start peggo
sudo systemctl stop peggo
sudo systemctl restart peggo
sudo systemctl status peggo

# enable start on system boot
sudo systemctl enable peggo

# To check Logs
journalctl -f -u peggo
```

### Step 5: (Optional) Protect Cosmos Keyring from unauthorized access

:::important
This is an advanced DevOps topic, consult with your sysadmin.
:::

Learn more about Cosmos Keyring setup [here](https://docs.cosmos.network/master/run-node/keyring.html). Once you've launched your node, the default keyring will have the validator operator key stored on disk in the encrypted form. Usually the keyring is located within node's homedir, i.e. `~/.injectived/keyring-file`.

Some sections of the Injective Staking documentation will guide you through using this key for governance purposes, i.e. submitting transactions and setting up an Ethereum bridge. In order to protect the keys from unauthorized access, even when the keyring passphrase is leaked via configs, you can set OS permissions to allow disk access to `injectived` / `peggo` processes only.

In Linux systems like Debian, Ubuntu and RHEL, this can be achieved using POSIX Access Control Lists (ACLs). Before beginning to work with ACLs, the file system must be mounted with ACLs turned on. There are some official guides for each distro:

* [Ubuntu](https://help.ubuntu.com/community/FilePermissionsACLs)
* [Debian](https://wiki.debian.org/Permissions)
* [Amazon Linux (RHEL)](https://www.redhat.com/sysadmin/linux-access-control-lists)

## Contribute

If you'd like to inspect the Peggo orchestrator source code and contribute, you can do so at https://github.com/InjectiveLabs/peggo.
