package client

// DONTCOVER

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"

	tmconfig "github.com/tendermint/tendermint/config"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	mintypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/hd"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/cmd/injectived/config"
	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
)

var (
	flagNodeDirPrefix  = "node-dir-prefix"
	flagNumValidators  = "v"
	flagOutputDir      = "output-dir"
	flagNodeDaemonHome = "node-daemon-home"
	flagCoinDenom      = "coin-denom"
	flagIPAddrs        = "ip-addresses"
)

const nodeDirPerm = 0o755

// TestnetCmd initializes all files for tendermint testnet and application
func TestnetCmd(
	mbm module.BasicManager, genBalancesIterator banktypes.GenesisBalancesIterator,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "testnet",
		Short: "Initialize files for an Injective testnet",
		Long: `testnet will create "v" number of directories and populate each with
necessary files (private validator, genesis, config, etc.).

Note, strict routability for addresses is turned off in the config file.`,

		Example: "injectived testnet --v 4 --keyring-backend test --output-dir ./output --ip-addresses 192.168.10.2",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			outputDir, _ := cmd.Flags().GetString(flagOutputDir)
			keyringBackend, _ := cmd.Flags().GetString(flags.FlagKeyringBackend)
			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			minGasPrices, _ := cmd.Flags().GetString(server.FlagMinGasPrices)
			nodeDirPrefix, _ := cmd.Flags().GetString(flagNodeDirPrefix)
			nodeDaemonHome, _ := cmd.Flags().GetString(flagNodeDaemonHome)
			ipAddresses, _ := cmd.Flags().GetStringSlice(flagIPAddrs)
			numValidators, _ := cmd.Flags().GetInt(flagNumValidators)
			algo, _ := cmd.Flags().GetString(flags.FlagKeyAlgorithm)
			coinDenom, _ := cmd.Flags().GetString(flagCoinDenom)

			if len(ipAddresses) == 0 {
				return errors.New("IP address list cannot be empty")
			}

			return InitTestnet(
				clientCtx, cmd, config, mbm, genBalancesIterator, outputDir, chainID, coinDenom, minGasPrices,
				nodeDirPrefix, nodeDaemonHome, keyringBackend, algo, ipAddresses, numValidators,
			)
		},
	}

	cmd.Flags().Int(flagNumValidators, 4, "Number of validators to initialize the testnet with")
	cmd.Flags().StringP(flagOutputDir, "o", "./mytestnet", "Directory to store initialization data for the testnet")
	cmd.Flags().String(flagNodeDirPrefix, "node", "Prefix the directory name for each node with (node results in node0, node1, ...)")
	cmd.Flags().String(flagNodeDaemonHome, "injectived", "Home directory of the node's daemon configuration")
	cmd.Flags().StringSlice(flagIPAddrs, []string{"192.168.0.1"}, "List of IP addresses to use (i.e. `192.168.0.1,172.168.0.1` results in persistent peers list ID0@192.168.0.1:46656, ID1@172.168.0.1)")
	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().String(server.FlagMinGasPrices, "", "Minimum gas prices to accept for transactions; All fees in a tx must meet this minimum (e.g. 0.01inj,0.001stake)")
	cmd.Flags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring's backend (os|file|test)")
	cmd.Flags().String(flags.FlagKeyAlgorithm, string(hd.EthSecp256k1Type), "Key signing algorithm to generate keys for")
	cmd.Flags().String(flagCoinDenom, chaintypes.InjectiveCoin, "Coin denomination used for staking, governance, mint, crisis and other parameters")
	return cmd
}

