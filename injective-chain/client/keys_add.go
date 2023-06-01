package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cryptohd "github.com/InjectiveLabs/injective-core/injective-chain/crypto/hd"
)

const (
	flagInteractive       = "interactive"
	flagRecover           = "recover"
	flagNoBackup          = "no-backup"
	flagCoinType          = "coin-type"
	flagAccount           = "account"
	flagIndex             = "index"
	flagMultisig          = "multisig"
	flagNoSort            = "nosort"
	flagHDPath            = "hd-path"
	flagMultiSigThreshold = "multisig-threshold"

	// DefaultKeyPass contains the default key password for genesis transactions
	DefaultKeyPass      = "12345678"
	mnemonicEntropySize = 256
)

/*
input
  - bip39 mnemonic
  - bip39 passphrase
  - bip44 path
  - local encryption password

output
  - armor encrypted private key (saved to file)
*/
func RunAddCmd(ctx client.Context, cmd *cobra.Command, args []string, inBuf *bufio.Reader) error {
	var err error

	name := args[0]
	interactive, _ := cmd.Flags().GetBool(flagInteractive)
	noBackup, _ := cmd.Flags().GetBool(flagNoBackup)
	showMnemonic := !noBackup
	kb := ctx.Keyring
	outputFormat := ctx.OutputFormat

	keyringAlgos, _ := kb.SupportedAlgorithms()
	algoStr, _ := cmd.Flags().GetString(flags.FlagKeyAlgorithm)
	algo, err := keyring.NewSigningAlgoFromString(algoStr, keyringAlgos)
	if err != nil {
		return err
	}

	if dryRun, _ := cmd.Flags().GetBool(flags.FlagDryRun); dryRun {
		// use in memory keybase
		kb = keyring.NewInMemory(ctx.Codec, cryptohd.EthSecp256k1Option())
	} else {
		_, err = kb.Key(name)
		if err == nil {
			// account exists, ask for user confirmation
			response, err2 := input.GetConfirmation(fmt.Sprintf("override the existing name %s", name), inBuf, cmd.ErrOrStderr())
			if err2 != nil {
				return err2
			}

			if !response {
				return errors.New("aborted")
			}

			err2 = kb.Delete(name)
			if err2 != nil {
				return err2
			}
		}

		multisigKeys, _ := cmd.Flags().GetStringSlice(flagMultisig)
		if len(multisigKeys) != 0 {
			pks := make([]cryptotypes.PubKey, len(multisigKeys))
			multisigThreshold, _ := cmd.Flags().GetInt(flagMultiSigThreshold)
			if err := validateMultisigThreshold(multisigThreshold, len(multisigKeys)); err != nil {
				return err
			}

			for i, keyname := range multisigKeys {
				k, err := kb.Key(keyname)
				if err != nil {
					return err
				}

				key, err := k.GetPubKey()
				if err != nil {
					return err
				}
				pks[i] = key
			}

			if noSort, _ := cmd.Flags().GetBool(flagNoSort); !noSort {
				sort.SliceStable(pks, func(i, j int) bool {
					return bytes.Compare(pks[i].Address(), pks[j].Address()) < 0
				})
			}

			pk := multisig.NewLegacyAminoPubKey(multisigThreshold, pks)
			info, err := kb.SaveMultisig(name, pk)
			if err != nil {
				return err
			}

			return printCreate(cmd, info, false, "", outputFormat)
		}
	}

	pubKey, _ := cmd.Flags().GetString(keys.FlagPublicKey)
	if pubKey != "" {
		var pk cryptotypes.PubKey
		err = ctx.Codec.UnmarshalInterfaceJSON([]byte(pubKey), &pk)
		if err != nil {
			return err
		}

		info, err := kb.SaveOfflineKey(name, pk)
		if err != nil {
			return err
		}

		return printCreate(cmd, info, false, "", outputFormat)
	}

	coinType, _ := cmd.Flags().GetUint32(flagCoinType)
	account, _ := cmd.Flags().GetUint32(flagAccount)
	index, _ := cmd.Flags().GetUint32(flagIndex)
	hdPath, _ := cmd.Flags().GetString(flagHDPath)
	useLedger, _ := cmd.Flags().GetBool(flags.FlagUseLedger)

	if hdPath == "" {
		hdPath = hd.CreateHDPath(coinType, account, index).String()
	} else if useLedger {
		return errors.New("cannot set custom bip32 path with ledger")
	}

	// If we're using ledger, only thing we need is the path and the bech32 prefix.
	if useLedger {
		bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()

		info, err := kb.SaveLedgerKey(name, algo, bech32PrefixAccAddr, coinType, account, index)
		if err != nil {
			return err
		}

		return printCreate(cmd, info, false, "", outputFormat)
	}

	// Get bip39 mnemonic
	var mnemonic, bip39Passphrase string

	recoverFlag, _ := cmd.Flags().GetBool(flagRecover)
	if recoverFlag {
		mnemonic, err = input.GetString("Enter your bip39 mnemonic", inBuf)
		if err != nil {
			return err
		}

		if !bip39.IsMnemonicValid(mnemonic) {
			return errors.New("invalid mnemonic")
		}
	} else if interactive {
		mnemonic, err = input.GetString("Enter your bip39 mnemonic, or hit enter to generate one.", inBuf)
		if err != nil {
			return err
		}

		if !bip39.IsMnemonicValid(mnemonic) && mnemonic != "" {
			return errors.New("invalid mnemonic")
		}
	}

	if mnemonic == "" {
		// read entropy seed straight from tmcrypto.Rand and convert to mnemonic
		entropySeed, err := bip39.NewEntropy(mnemonicEntropySize)
		if err != nil {
			return err
		}

		mnemonic, err = bip39.NewMnemonic(entropySeed)
		if err != nil {
			return err
		}
	}

	// override bip39 passphrase
	if interactive {
		bip39Passphrase, err = input.GetString(
			"Enter your bip39 passphrase. This is combined with the mnemonic to derive the seed. "+
				"Most users should just hit enter to use the default, \"\"", inBuf)
		if err != nil {
			return err
		}

		// if they use one, make them re-enter it
		if bip39Passphrase != "" {
			p2, err := input.GetString("Repeat the passphrase:", inBuf)
			if err != nil {
				return err
			}

			if bip39Passphrase != p2 {
				return errors.New("passphrases don't match")
			}
		}
	}

	info, err := kb.NewAccount(name, mnemonic, bip39Passphrase, hdPath, algo)
	if err != nil {
		return err
	}

	// Recover key from seed passphrase
	if recoverFlag {
		// Hide mnemonic from output
		showMnemonic = false
		mnemonic = ""
	}

	return printCreate(cmd, info, showMnemonic, mnemonic, outputFormat)
}

