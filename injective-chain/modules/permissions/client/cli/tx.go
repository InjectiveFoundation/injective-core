package cli

import (
	"encoding/json"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
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
	return &cobra.Command{
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
			  "role_permissions": {
				"admin": 7,
				"minter": 1,
			  },
			  "address_roles": {
				"inj1f2kdg34689x93cvw2y59z7y46dvz2fk8g3cggx": {
				  "roles": ["admin"]
				}
			  }
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
	cmd := cli.TxCmd(
		"update-namespace <denom> [flags]",
		"Update existing namespace with new values",
		&types.MsgUpdateNamespace{},
		cli.FlagsMapping{
			"WasmHook":    cli.Flag{Flag: fWasmHook},
			"MintsPaused": cli.Flag{Flag: fMintsPaused},
			"SendsPaused": cli.Flag{Flag: fSendsPaused},
			"BurnsPaused": cli.Flag{Flag: fBurnsPaused},
		},
		nil,
	)

	cmd.Flags().String(fWasmHook, "", "Wasm contract address")
	cmd.Flags().Bool(fSendsPaused, false, "Send tokens paused")
	cmd.Flags().Bool(fMintsPaused, false, "Mint tokens paused")
	cmd.Flags().Bool(fBurnsPaused, false, "Burn tokens paused")

	cmd.Example = `injectived tx permissions update-namespace inj
					--mints-paused false
					--burns-paused false
					--sends-paused true`

	return cmd
}

func UpdateNamespaceRolesCmd() *cobra.Command {
	cmd := cli.TxCmd("update-namespace-roles <denom> [flags]",
		"Update the namespace roles and permissions",
		&types.MsgUpdateNamespaceRoles{},
		cli.FlagsMapping{
			"RolePermissions": cli.Flag{
				Flag: fPermissions,
				Transform: func(origV string, _ grpc1.ClientConn) (any, error) {
					file, err := os.ReadFile(origV)
					if err != nil {
						return nil, err
					}

					var perms map[string]int32
					if err := json.Unmarshal(file, &perms); err != nil {
						return nil, err
					}

					return perms, nil
				},
			},

			"AddressRoles": cli.Flag{
				Flag: fUpdateRoles,
				Transform: func(origV string, _ grpc1.ClientConn) (any, error) {
					file, err := os.ReadFile(origV)
					if err != nil {
						return nil, err
					}

					var roles map[string]*types.Roles
					if err := json.Unmarshal(file, &roles); err != nil {
						return nil, err
					}

					return roles, nil
				},
			},
		},
		nil,
	)

	cmd.Flags().String(fPermissions, "", "JSON file defining role permissions")
	cmd.Flags().String(fUpdateRoles, "", "JSON file defining address' roles")

	cmd.Example = `injectived tx permissions update-namespace-roles inj --permissions perms.json`

	return cmd
}

func RevokeNamespaceRoleCmd() *cobra.Command {
	cmd := cli.TxCmd("revoke-namespace-roles <denom> [flag]",
		"Revoke address roles in a specific namespace",
		&types.MsgRevokeNamespaceRoles{},
		cli.FlagsMapping{
			"AddressRolesToRevoke": cli.Flag{
				Flag: fRevokeRoles,
				Transform: func(origV string, _ grpc1.ClientConn) (any, error) {
					file, err := os.ReadFile(origV)
					if err != nil {
						return nil, err
					}

					var revoked map[string]*types.Roles
					if err := json.Unmarshal(file, &revoked); err != nil {
						return nil, err
					}

					return revoked, nil
				},
			},
		},
		nil,
	)

	cmd.Flags().String(fRevokeRoles, "", "JSON file indicating roles revoked")

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