// InitTestnet initializes the testnet configuration
func InitTestnet(
	clientCtx client.Context,
	cmd *cobra.Command,
	nodeConfig *tmconfig.Config,
	mbm module.BasicManager,
	genBalIterator banktypes.GenesisBalancesIterator,
	outputDir,
	chainID,
	coinDenom,
	minGasPrices,
	nodeDirPrefix,
	nodeDaemonHome,
	keyringBackend,
	algoStr string,
	ipAddresses []string,
	numValidators int,
) error {

	if chainID == "" {
		chainID = fmt.Sprintf("injective-%d", tmrand.Int63n(9999999999999)+1)
	}

	// is the amount of staking tokens required for 1 unit of consensus-engine power
	sdk.DefaultPowerReduction = sdk.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	if !chaintypes.IsValidChainID(chainID) {
		return fmt.Errorf("invalid chain-id: %s", chainID)
	}

	if err := sdk.ValidateDenom(coinDenom); err != nil {
		return err
	}

	if len(ipAddresses) != 0 {
		numValidators = len(ipAddresses)
	}

	nodeIDs := make([]string, numValidators)
	valPubKeys := make([]cryptotypes.PubKey, numValidators)

	appConfig := config.DefaultConfig()
	appConfig.MinGasPrices = minGasPrices
	appConfig.API.Enable = true
	appConfig.Telemetry.Enabled = true
	appConfig.Telemetry.PrometheusRetentionTime = 60
	appConfig.Telemetry.EnableHostnameLabel = false
	appConfig.Telemetry.GlobalLabels = [][]string{{"chain_id", chainID}}

	var (
		genAccounts []authtypes.GenesisAccount
		genBalances []banktypes.Balance
		genFiles    []string
	)

	inBuf := bufio.NewReader(cmd.InOrStdin())
	// generate private keys, node IDs, and initial transactions
	for i := 0; i < numValidators; i++ {
		nodeDirName := fmt.Sprintf("%s%d", nodeDirPrefix, i)
		nodeDir := filepath.Join(outputDir, nodeDirName, nodeDaemonHome)
		gentxsDir := filepath.Join(outputDir, "gentxs")

		nodeConfig.SetRoot(nodeDir)
		nodeConfig.RPC.ListenAddress = "tcp://0.0.0.0:26657"
		nodeConfig.Consensus.TimeoutCommit = 1500 * time.Millisecond

		if err := os.MkdirAll(filepath.Join(nodeDir, "config"), nodeDirPerm); err != nil {
			_ = os.RemoveAll(outputDir)
			return err
		}

		nodeConfig.Moniker = nodeDirName

		var (
			ip  string
			err error
		)

		if len(ipAddresses) == 1 {
			ip, err = getIP(i, ipAddresses[0])
			if err != nil {
				_ = os.RemoveAll(outputDir)
				return err
			}
		} else {
			ip = ipAddresses[i]
		}

		nodeIDs[i], valPubKeys[i], err = genutil.InitializeNodeValidatorFiles(nodeConfig)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return err
		}

		memo := fmt.Sprintf("%s@%s:26656", nodeIDs[i], ip)
		genFiles = append(genFiles, nodeConfig.GenesisFile())

		kb, err := keyring.New(
			sdk.KeyringServiceName(),
			keyringBackend,
			nodeDir,
			inBuf,
			hd.EthSecp256k1Option(),
		)
		if err != nil {
			return err
		}

		keyringAlgos, _ := kb.SupportedAlgorithms()
		algo, err := keyring.NewSigningAlgoFromString(algoStr, keyringAlgos)
		if err != nil {
			return err
		}
		addr, secret, err := testutil.GenerateSaveCoinKey(kb, nodeDirName, "", true, algo)
		if err != nil {
			_ = os.RemoveAll(outputDir)
			return err
		}

		info := map[string]string{"secret": secret}

		cliPrint, err := json.Marshal(info)
		if err != nil {
			return err
		}

		// save private key seed words
		if err := writeFile(fmt.Sprintf("%v.json", "key_seed"), nodeDir, cliPrint); err != nil {
			return err
		}

		accStakingTokens := sdk.TokensFromConsensusPower(2500000, sdk.DefaultPowerReduction)
		coins := sdk.NewCoins(
			sdk.NewCoin(coinDenom, accStakingTokens),
		)

		genBalances = append(genBalances, banktypes.Balance{Address: addr.String(), Coins: coins})
		genAccounts = append(genAccounts, &chaintypes.EthAccount{
			BaseAccount: authtypes.NewBaseAccount(addr, nil, 0, 0),
			CodeHash:    ethcrypto.Keccak256(nil),
		})

		valTokens := sdk.TokensFromConsensusPower(2400000, sdk.DefaultPowerReduction)
		createValMsg, err := stakingtypes.NewMsgCreateValidator(
			sdk.ValAddress(addr),
			valPubKeys[i],
			sdk.NewCoin(coinDenom, valTokens),
			stakingtypes.NewDescription(nodeDirName, "", "", "", ""),
			stakingtypes.NewCommissionRates(sdk.OneDec(), sdk.OneDec(), sdk.OneDec()),
			sdk.OneInt(),
		)

		if err != nil {
			return err
		}

		txBuilder := clientCtx.TxConfig.NewTxBuilder()
		if err := txBuilder.SetMsgs(createValMsg); err != nil {
			return err
		}

		txBuilder.SetMemo(memo)

		txFactory := tx.Factory{}
		txFactory = txFactory.
			WithChainID(chainID).
			WithMemo(memo).
			WithKeybase(kb).
			WithTxConfig(clientCtx.TxConfig)

		if err := tx.Sign(txFactory, nodeDirName, txBuilder, false); err != nil {
			return err
		}

		txBz, err := clientCtx.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
		if err != nil {
			return err
		}

		if err := writeFile(fmt.Sprintf("%v.json", nodeDirName), gentxsDir, txBz); err != nil {
			return err
		}

		config.WriteConfigFile(filepath.Join(nodeDir, filepath.Join("config", "app.toml")), appConfig)

		ethPrivKey, err := keyring.NewUnsafe(kb).UnsafeExportPrivKeyHex(nodeDirName)
		if err != nil {
			return err
		}
		initPeggo(outputDir, nodeDirName, []byte(strings.ToUpper(ethPrivKey)))
	}

	if err := initGenFiles(clientCtx, mbm, chainID, coinDenom, genAccounts, genBalances, genFiles, numValidators); err != nil {
		return err
	}

	err := collectGenFiles(
		clientCtx, nodeConfig, chainID, nodeIDs, valPubKeys, numValidators,
		outputDir, nodeDirPrefix, nodeDaemonHome, genBalIterator,
	)
	if err != nil {
		return err
	}

	cmd.PrintErrf("Successfully initialized %d node directories\n", numValidators)
	return nil
}

