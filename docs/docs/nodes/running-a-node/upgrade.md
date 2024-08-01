---
sidebar_position: 5
title: Node Upgrades and Maintenance
---

# Upgrading and Maintaining Your Node (For Non-Validators)

## Chain Upgrades

Injective periodically undergoes software upgrades. When a chain upgrade governance proposal is passed, a block height will be specified at which all nodes will automatically panic and stop running. At this point, the upgraded `injectived` binaries  can be installed, and the node can be restarted.

See [here](https://github.com/InjectiveLabs/injective-chain-releases/releases) for the most recent and prior chain releases.

## Node Upgrade Directions

To summarize, follow these steps to upgrade your node:
1. Sync your node to the block height predetermined by the upgrade governance proposal.
2. The node will automatically panic/stop at the predetermined upgrade height.
3. Remove the old binaries and install the new release binaries.
4. Restart the node.

## Node Maintenance (Managing Storage)

As Injective state grows, your disk space may fill up. It’s recommended you periodically prune the chain data by downloading new snapshots. Beyond the overhead on the disk, the node is more performant when the chain state is smaller.

Injective validators take daily light snapshots that you can use to clean the chain state which grows at about 10-15GB daily. These snapshots are normally only around 2-3GB. We recommend pruning the chain data every 300-400GB. For links to snapshots as well as directions for applying the snapshot/syncing the node, see [Join Mainnet](./mainnet).