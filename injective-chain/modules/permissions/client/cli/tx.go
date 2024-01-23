package cli

import (
	"encoding/json"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

func GetTxCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, false)

	cmd.AddCommand(
		CreateNamespaceCmd(),
		DeleteNamespaceCmd(),
		UpdateNamespaceCmd(),
		UpdateNamespaceRolesCmd(),
		RevokeNamespaceRoleCmd(),
		ClaimVoucherCmd(),
	)

	return cmd
}

const (
	fWasmHook    = "wasm-hook"
	fMintsPaused = "mints-paused"
	fBurnsPaused = "burns-paused"
	fSendsPaused = "sends-paused"
	fPermissions = "permissions"
	fUpdateRoles = "update-roles"
	fRevokeRoles = "revoke-roles"
)

func CreateNamespaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-namespace <namespace.json>",
		Args:  cobra.ExactArgs(1),
		Short: "Create a denom namespace from a json file",
		Long: `Create a denom namespace from a json file.

		Example:
		$ %s tx permissions create-namespace namespace.json \

			Where namespace.json contains:
			{
				"denom": "inj",
				"wasm_hook": "inj1dzqd00lfd4y4qy2pxa0dsdwzfnmsu27hgttswz",
				"role_permissions": [
					{
						"role": "admin",
						"permissions": 7
					},
					{
						"role": "minter",
						"permissions": 1
					}
				],
				"address_roles": [
					{
						"address": "inj122qtfcjfx9suvgr5s7rtqgfy8xvtjhm8uc4x9f",
						"roles": ["whitelisted"]
					},
					{
						"address": "inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r",
						"roles": ["receiver"]
					}
				]
			}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			file, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			var ns types.Namespace
			if err := json.Unmarshal(file, &ns); err != nil {
				return err
			}

			msg := &types.MsgCreateNamespace{
				Sender:    clientCtx.GetFromAddress().String(),
				Namespace: ns,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func DeleteNamespaceCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"delete-namespace <denom>",
		"Delete the namespace associated with denom",
		&types.MsgDeleteNamespace{}, nil, nil,
	)

	cmd.Example = `injectived tx permissions delete-namespace inj`

	return cmd
}

func UpdateNamespaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-namespace <denom> [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Update existing namespace params with new values",
		Long:  `Update existing namespace params with new values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateNamespace{
				Sender:         clientCtx.GetFromAddress().String(),
				NamespaceDenom: args[0],
			}

			if newWasmValue := cmd.Flags().Lookup(fWasmHook); newWasmValue.Changed {
				msg.WasmHook = &types.MsgUpdateNamespace_MsgSetWasmHook{NewValue: newWasmValue.Value.String()}
			}

			if newSendsValue := cmd.Flags().Lookup(fSendsPaused); newSendsValue.Changed {
				boolValue, err := cmd.Flags().GetBool(fSendsPaused)
				if err != nil {
					return err
				}
				msg.SendsPaused = &types.MsgUpdateNamespace_MsgSetSendsPaused{NewValue: boolValue}
			}

			if newMintsValue := cmd.Flags().Lookup(fMintsPaused); newMintsValue.Changed {
				boolValue, err := cmd.Flags().GetBool(fMintsPaused)
				if err != nil {
					return err
				}
				msg.MintsPaused = &types.MsgUpdateNamespace_MsgSetMintsPaused{NewValue: boolValue}
			}

			if newBurnsValue := cmd.Flags().Lookup(fBurnsPaused); newBurnsValue.Changed {
				boolValue, err := cmd.Flags().GetBool(fBurnsPaused)
				if err != nil {
					return err
				}
				msg.BurnsPaused = &types.MsgUpdateNamespace_MsgSetBurnsPaused{NewValue: boolValue}
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(fWasmHook, "", "Wasm contract address")
	cmd.Flags().Bool(fSendsPaused, false, "Send tokens paused")
	cmd.Flags().Bool(fMintsPaused, false, "Mint tokens paused")
	cmd.Flags().Bool(fBurnsPaused, false, "Burn tokens paused")

	cmd.Example = `injectived tx permissions update-namespace inj
					--mints-paused false
					--burns-paused false
					--sends-paused true
					--wasm-hook inj1dzqd00lfd4y4qy2pxa0dsdwzfnmsu27hgttswz`

	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func UpdateNamespaceRolesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-namespace-roles <roles.json>",
		Args:  cobra.ExactArgs(1),
		Short: "Update the namespace roles and permissions",
		Long: `"Update the namespace roles and permissions by adding or updating roles and new addresses.

		Example:
		$ %s tx permissions update-namespace-roles roles.json \

			Where roles.json contains:
			{
				"namespace_denom": "inj",
				"role_permissions": [
					{
						"role": "admin",
						"permissions": 7
					},
					{
						"role": "minter",
						"permissions": 1
					}
				],
				"address_roles": [
					{
						"address": "inj122qtfcjfx9suvgr5s7rtqgfy8xvtjhm8uc4x9f",
						"roles": ["whitelisted"]
					},
					{
						"address": "inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r",
						"roles": ["receiver"]
					}
				]
			}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			file, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateNamespaceRoles{}
			if err := json.Unmarshal(file, &msg); err != nil {
				return err
			}

			msg.Sender = clientCtx.GetFromAddress().String()

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func RevokeNamespaceRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-namespace-roles <roles.json>",
		Args:  cobra.ExactArgs(1),
		Short: "Revoke address roles in a specific namespace",
		Long: `"Revoke address roles in a specific namespace.

		Example:
		$ %s tx permissions revoke-namespace-roles roles.json \

			Where roles.json contains:
			{
				"namespace_denom": "inj",
				"address_roles_to_revoke": [
					{
						"address": "inj122qtfcjfx9suvgr5s7rtqgfy8xvtjhm8uc4x9f",
						"roles": ["whitelisted"]
					},
					{
						"address": "inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r",
						"roles": ["receiver"]
					}
				]
			}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			file, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgRevokeNamespaceRoles{}
			if err := json.Unmarshal(file, &msg); err != nil {
				return err
			}

			msg.Sender = clientCtx.GetFromAddress().String()

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func ClaimVoucherCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"claim-voucher <originator_address>",
		"Claims the voucher from originator",
		&types.MsgClaimVoucher{}, nil, nil,
	)

	cmd.Example = `injectived tx permissions claim-voucher injEXCHANGEMODULEADDRESS`

	return cmd
}