func initPeggo(outputDir, nodeDirName string, _ []byte) {
	peggoDir := filepath.Join(outputDir, nodeDirName, "peggo")
	if envdata, _ := os.ReadFile("./templates/peggo_config.template"); len(envdata) > 0 {
		s := bufio.NewScanner(bytes.NewReader(envdata))
		for s.Scan() {
			parts := strings.Split(s.Text(), "=")
			if len(parts) != 2 {
				continue
			} else {
				content := []byte(s.Text())
				if parts[0] == "PEGGO_COSMOS_PK" {
					newPrivkey, _ := ethsecp256k1.GenerateKey()
					privKeyStr := common.Bytes2Hex(newPrivkey.GetKey())
					privKeyBytes := []byte(strings.ToUpper(privKeyStr))
					content = append([]byte(parts[0]+"="), privKeyBytes...)
				} else if parts[0] == "PEGGO_ETH_PK" {
					newPrivkey, _ := ethsecp256k1.GenerateKey()
					privKeyStr := common.Bytes2Hex(newPrivkey.GetKey())
					privKeyBytes := []byte(strings.ToUpper(privKeyStr))
					content = append([]byte(parts[0]+"="), privKeyBytes...)
				}
				if err := appendToFile("config.env", peggoDir, content); err != nil {
					fmt.Println("Error writing peggo config", "error", err)
				}
			}
		}
	}
}

