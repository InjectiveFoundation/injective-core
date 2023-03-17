# Client

## CLI

A user can query and interact with the `crisis` module using the CLI.

### Transactions

The `tx` commands allow users to interact with the `crisis` module.

```bash
injectived tx crisis --help
```

#### invariant-broken

The `invariant-broken` command submits proof when an invariant was broken to halt the chain

```bash
injectived tx crisis invariant-broken [module-name] [invariant-route] [flags]
```

Example:

```bash
injectived tx crisis invariant-broken bank total-supply --from=[keyname or address]
```
