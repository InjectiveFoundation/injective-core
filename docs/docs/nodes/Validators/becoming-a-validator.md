---
sidebar_position: 1
label:  Becoming a Validator
---

# Becoming a Validator

### Hardware Requirements

| *Minimum* | *Recommendation* | 
| :---: | :---: |
| RAM Memory  64GB  | RAM Memory 128GB |
| CPU 8 cores  | CPU 12 cores  |
| Storage 1TB  | Storage 2TB  |
| Network 1Gbps+  | Network 5Gbps+  |

### Step 1: Create a Validator Account

First, run the keygen command with your desired validator key name.   

```bash
export VALIDATOR_KEY_NAME=[my-validator-key]
injectived keys add $VALIDATOR_KEY_NAME
```

This will derive a new private key and encrypt it to disk. Make sure to remember the password you used.

```bash
# EXAMPLE OUTPUT
- name: myvalidatorkey
  type: local
  address: inj1queq795wx8gzqc8706uz80whp07mcgg5nmpj6h
  pubkey: injpub1r0mckeepqwzmrzt5af00hgc7fhve05rr0q3q6wvx4xn6k46zguzykdszg6cnu0zca4q
  mnemonic: ""
  threshold: 0
  pubkeys: []


**Important** write this mnemonic phrase in a safe place.
It is the only way to recover your account if you ever forget your password.
```

**⚠️ important ⚠️  
The output will contain a mnemonic phrase that represents your key in plain text. Make sure to save this phrase as a backup of your key, since without a key you will not be able to control your validator. The phrase is better be backed up on physical paper, storing it in cloud storage may compromise your validator later.**

Remember the address starting from `inj`, this is going to be your Injective Validator Account address.

### Step 2: Obtain Mainnet INJ

To proceed with the next step, you will need to obtain some real INJ on Mainnet Ethereum (ERC-20 token address [`0xe28b3b32b6c345a34ff64674606124dd5aceca30`](https://etherscan.io/token/0xe28b3b32b6c345a34ff64674606124dd5aceca30)).

### Step 3: "Transfer" INJ to your validator account on Injective

Deposit your Mainnet INJ tokens into your validator's account on Injective by using the staking dashboard. You will have to [connect your wallet](https://medium.com/injective-labs/injective-hub-guide-9a14f09f6a7d)on our [Hub](https://hub.injective.network/bridge) and then deposit INJ from Ethereum Mainnet network. This will trigger an automated bridge that maps tokens from Ethereum network to Injective.

![validator-transfer](./bridge-transfer.png)

After a few minutes, you should be able to verify that your deposit was successful on the UI. Alternatively, you can query your account balance using the `injectived` CLI with the following command:

```bash
injectived q bank balances <my-validator-inj-address>
```

### Step 4: Create your validator account

Obtain your node's tendermint validator Bech32 encoded PubKey consensus address.

```bash
VALIDATOR_PUBKEY=$(injectived tendermint show-validator)
echo $VALIDATOR_PUBKEY

# Example: {"@type": "/cosmos.crypto.ed25519.PubKey", "key": "GWEJv/KSFhUUcKBWuf9TTT3Ful+3xV/1lFhchyW1TZ8="}
```

Then create your new validator initialized with a self-delegation with your INJ tokens. Most critically, you will need to decide on the values of your validator's staking parameters.

* `--moniker` - Your validator's name
* `--amount` -  Your validator's initial amount of INJ to bond
* `--commission-max-change-rate` - Your validator's maximum commission change rate percentage (per day)
* `--commission-max-rate` - Your validator's maximum commission rate percentage
* `--commission-rate` - Your validator's initial commission rate percentage
* `--min-self-delegation` - Your validator's minimum required self delegation

Once you decide on your desired values, set them as follows.
```bash
MONIKER=<my-moniker>
AMOUNT=100000000000000000000inj # to delegate 100 INJ, as INJ is represented with 18 decimals.  
COMMISSION_MAX_CHANGE_RATE=0.1 # e.g. for a 10% maximum change rate percentage per day
COMMISSION_MAX_RATE=0.1 # e.g. for a 10% maximum commission rate percentage
COMMISSION_RATE=0.1 # e.g. for a 10% initial commission rate percentage
MIN_SELF_DELEGATION_AMOUNT=50000000000000000000 # e.g. for a minimum 50 INJ self delegation required on the validator
```

Then run the following command to create your validator.

```bash
injectived tx staking create-validator \
--moniker=$MONIKER \
--amount=$AMOUNT \
--gas-prices=500000000inj \
--pubkey=$VALIDATOR_PUBKEY \
--from=$VALIDATOR_KEY_NAME \
--keyring-backend=file \
--yes \
--node=tcp://localhost:26657 \
--chain-id=injective-1
--commission-max-change-rate=$COMMISSION_MAX_CHANGE_RATE \
--commission-max-rate=$COMMISSION_MAX_RATE \
--commission-rate=$COMMISSION_RATE \
--min-self-delegation=$MIN_SELF_DELEGATION_AMOUNT
```

Extra `create-validator` options to consider:

```
--identity=        		The optional identity signature (ex. UPort or Keybase)
--pubkey=          		The Bech32 encoded PubKey of the validator
--security-contact=		The validator's (optional) security contact email
--website=         		The validator's (optional) website
```

You can check that your validator was successfully created by checking the [staking dashboard](https://staking.injective.network/validators) or by entering the following CLI command.

```bash
injectived q staking validators
```

If you see your validator in the list of validators, then congratulations, you've officially joined as an Injective Mainnet validator! 🎉


### Step 5: (Optional) Delegate Additional INJ to your Validator

To gain a deeper empirical understanding of user experience that your future delegators will experience, you can try delegation through [Staking Guide](https://medium.com/injective-labs/injective-hub-guide-9a14f09f6a7d).

These steps will allow you to experience the delegation flow using MetaMask Transactions. 🦊

Alternatively, you can always use the Injective CLI to send a delegation transaction.  

```bash
injectived tx staking delegate [validator-addr] [amount] --from $VALIDATOR_KEY_NAME --keyring-backend=file --yes --node=tcp://localhost:26657
```

### Step 6: (Recommended) Connecting Your Validator Identity with Keybase

By adding your Keybase pubkey to your validator identity information in Injective, you can automatically pull in your Keybase public profile information in client applications like the Injective Hub and Explorer. Here's how to connect your validator identity with your Keybase pubkey:

1. Create a validator profile on Keybase at [https://keybase.io/](https://keybase.io/) and make sure it's complete.
2. Add your validator identity pubkey to Injective:
    - Send a `MsgEditValidator` to update your `Identity` validator identity with your Keybase pubkey. You can also use this message to change your website, contact email, and other details.

That's it! Once you've connected your validator identity with Keybase, the Injective Explorer and Hub can automatically pull in your brand identity, and other public profile information.

### Next Steps

Next, proceed to setup your Ethereum Bridge Relayer. This is a necessary step in order to prevent your validator from being slashed. You should do this immediately after setting up your validator.
