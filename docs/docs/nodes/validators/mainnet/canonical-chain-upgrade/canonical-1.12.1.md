---
sidebar_position: 15
---
   
# Upgrade to v1.12.1
Thursday January 22 2024

This is a **non-consensus breaking release**, node operators can upgrade in their earliest convenience.

**Network downtime will not occur during this upgrade.**

All nodes, validator and non-consensus nodes must be upgraded as soon as possible to this version.

Validators should also upgrade the peggo version.

  - [Summary](#summary)
  - [Risks](#risks)
  - [Recovery](#recovery)
  - [Upgrade Procedure](#upgrade-procedure)
  - [Notes for Validator Operators](##notes-for-validator-operators)
  - [Notes for Service Providers](##notes-for-DEX-relayer-providers)

## Summary

The Injective Canonical Chain will undergo a improvement upgrade on **Jan 22th**.

The following is a short summary of the upgrade steps:

1. Backing up configs, data, and keys used for running the Injective Canonical Chain.
3. Install the [v1.12.1-1705909076](https://github.com/InjectiveLabs/injective-chain-releases/releases/tag/v1.12.1-1705909076)
4. Start your node with the new injectived binary to fulfill the upgrade.

Upgrade coordination and support for validators will be available on the `#validators` private channel of the [Injective Discord](https://discord.gg/injective).

The network upgrade can take the following potential pathways:
1. **Happy path**  
Validators successfully upgrade chain without purging the blockchain history and all validators are up within 5-10 minutes of the upgrade.

2. **Not-so-happy path**  
Validators have trouble upgrading to latest Canonical chain.

3. **Abort path**  
In the rare event that the team becomes aware of unnoticed critical issues, the Injective team will attempt to patch all the breaking states and provide another official binary within 36 hours.  
If the chain is not successfully resumed within 36 hours, the upgrade will be announced as aborted on the #mainnet-validators channel of [Discord](https://discord.gg/injective), and validators will need to resume running the chain without any updates or changes.

## Recovery

Prior to exporting chain state, validators are encouraged to take a full data snapshot at the export height before proceeding. Snapshotting depends heavily on infrastructure, but generally this can be done by backing up the `.injectived` directory.

It is critically important to backup the `.injectived/data/priv_validator_state.json` file after stopping your injectived process. This file is updated every block as your validator participates in a consensus rounds. It is a critical file needed to prevent double-signing, in case the upgrade fails and the previous chain needs to be restarted.

In the event that the upgrade does not succeed, validators and operators must restore the snapshot and downgrade back to [Injective Chain v1.12.0 release](https://github.com/InjectiveLabs/injective-chain-releases/releases/v1.12.0-1704530206) and continue the chain until next upgrade announcement.

### Upgrade Procedure

## Notes for Validators

You must remove the wasm cache before upgrading to the new version (rm -rf .injectived/wasm/wasm/cache/).

1. Verify you are currently running the correct version (`b92723b13`) of `injectived`:
   ```bash
      injectived version
      Version dev (b92723b13)
      Compiled at 20240106-0837 using Go go1.19.3 (amd64)
   ```

2. Make a backup of your `.injectived` directory
    ```bash
    cp ~/.injectived ./injectived-backup
    ```

   3. Download and install the injective-chain `v1.12.1 release`
   ```bash
   wget https://github.com/InjectiveLabs/injective-chain-releases/releases/download/v1.12.1-1705909076/linux-amd64.zip
   unzip linux-amd64.zip
   sudo mv injectived peggo /usr/bin
   sudo mv libwasmvm.x86_64.so /usr/lib
   ```

4. Verify you are currently running the correct version (`c1a64b7ed`) of `injectived` after downloading the v1.12.1 release:
    ```bash
   injectived version
   Version dev (c1a64b7ed)
   Compiled at 20240122-0743 using Go go1.19.3 (amd64)
   ```

5. Start injectived
    ```bash
   injectived start
   ```
6. Verify you are currently running the correct version (`e8089a7`) of `peggo` after downloading the v1.12.0 release:
   ```bash
    peggo version
    Version dev (e8089a7)
    Compiled at 20240122-0743 using Go go1.19.3 (amd64)
   ```
8. Start peggo
   ```bash
   peggo orchestrator
   ```   