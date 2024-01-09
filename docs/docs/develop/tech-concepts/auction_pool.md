# Contributing to the Burn Auction Pool

Every week, 60% of the accumulated trading fees transform into a pool of assets that's up for auction.

Members of the Injective community can participate in these auctions. Here's how it happens:

- Members place their bids on the asset basket using INJ.
- Once the auction concludes, the highest bid wins the assets.
- This winning bid is not re-circulated. Instead, it's burned, ensuring the value of INJ is maintained.

Beyond the fees, there's another way the auction pool can grow: direct contributions from community members. If you wish to boost the auction pool, you can directly send funds to the Auction subaccount.

:::info

If you're ready to contribute, send funds to this **subaccount**:

```0x1111111111111111111111111111111111111111111111111111111111111111```

Be aware that any funds you send will be reflected in the next auction, not the current one.

:::

Contributions should be directed to the subaccount. For this purpose, you'll employ the [MsgExternalTransfer](../modules/Injective/exchange/messages#msgexternaltransfer).

For a more practical view:

- Dive into the [Injective Python SDK example](https://github.com/InjectiveLabs/sdk-python/blob/master/examples/chain_client/30_ExternalTransfer.py)) to get a comprehensive understanding.

- Alternatively, refer to the following simplified code snippet for a streamlined approach.

``` python MsgExternalTransfer example
import asyncio
import logging

from pyinjective.composer import Composer as ProtoMsgComposer
from pyinjective.async_client import AsyncClient
from pyinjective.transaction import Transaction
from pyinjective.constant import Network
from pyinjective.wallet import PrivateKey

async def main() -> None:
    # select network: local, testnet, mainnet
    network = Network.mainnet()
    composer = ProtoMsgComposer(network=network.string())
    
    # initialize grpc client
    # client = AsyncClient(network, insecure=False)
    client = AsyncClient(network, insecure=True)

    await client.sync_timeout_height()

    # load account
    priv_key = PrivateKey.from_hex("Your PK")
    pub_key = priv_key.to_public_key()
    
    address = pub_key.to_address()
    account = await client.get_account(address.to_acc_bech32())
    print(f" pubkey {address.to_acc_bech32()}")
    subaccount_id = address.get_subaccount_id(index=1)
    dest_subaccount_id = "0x1111111111111111111111111111111111111111111111111111111111111111"

    # prepare tx msg
    msg = composer.MsgExternalTransfer(
        sender=address.to_acc_bech32(),
        source_subaccount_id=subaccount_id,
        destination_subaccount_id=dest_subaccount_id,
        amount=0.123123123,
        denom="INJ"
    )
   
    tx = (
        Transaction()
        .with_messages(msg)
        .with_sequence(client.get_sequence())
        .with_account_num(client.get_number())
        .with_chain_id(network.chain_id)
    )

    # build tx
    gas_price = 500000000
    gas_limit = 90000 + 20000  # add 20k for gas, fee computation
    gas_fee = '{:.18f}'.format((gas_price * gas_limit) / pow(10, 18)).rstrip('0')
    fee = [composer.Coin(
        amount=gas_price * gas_limit,
        denom=network.fee_denom,
    )]
    tx = tx.with_gas(gas_limit).with_fee(fee).with_memo('').with_timeout_height(client.timeout_height)
    sign_doc = tx.get_sign_doc(pub_key)
    sig = priv_key.sign(sign_doc.SerializeToString())
    tx_raw_bytes = tx.get_tx_data(sig, pub_key)

    # broadcast tx: send_tx_async_mode, send_tx_sync_mode, send_tx_block_mode
    res = await client.send_tx_sync_mode(tx_raw_bytes)
    print(res)
    print("gas wanted: {}".format(gas_limit))
    print("gas fee: {} INJ".format(gas_fee))

if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(main())
```
