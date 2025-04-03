# Injective Block SDK Integration

Injective integrates the [Block SDK](https://github.com/InjectiveLabs/block-sdk). Our solution leverages a multi‑lane mempool that separates transactions into four distinct lanes:

- **Oracle Lane** – for transactions that contain oracle messages.
- **Governance Lane** – for any transaction sent by an admin of the exchange module.
- **Exchange Lane** – for transactions that contain exchange messages.
- **Default Lane** – for all other transactions.

## Key Features

### 1. Multi‑Lane Mempool

- **Priority Ordering:**  
  The lanes are ordered by priority: the Oracle Lane has the highest priority, followed by the Governance Lane, then the Exchange Lane, and finally the Default Lane. This ordering is critical when building and verifying block proposals.

- **Dedicated Lane Logic:**  
  Each lane uses its own match handler. We have the following lanes in order of priority:
  1. The **Oracle Lane** checks that the transaction contains an oracle message.
  2. The **Governance Lane** checks that the first signer is an admin of the exchange module.
  3. The **Exchange Lane** verifies that at least one exchange message is present and orders transactions using custom fee/priority logic.
  4. The **Default Lane** accepts any transaction not matching the other lanes.

### 2. Routing Transactions Based on Priority

A key enhancement in our integration is the custom logic for routing transactions based on priority.

- **If a transaction from the same signer already exists in a lane with lower priority:**  
  The match handler returns `false`, meaning the new transaction is not captured by the current (higher‑priority) lane. Instead, it falls through to be processed by the lower‑priority lane.

- **Otherwise:**  
  The new transaction is accepted in the current lane.

This mechanism prevents account sequence mismatches. **As a user, you need to be aware that prior lower‑priority transactions may be processed first, even if you submit a new transaction with higher priority. Further, they might cause delays in processing your new transaction.**

### 3. Exchange Lane Priority

The Exchange Lane uses custom priority logic to order transactions based on fee discounts, account tiers, and special handling for liquidation messages. Key points include:

- **Custom Tx Priority Calculation:**  
  The Exchange Lane extracts the transaction’s priority by considering the account’s fee discount tier and, if applicable, whether the transaction contains only liquidation messages. Liquidation messages are assigned highest priority.

- **Account Tier and Fee Discount:**  
  For regular exchange transactions, the lane computes the highest account tier (from the transaction’s signer data) as a measure of priority. Higher tiers result in higher priority.

### 4. Oracle and Governance Lane Max Gas Limit

Transactions that exceed the max gas limit of the Oracle and Governance Lanes 
are rejected in the `matching` stage (when a tx is inserted in the mempool). If 
we didn't do this in the matching stage, a big transaction could make it into the 
lane, and be rejected later in the `prepare` stage (in PrepareProposal). The 
transaction would remain in the mempool indefinitely, thereby blocking the sender 
account from submitting any other transactions.