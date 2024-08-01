---
sidebar_position: 7
---
   
# Upgrade to 10004-rc1-patch
Sunday Feb 20th

Upgrade Injective from 10004-rc1 to 10004-rc1 [patch version 10004-rc1-1645352045](https://github.com/InjectiveLabs/injective-chain-releases/releases/tag/v1.4.0-1645352045) created due to an incident that happened on Sunday, Feb 20 at 3:55 AM UTC-05:00, when [Network halted at block 7941974](https://explorer.injective.network/block/7941974).

This indicates that the upgrade procedure should be performed on block number **7941974**

 - [Summary](#summary)
  - [Risks](#risks)
  - [Recovery](#recovery)
  - [Upgrade Procedure](#upgrade-procedure)
  - [Notes for Service Providers](#notes-for-DEX-relayer-providers)

## Summary

- At 3:55 AM UTC-05:00 Network halted, [block height 7941974](https://explorer.injective.network/block/7941974)
- At 5:05 AM UTC-05:00 [New release created with rolled back fix, Version 10004-rc1-1645352045](https://github.com/InjectiveLabs/injective-chain-releases/releases/tag/v1.4.0-1645352045)
- At 06:08 UTC-05:00 [Network is back up]( https://explorer.injective.network/block/7941975)


The following is a short summary of the upgrade steps:

1. Backing up configs, data, and keys used for running the Injective Canonical Chain.
2. Install the [Mainnet-10004-rc1-v1.4.0-1642928125](https://github.com/InjectiveLabs/injective-chain-releases/releases/tag/v1.4.0-1645352045)
3. Start your node with the new injectived binary to fulfill the upgrade.

Upgrade coordination and support for validators will be available on the `#mainnet-validators` private channel of the [Injective Discord](https://discord.gg/injective).

The network upgrade can take the following potential pathways:
1. Happy path  
Validators successfully migrate from the 10004-rc1 Injective Chain to 10004-rc1 patch without purging the blockchain history and all validators are up within 1-2 hours.

2. Not-so-happy path  
Validators have trouble upgrading the chain. This could be some consensus-breaking changes not covered in the upgrade handler, or compatibility issue of the migrated state with new injectived binary, but validators can at least export the genesis.

3. Abort path  
In the rare event that the team becomes aware of unnoticed critical issues, the Injective team will attempt to patch all the breaking states and provide another official binary within 36 hours.  

## Recovery

Prior to exporting Injective state, validators are encouraged to take a full data snapshot at the
export height before proceeding. Snapshotting depends heavily on infrastructure, but generally, this
can be done by backing up the `.injectived` directory. We would suggest using `aws s3 sync` with the delete flag, this will shorten the snapshotting time as only the file diffs are uploaded to the S3 bucket.

It is critically important to back up the `.injectived/data/priv_validator_state.json` file after stopping your injectived process. This file is updated every block as your validator participates in consensus rounds. It is a critical file needed to prevent double-signing, in case the upgrade fails and the previous chain needs to be restarted.

## Upgrade Procedure

1. Verify you are currently running the correct (pre-upgrade) version (`94583db`) of `injectived`:
   ```bash
   $ injectived version
   Version dev (94583db)
   Compiled at 20220123-0855 using Go go1.17.6 (amd64)
   ```

2. After the chain has halted, make a backup of your `.injectived` directory
    ```bash
    cp ~/.injectived ./injectived-backup
    ```
   **NOTE**: It is recommended for validators and operators to take a full data snapshot at the export
   height before proceeding in case the upgrade does not go as planned or if not enough voting power
   comes online in a sufficient and agreed upon amount of time. In such a case, the chain will fallback
   to continue operating the existing Canonical Chain. See [Recovery](#recovery) for details on how to proceed.

3. Download and install Injective 10004-rc1 release
  ```bash
  wget https://github.com/InjectiveLabs/injective-chain-releases/releases/download/v1.4.0-1645352045/linux-amd64.zip
  unzip linux-amd64.zip
  sudo mv injectived /usr/bin
  ```

4. Verify you are currently running the correct new version (`30b3dbf`) of `injectived` after downloading the 10004-rc1 release:
    ```bash
   $ injectived version
   Version dev (30b3dbf)
   Compiled at 20220220-1014 using Go go1.17.6 (amd64)
   ```

5. Coordinate to restart your injectived with other validators
   ```bash
   injectived start
   ```
   The binary will perform the upgrade automatically and continue the next consensus round if everything goes well.

## Notes for DEX relayer providers
Relayer upgrade will be available after the chain is successfully upgraded as it relies on several other components that work with injectived.