func printCreate(cmd *cobra.Command, k *keyring.Record, showMnemonic bool, mnemonic, outputFormat string) error {
	switch outputFormat {
	case keys.OutputFormatText:
		cmd.PrintErrln()
		printKeyInfo(cmd.OutOrStdout(), k, keyring.MkAccKeyOutput, outputFormat)

		// print mnemonic unless requested not to.
		if showMnemonic {
			fmt.Fprintln(cmd.ErrOrStderr(), "\n**Important** write this mnemonic phrase in a safe place.")
			fmt.Fprintln(cmd.ErrOrStderr(), "It is the only way to recover your account if you ever forget your password.")
			fmt.Fprintln(cmd.ErrOrStderr(), "")
			fmt.Fprintln(cmd.ErrOrStderr(), mnemonic)
		}
	case keys.OutputFormatJSON:
		out, err := keyring.MkAccKeyOutput(k)
		if err != nil {
			return err
		}

		if showMnemonic {
			out.Mnemonic = mnemonic
		}

		jsonString, err := json.Marshal(out)
		if err != nil {
			return err
		}

		cmd.Println(string(jsonString))

	default:
		return fmt.Errorf("invalid output format %s", outputFormat)
	}

	return nil
}

func validateMultisigThreshold(k, nKeys int) error {
	if k <= 0 {
		return fmt.Errorf("threshold must be a positive integer")
	}
	if nKeys < k {
		return fmt.Errorf(
			"threshold k of n multisignature: %d < %d", nKeys, k)
	}
	return nil
}

type bechKeyOutFn func(keyInfo *keyring.Record) (keyring.KeyOutput, error)

func printTextInfos(w io.Writer, kos []keyring.KeyOutput) {
	out, err := yaml.Marshal(&kos)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(w, string(out))
}

func printKeyInfo(w io.Writer, keyInfo *keyring.Record, bechKeyOut bechKeyOutFn, output string) {
	ko, err := bechKeyOut(keyInfo)
	if err != nil {
		panic(err)
	}

	switch output {
	case keys.OutputFormatText:
		printTextInfos(w, []keyring.KeyOutput{ko})

	case keys.OutputFormatJSON:
		out, err := json.Marshal(ko)
		if err != nil {
			panic(err)
		}

		fmt.Fprintln(w, string(out))
	}
}
