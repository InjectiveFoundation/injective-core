# Command examples

## Batch exchange modifications proposal

```sh
yes 12345678 | injectived tx exchange batch-exchange-modifications-proposal --proposal="./injective-chain/modules/exchange/client/cli/proposals/batchproposal.json" --deposit=10000000inj --from=genesis
```

## Batch community pool spend proposal

```sh
yes 12345678 | injectived tx exchange batch-community-pool-spend-proposal --proposal="./injective-chain/modules/exchange/client/cli/proposals/batchcommunitypoolspendproposal.json" --deposit=10000000inj --from=genesis
```
