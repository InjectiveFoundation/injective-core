---
sidebar_position: 4
title: How to Launch Permissioned Assets
---

# How to Launch Permissioned Assets

Permissioned assets can be launched using [Injective APIs/SDKs](https://api.injective.exchange/#permissions) or the Injective CLI, `injectived`. See https://docs.injective.network/toolkits/injectived for more information on using the Injective CLI.

```bash
injectived tx permissions [command]
```

- There are four transaction commands available through the CLI:
    - `create-namespace`
        - Used to create a permissioned namespace from a json file for a `TokenFactory` denom
        - When creating a namespace, the address must be the admin of the same `TokenFactory` denom. Otherwise the namespace cannot be launched
    - `update-namespace`
        - Used to update the namespace parameters including:
            - Contract hook
            - Role permissions
            - Role managers
            - Policy statuses
            - Policy managers
        - Namespace updates are incremental, so unless a change is explicitly stated in the JSON, existing state will be untouched
    - `update-namespace-roles`
        - Used to assign roles to addresses and revoke roles from addresses
        - Like with namespace updates, role updates are also incremental
    - `claim-voucher`
        - Mainly used when a user is not authorized to receive a permissioned asset but is sent funds from an Injective module. The funds will be held in an Injective module address until the user receives the correct permissions to receive the asset

## `create-namespace`

```bash
injectived tx permissions create-namespace <namespace.json> [flags]
```

- The json file should have the following format (remove all comments before submitting):

```json
{ // Remove all comments before submitting! 
  "denom": "factory/inj1address/myTokenDenom",
  "contract_hook": "ContractHookAddress",
  "role_permissions": [ // CAUTION: makes sure to set role permissions for all namespace management roles!
    {
      "name": "EVERYONE",
      "role_id": 0,
      "permissions": 10 // SEND (8) + RECEIVE (2); excludes MINT, SUPER_BURN, and management actions
    },
    {
      "name": "admin",
      "role_id": 1,
      "permissions": 2013265920 // MODIFY_ROLE_PERMISSIONS, MODIFY_ROLE_MANAGERS, etc. (all namespace management actions)
    },
    {
      "name": "user",
      "role_id": 2,
      "permissions": 15 // MINT (1), RECEIVE (2), BURN (4), SEND (8)
    }
  ],
  "actor_roles": [
    {
      "actor": "inj1specificactoraddress",
      "roles": ["admin"]
    },
    {
      "actor": "inj1anotheractoraddress",
      "roles": ["user"]
    }
  ],
  "role_managers": [ // CAUTION: Make sure to set role managers for all namespace management roles!
    {
      "manager": "inj1manageraddress",
      "roles": ["admin"]
    }
  ],
  "policy_statuses": [ 
    {
      "action": 1, // Action_MINT
      "is_disabled": false,
      "is_sealed": false
    },
    {
      "action": 4, // Action_BURN
      "is_disabled": false,
      "is_sealed": false
    }
  ],
  "policy_manager_capabilities": [
    {
      "manager": "inj1policymanageraddress",
      "action": 268435456, // MODIFY_CONTRACT_HOOK
      "can_disable": true,
      "can_seal": false
    }
  ]
}

```

## `update-namespace`

```json
injectived tx permissions update-namespace <namespace-update.json> [flags]
```

- Namespace updates are incremental, so unless a change is explicitly stated in the JSON, existing state will be untouched

```json
{ // Remove all comments before submitting! 
  "denom": "factory/inj1address/myTokenDenom",
  "contract_hook": {
    "new_value": "newContractHookAddress"
  },
  "role_permissions": [
    {
      "name": "user",
      "role_id": 2,
      "permissions": 10 // RECEIVE (2) + SEND (8)
    },
    {
      "name": "EVERYONE",
      "role_id": 0,
      "permissions": 0 // Revoke all permissions
    }
  ],
  "role_managers": [
    {
      "manager": "inj1manageraddress",
      "roles": ["admin", "user"]
    }
  ],
  "policy_statuses": [
    {
      "action": 1, // MINT
      "is_disabled": true,
      "is_sealed": false
    },
    {
      "action": 4, // BURN
      "is_disabled": false,
      "is_sealed": true
    }
  ],
  "policy_manager_capabilities": [
    {
      "manager": "inj1policymanageraddress",
      "action": 536870912, // MODIFY_ROLE_PERMISSIONS
      "can_disable": true,
      "can_seal": false
    }
  ]
}

```

## `update-namespace-roles`

```json
 injectived tx permissions update-namespace-roles <roles.json> [flags]
```

```json
{
  "denom": "factory/inj1address/myTokenDenom",
  "role_actors_to_add": [
    {
      "role": "admin",
      "actors": [
        "inj1actoraddress1",
        "inj1actoraddress2"
      ]
    },
    {
      "role": "user",
      "actors": [
        "inj1actoraddress3"
      ]
    }
  ],
  "role_actors_to_revoke": [
    {
      "role": "user",
      "actors": [
        "inj1actoraddress4"
      ]
    },
    {
      "role": "admin",
      "actors": [
        "inj1actoraddress5"
      ]
    }
  ]
}
```

## `claim-voucher`

```bash
injectived tx permissions claim-voucher <denom>
```

- No JSON is needed for this command since the only parameter needed is the denom
