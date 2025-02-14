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
		UpdateNamespaceCmd(),
		UpdateNamespaceRolesCmd(),
		ClaimVoucherCmd(),
	)

	return cmd
}

func CreateNamespaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-namespace <namespace.json>",
		Args:  cobra.ExactArgs(1),
		Short: "Create a namespace from a json file",
		Long: `Create a namespace from a json file.

		Example:
		$ %s tx permissions create-namespace namespace.json \

			Where namespace.json contains:
			{
				"denom": "inj",
				"contract_hook": "inj1dzqd00lfd4y4qy2pxa0dsdwzfnmsu27hgttswz",
				"role_permissions": [
					{
						"role": "EVERYONE",
						"role_id": 0,
						"permissions": 14,
					},
					{
						"role": "minter",
						"role_id": 1,
						"permissions": 15
					}
				],
				"actor_roles": [
					{
						"actor": "inj122qtfcjfx9suvgr5s7rtqgfy8xvtjhm8uc4x9f",
						"roles": ["minter"]
					},
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

func UpdateNamespaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-namespace <namespace-update.json> [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Update existing namespace params with new values",
		Long:  `Update existing namespace params with new values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			file, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			var msg types.MsgUpdateNamespace
			if err := json.Unmarshal(file, &msg); err != nil {
				return err
			}

			msg.Sender = clientCtx.GetFromAddress().String()

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Example = `injectived tx permissions update-namespace namespace-update.json`

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
				"denom": "inj",
				"role_actors_to_add": [
					{
						"role": "whitelisted"
						"actors": ["inj122qtfcjfx9suvgr5s7rtqgfy8xvtjhm8uc4x9f"],
					},
				],
				"role_actors_to_revoke": [
					{
						"role": "whitelisted"
						"actors": ["inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r"],
					},
				],

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

			msg := &types.MsgUpdateActorRoles{}
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
