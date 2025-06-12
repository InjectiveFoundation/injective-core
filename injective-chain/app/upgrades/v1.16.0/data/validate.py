import json
from collections import defaultdict


def main():
    testnet_tokens = get_json_from_file("upgrade_testnet_tokens.json")
    testnet_token_to_decimals = defaultdict(int)

    print("Validating testnet tokens")
    for token in testnet_tokens:
        decimals = token["decimals"]
        denom = token["denom"]
        if denom in testnet_token_to_decimals and testnet_token_to_decimals[denom] != decimals:
            print(f"Token {token['symbol']} {denom} has conflicting decimals {decimals} != {testnet_token_to_decimals[denom]}")
        testnet_token_to_decimals[denom] = token["decimals"]

    print(testnet_token_to_decimals)

def get_json_from_file(filename):
    with open(filename, 'r') as f:
        return json.load(f)

if __name__ == '__main__':
    main()