func initGenFiles(
	clientCtx client.Context,
	mbm module.BasicManager,
	chainID,
	coinDenom string,
	genAccounts []authtypes.GenesisAccount,
	genBalances []banktypes.Balance,
	genFiles []string,
	numValidators int,
) error {

	appGenState := mbm.DefaultGenesis(clientCtx.Codec)

	// set the accounts in the genesis state
	var authGenState authtypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[authtypes.ModuleName], &authGenState)

	accounts, err := authtypes.PackAccounts(genAccounts)
	if err != nil {
		return err
	}

	authGenState.Accounts = accounts
	appGenState[authtypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&authGenState)

	// set the balances in the genesis state
	var bankGenState banktypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[banktypes.ModuleName], &bankGenState)
	bankGenState.Params.DefaultSendEnabled = true
	var totalSupply sdk.Coins
	bankGenState.Balances = banktypes.SanitizeGenesisBalances(genBalances)
	for _, balance := range bankGenState.Balances {
		totalSupply = totalSupply.Add(balance.Coins...)
	}
	bankGenState.Supply = totalSupply
	appGenState[banktypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&bankGenState)

	// staking genesis params
	var stakingGenState stakingtypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[stakingtypes.ModuleName], &stakingGenState)
	stakingGenState.Params.BondDenom = coinDenom
	stakingGenState.Params.MaxValidators = uint32(20)
	appGenState[stakingtypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&stakingGenState)

	// slashing genesis params
	var slashingGenState slashingtypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[slashingtypes.ModuleName], &slashingGenState)
	slashingGenState.Params.SignedBlocksWindow = int64(40000)
	slashingGenState.Params.SlashFractionDowntime = sdk.NewDecWithPrec(1, 4)
	appGenState[slashingtypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&slashingGenState)

	// gov genesis params
	var govGenState govtypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[govtypes.ModuleName], &govGenState)
	govGenState.DepositParams.MinDeposit[0].Denom = coinDenom
	govGenState.DepositParams.MinDeposit = sdk.NewCoins(sdk.NewCoin(coinDenom, sdk.TokensFromConsensusPower(500, sdk.DefaultPowerReduction)))
	appGenState[govtypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&govGenState)

	// mint genesis params
	var mintGenState mintypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[mintypes.ModuleName], &mintGenState)
	mintGenState.Params.MintDenom = coinDenom
	mintGenState.Params.InflationRateChange = sdk.NewDecWithPrec(5, 2)
	mintGenState.Params.InflationMin = sdk.NewDecWithPrec(4, 2)
	mintGenState.Params.InflationMax = sdk.NewDecWithPrec(10, 2)
	mintGenState.Params.BlocksPerYear = uint64(60 * 60 * 8766 / 2)
	mintGenState.Minter.Inflation = sdk.NewDecWithPrec(5, 2)
	appGenState[mintypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&mintGenState)

	// crisis genesis params
	var crisisGenState crisistypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[crisistypes.ModuleName], &crisisGenState)

	crisisGenState.ConstantFee.Denom = coinDenom
	crisisGenState.ConstantFee.Amount = sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction)
	appGenState[crisistypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&crisisGenState)

	// ibc genesis params
	var ibcTransferGenState ibctransfertypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[ibctransfertypes.ModuleName], &ibcTransferGenState)
	ibcTransferGenState.Params.SendEnabled = false
	ibcTransferGenState.Params.ReceiveEnabled = false

	appGenState[ibctransfertypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&ibcTransferGenState)

	// peggy genesis params
	var peggyGenState peggytypes.GenesisState
	clientCtx.Codec.MustUnmarshalJSON(appGenState[peggytypes.ModuleName], &peggyGenState)
	peggyGenState.Params.AverageBlockTime = uint64(2000)
	peggyGenState.Params.AverageEthereumBlockTime = uint64(15000)
	peggyGenState.Params.SignedValsetsWindow = uint64(25000)
	peggyGenState.Params.SignedBatchesWindow = uint64(25000)
	peggyGenState.Params.SignedClaimsWindow = uint64(25000)
	peggyGenState.Params.UnbondSlashingValsetsWindow = uint64(25000)
	peggyGenState.Params.BridgeChainId = uint64(1)
	peggyGenState.Params.CosmosCoinErc20Contract = common.HexToAddress("0xe28b3B32B6c345A34Ff64674606124Dd5Aceca30").String()
	appGenState[peggytypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&peggyGenState)

	appGenStateJSON, err := json.MarshalIndent(appGenState, "", "  ")
	if err != nil {
		return err
	}

	genDoc := types.GenesisDoc{
		ChainID:    chainID,
		AppState:   appGenStateJSON,
		Validators: nil,
	}

	// generate empty genesis files for each validator and save
	for i := 0; i < numValidators; i++ {
		if err := genDoc.SaveAs(genFiles[i]); err != nil {
			return err
		}
	}
	return nil
}

