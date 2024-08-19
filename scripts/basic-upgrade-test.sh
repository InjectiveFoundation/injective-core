#!/bin/bash

PASSPHRASE="12345678"
TX_OPTS="--from=genesis  --chain-id=injective-1 --gas-prices 500000000inj --broadcast-mode=sync --yes"

# calculate halt height
CUR_HEIGHT=$(curl -sS localhost:26657/block | jq .result.block.header.height | tr -d '"')
HALT_HEIGHT=$(($CUR_HEIGHT + 5))

fetch_proposal_id() {
  current_proposal_id=$(curl 'http://localhost:10337/cosmos/gov/v1beta1/proposals?proposal_status=0&pagination.limit=1&pagination.reverse=true' | jq -r '.proposals[].proposal_id')
  proposal=$((current_proposal_id))
}

vote() {
        PROPOSAL_ID=$1
        echo $PROPOSAL_ID
        yes $PASSPHRASE | injectived tx gov vote $PROPOSAL_ID yes $TX_OPTS
}

yes $PASSPHRASE | injectived tx upgrade software-upgrade v1.13.1 \
 --title "Injective Protocol 1.12 Dry Run" \
 --upgrade-height $HALT_HEIGHT \
 --summary "hi" \
 --deposit 500000000000000000000inj $TX_OPTS \
 --no-validate

sleep 3

fetch_proposal_id
vote $proposal
