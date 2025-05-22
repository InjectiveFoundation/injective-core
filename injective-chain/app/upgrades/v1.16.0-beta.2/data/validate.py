import json
from collections import defaultdict


def main():
    mainnet_tokens = get_json_from_file("upgrade_mainnet_tokens.json")
    testnet_tokens = get_json_from_file("upgrade_testnet_tokens.json")

    mainnet_token_to_decimals = defaultdict(int)
    testnet_token_to_decimals = defaultdict(int)

    print("Validating mainnet tokens")
    for token in mainnet_tokens:
        decimals = token["decimals"]
        denom = token["denom"]
        if denom in mainnet_token_to_decimals and mainnet_token_to_decimals[denom] != decimals:
            print(f"Token {token['symbol']} {denom} has conflicting decimals {decimals} != {mainnet_token_to_decimals[denom]}")
        mainnet_token_to_decimals[denom] = token["decimals"]

    print("Validating testnet tokens")
    for token in testnet_tokens:
        decimals = token["decimals"]
        denom = token["denom"]
        if denom in testnet_token_to_decimals and testnet_token_to_decimals[denom] != decimals:
            print(f"Token {token['symbol']} {denom} has conflicting decimals {decimals} != {testnet_token_to_decimals[denom]}")
        testnet_token_to_decimals[denom] = token["decimals"]

    print(mainnet_token_to_decimals)
def get_json_from_file(filename):
    with open(filename, 'r') as f:
        return json.load(f)


if __name__ == '__main__':
    main()
