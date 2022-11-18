<!--
order: 1
title: Creating Transactions
-->

# Creating Transactions

In this document we are going to explain the concepts of creating Transactions on Injective (on a higher level), signing them using a privateKey (wallet) and broadcasting them to the node. {synopsis}

## Pre-requisite Readings

- [Basic Transaction Concepts](./../concepts/04_transactions_and_messages.md) {prereq}

# Transaction Flow

Every transaction on Injective follows the same flow. The flow consists of three steps: preparing, signing and broadcasting the transaction. Let's dive into each step separately and explain the process in-depth (including examples) so we can understand the whole transaction flow.

## Preparing a transaction

First of, we need to prepare the transaction for signing. There are two different ways to do this - depending on the type of the wallet you want to use for signing. 

### Ethereum native wallets

To use Ethereum native wallets, we have to convert the transaction to EIP712 typed data and use the wallet to sign this typed data. There are two approaches we can take to achieve this:

1. Using the `web3-gateway` API which accepts a Message (and some other details) and returns EIP712 TypedData that can be signed,

So, what is the `web3-gateway`? The `web3-gateway` is a microservice which is part of the `indexer-api` and allows developers to introduce fee delegation service for their end-users (meaning that the end-users will not pay gas when they submit transactions, this gas will be covered by the owner of the `web3-gateway` microservice) and converting the transaction into a EIP712 TypedData that can be signed using Ethereum native wallets. [Full Example](#ethereum-native-wallet---using-the-web3-gateway)

2. Using our custom abstraction for the Messages which allows the developer to get EIP712 TypedData straight from the proto file of the particular message (experimental at this point but the default and the proper way).

With this approach we have to prepare everything about the transaction on the client side, including the Transaction context (fees, accountDetails, etc). [Full Example](#ethereum-native-wallet---using-the-client-side-approach)

### Cosmos native wallets

At this point you **can't** use some online abstractions that provide a quick way to prepare the transaction for you based on the provided Message and the signer (ex. using the `@cosmjs/stargate` package). The reason why is that these packages don't support Injective's publicKey typeUrl, so we have to do the preparation of the address on the client side. 

Worry not, we have provided functions that will prepare the `txRaw` transaction for you in our sdks. `txRaw` is the transaction interface used in Cosmos that contains details about the transaction and the signer itself. [Full Example](#cosmos-native-wallets---ex-keplr)

## Signing a transaction

Once you prepared the transaction, we proceed to signing. 

### Ethereum native wallets 

Once you get the EIP712 TypedData use any Ethereum native wallet to sign. 

### Cosmos native wallets

Once you get the `txRaw` transaction use any Cosmos native wallet to sign (ex: Keplr),

You can also use our `@injectivelabs/wallet-ts` package to get out-of-the-box wallet provides that will give you abstracted methods which you can use to sign transaction. Refer to the [documentation](../tools/injectivets/02_wallet-ts.md) of the package, its really simple to setup and use. 

## Broadcasting a transaction

Once we have the signature ready, we need to broadcast the transaction to the Injective chain itself. Same as preparing the transaction, depending on the wallet (and the approach we want to take) broadcasting the transaction takes a different form. 

### Ethereum native wallets

1. Using the `web3-gateway` API which accepts the response from the `prepareTx` and the signature and broadcasts the transaction to the node,

2. Using the client side approach where we make a native cosmos transaction, include the signature we got from the wallet provider and broadcast it to the chain 

### Cosmos native wallets 

As we already should have prepared the `txRaw` before in the preparing of the transaction step, we just include the signature in the `txRaw` and broadcast it to the chain. 

## Code Examples 

### Ethereum native wallet - using the `web3-gateway`

```js
import { 
  MsgSend,
  IndexerGrpcTransactionApi, 
  getAddressFromInjectiveAddress
} from '@injectivelabs/sdk-ts'
import { EthereumChainId } from '@injectivelabs/ts-types'

/** Prepare the Message */
const injectiveAddress = ''
const ethereumChainId = EthereumChainId.Mainnet
const ethereumAddress = getAddressFromInjectiveAddress(injectiveAddress)
const memo = '' 
const amount = {
  amount: new BigNumberInBase(0.01).toWei().toFixed(),
  denom: "inj",
};

const msg = MsgSend.fromJSON({
  amount,
  srcInjectiveAddress: injectiveAddress,
  dstInjectiveAddress: injectiveAddress,
});

/** Preparing the transaction */
const prepareTxResponse = await transactionApi.prepareTxRequest({
  memo: memo,
  message: msg.toWeb3(),
  address: tx.address,
  chainId: ethereumAddress,
  estimateGas: false,
})
const dataToSign = txResponse.getData()

/** Use your preferred approach to sign EIP712 TypedData, example with Metamask */
const signature = await window.ethereum.request({
  method: 'eth_signTypedData_v4',
  params: [ethereumAddress, dataToSign],
})

/** Broadcasting the transaction */
const response = await transactionApi.broadcastTxRequest({
  signature,
  txResponse: prepareTxResponse,
  message: msg.toWeb3(),
  chainId: ethereumChainId,
})
```

### Ethereum native wallet - using the client-side approach

```js
import { 
  MsgSend,
  getEip712Tx,
  ChainRestAuthApi,
  ChainRestTendermintApi,
  BaseAccount,
  DEFAULT_STD_FEE,
  hexToBase64,
  hexToBuff,
  DEFAULT_TIMEOUT_HEIGHT,
  getAddressFromInjectiveAddress
} from '@injectivelabs/sdk-ts'
import {
  createTransaction,
  createTxRawEIP712,
  createWeb3Extension,
  SIGN_AMINO,
  TxGrpcClient,
} from '@injectivelabs/tx-ts'
import { EthereumChainId } from '@injectivelabs/ts-types'

/** Prepare the Message */
const injectiveAddress = ''
const chainId = '' /* example injective-1 */
const sentryLcdEndpoint = '' /* example: https://lcd.injective.network */
const sentryGrpcApi = '' /* example: https://grpc.injective.network */
const ethereumChainId = EthereumChainId.Mainnet
const ethereumAddress = getAddressFromInjectiveAddress(injectiveAddress)
const memo = '' 
const amount = {
  amount: new BigNumberInBase(0.01).toWei().toFixed(),
  denom: "inj",
};

/** Preparing the transaction */
const msg = MsgSend.fromJSON({
  amount,
  srcInjectiveAddress: injectiveAddress,
  dstInjectiveAddress: injectiveAddress,
});

/** Account Details **/
const chainRestAuthApi = new ChainRestAuthApi(
  sentryLcdEndpoint,
)
const accountDetailsResponse = await chainRestAuthApi.fetchAccount(
  injectiveAddress,
)
const baseAccount = BaseAccount.fromRestApi(accountDetailsResponse)
const accountDetails = baseAccount.toAccountDetails()

/** Block Details */
const chainRestTendermintApi = new ChainRestTendermintApi(
  sentryLcdEndpoint,
)
const latestBlock = await chainRestTendermintApi.fetchLatestBlock()
const latestHeight = latestBlock.header.height
const timeoutHeight = new BigNumberInBase(latestHeight).plus(
  DEFAULT_TIMEOUT_HEIGHT,
)

/** EIP712 for signing on Ethereum wallets */
const eip712TypedData = getEip712Tx({
  msgs: [msg],
  tx: {
    accountNumber: accountDetails.accountNumber.toString(),
    sequence: accountDetails.sequence.toString(),
    timeoutHeight: timeoutHeight.toFixed(),
    chainId: chainId,
  },
  ethereumChainId: ethereumChainId,
})

/** Use your preferred approach to sign EIP712 TypedData, example with Metamask */
const signature = await window.ethereum.request({
  method: 'eth_signTypedData_v4',
  params: [ethereumAddress, JSON.stringify(eip712TypedData)],
})

/** Get Public Key of the signer */
const publicKeyHex = recoverTypedSignaturePubKey(
  eip712TypedData,
  signature,
)
const publicKeyBase64 = hexToBase64(publicKeyHex)

/** Broadcasting the transaction */
const txGrpcClient = new TxGrpcClient(sentryGrpcApi)
const { txRaw } = createTransaction({
  message: msgs.map((m) => m.toDirectSign()),
  memo: memo,
  signMode: SIGN_AMINO,
  fee: DEFAULT_STD_FEE,
  pubKey: publicKeyBase64,
  sequence: baseAccount.sequence,
  timeoutHeight: timeoutHeight.toNumber(),
  accountNumber: baseAccount.accountNumber,
  chainId: chainId,
})
const web3Extension = createWeb3Extension({
  ethereumChainId,
})
const txRawEip712 = createTxRawEIP712(txRaw, web3Extension)

/** Append Signatures */
txRawEip712.setSignaturesList([signatureBuff])

/** Broadcast the transaction */
const response = await txGrpcClient.broadcast(txRawEip712)

if (response.code !== 0) {
  throw new Error(`Transaction failed: ${response.rawLog}`)
}

return response.txhash
```

### Cosmos native wallets - ex: Keplr

```js
import { 
  MsgSend,
  ChainRestAuthApi,
  ChainRestTendermintApi,
  BaseAccount,
  DEFAULT_STD_FEE,
  DEFAULT_TIMEOUT_HEIGHT,
} from '@injectivelabs/sdk-ts'
import {
  createTransaction,
  TxGrpcClient,
} from '@injectivelabs/tx-ts'
import { KeplrWallet } from '@injectivelabs/wallet-ts/dist/keplr'
import { DEFAULT_STD_FEE } from '@injectivelabs/utils'

const injectiveAddress = ''
const chainId = '' /* example injective-1 */
const sentryLcdEndpoint = '' /* example: https://lcd.injective.network */
const sentryGrpcApi = '' /* example: https://grpc.injective.network */
const memo = '' 
const amount = {
  amount: new BigNumberInBase(0.01).toWei().toFixed(),
  denom: "inj",
};

/** Account Details **/
const chainRestAuthApi = new ChainRestAuthApi(
  sentryLcdEndpoint,
)
const accountDetailsResponse = await chainRestAuthApi.fetchAccount(
  injectiveAddress,
)
const baseAccount = BaseAccount.fromRestApi(accountDetailsResponse)
const accountDetails = baseAccount.toAccountDetails()

/** Block Details */
const chainRestTendermintApi = new ChainRestTendermintApi(
  sentryLcdEndpoint,
)
const latestBlock = await chainRestTendermintApi.fetchLatestBlock()
const latestHeight = latestBlock.header.height
const timeoutHeight = new BigNumberInBase(latestHeight).plus(
  DEFAULT_TIMEOUT_HEIGHT,
)

/** Preparing the transaction */
const msg = MsgSend.fromJSON({
  amount,
  srcInjectiveAddress: injectiveAddress,
  dstInjectiveAddress: injectiveAddress,
});
const keplrWallet = new KeplrWallet(chainId)
const endpoints = await keplrWallet.getChainEndpoints()
const key = await keplrWallet.getKey()
const signer = await keplrWallet.getOfflineSigner()

/** Prepare the Transaction **/
const { txRaw, signDoc } = createTransaction({
  message: msgs.map((m) => m.toDirectSign()),
  memo: memo,
  fee: DEFAULT_STD_FEE,
  pubKey: Buffer.from(key.pubKey).toString('base64'),
  sequence: baseAccount.sequence,
  timeoutHeight: timeoutHeight.toNumber(),
  accountNumber: baseAccount.accountNumber,
  chainId: chainId,
})

const signature = await signer.signDirect(address, signDoc)

/** Append Signatures */
txRaw.setSignaturesList([Buffer.from(response.signature.signature, 'base64')]);

/** Broadcast the transaction */
const response = await txGrpcClient.broadcast(txRawEip712)

if (response.code !== 0) {
  throw new Error(`Transaction failed: ${response.rawLog}`)
}

return response.txhash
```