func collectGenFiles(
	clientCtx client.Context, nodeConfig *tmconfig.Config, chainID string,
	nodeIDs []string, valPubKeys []cryptotypes.PubKey, numValidators int,
	outputDir, nodeDirPrefix, nodeDaemonHome string, genBalIterator banktypes.GenesisBalancesIterator,
) error {

	var appState json.RawMessage
	genTime := tmtime.Now()

	for i := 0; i < numValidators; i++ {
		nodeDirName := fmt.Sprintf("%s%d", nodeDirPrefix, i)
		nodeDir := filepath.Join(outputDir, nodeDirName, nodeDaemonHome)
		gentxsDir := filepath.Join(outputDir, "gentxs")
		nodeConfig.Moniker = nodeDirName

		nodeConfig.SetRoot(nodeDir)

		nodeID, valPubKey := nodeIDs[i], valPubKeys[i]
		initCfg := genutiltypes.NewInitConfig(chainID, gentxsDir, nodeID, valPubKey)

		genDoc, err := types.GenesisDocFromFile(nodeConfig.GenesisFile())
		if err != nil {
			return err
		}

		nodeAppState, err := genutil.GenAppStateFromConfig(clientCtx.Codec, clientCtx.TxConfig, nodeConfig, initCfg, *genDoc, genBalIterator)
		if err != nil {
			return err
		}

		if appState == nil {
			// set the canonical application state (they should not differ)
			appState = nodeAppState
		}

		genFile := nodeConfig.GenesisFile()

		// overwrite each validator's genesis file to have a canonical genesis time
		if err := genutil.ExportGenesisFileWithTime(genFile, chainID, nil, appState, genTime); err != nil {
			return err
		}
	}

	return nil
}

func getIP(i int, startingIPAddr string) (ip string, err error) {
	if startingIPAddr == "" {
		ip, err = server.ExternalIP()
		if err != nil {
			return "", err
		}
		return ip, nil
	}
	return calculateIP(startingIPAddr, i)
}

func calculateIP(ip string, i int) (string, error) {
	ipv4 := net.ParseIP(ip).To4()
	if ipv4 == nil {
		return "", fmt.Errorf("%v: non ipv4 address", ip)
	}

	for j := 0; j < i; j++ {
		ipv4[3]++
	}

	return ipv4.String(), nil
}

func writeFile(name, dir string, contents []byte) error {
	writePath := filepath.Clean(dir)
	file := filepath.Join(writePath, name)

	err := tmos.EnsureDir(writePath, 0o755)
	if err != nil {
		return err
	}

	err = tmos.WriteFile(file, contents, 0o644)
	if err != nil {
		return err
	}

	return nil
}

func appendToFile(name, dir string, contents []byte) error {
	writePath := filepath.Clean(dir)
	file := filepath.Join(dir, name)

	err := tmos.EnsureDir(writePath, 0o755)
	if err != nil {
		return err
	}

	if _, err = os.Stat(file); err == nil {
		err = os.Chmod(file, 0o777)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()

	_, err = f.Write(contents)
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = f.WriteString("\n")
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
