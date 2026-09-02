//revive:disable-next-line:package-directory-mismatch // Semver upgrade directory names cannot be valid Go package identifiers.
package v1dot20dot3safeharbor2

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

const (
	nativeUSDC = "erc20:0xa00C59fF5a080D2b954d0c75e46E22a0c371235a"
	peggyUSDT  = "peggy0xdAC17F958D2ee523a2206206994597C13D831ec7"

	recoveryEventType        = "safeharbor_default_subaccount_recovery"
	recoverySkippedEventType = "safeharbor_default_subaccount_recovery_skipped"
)

// DefaultSubaccountRecovery is one incident-attributed nonce-0 balance recovery.
// IncidentAmount is the amount attributed in the incident snapshot, expressed in the denom's
// atomic bank units. It is retained for auditing; the upgrade transfers the current sweepable
// balance so that balances already swept before the upgrade are not paid twice.
type DefaultSubaccountRecovery struct {
	Recipient      string
	Denom          string
	IncidentAmount int64
}

// The manifest was derived by comparing mainnet exchange state immediately before the first
// exploit transaction (height 180945883) with the frozen post-incident state (height 181027005).
// Source CSV SHA-256: 3f34284734e67a2e94333155834c24e0d957e0314e48d2de4cb1e9eae176bec4.
//
// Only balances whose positive integer available amount increased during the incident are listed.
// Pre-existing nonce-0 anomalies and sub-atomic dust are deliberately excluded.
var defaultSubaccountRecoveries = []DefaultSubaccountRecovery{
	{Recipient: "inj1035p9p6hhn9pzjk00xpa5uuf7gptf0c44p5ch8", Denom: nativeUSDC, IncidentAmount: 183915552},
	{Recipient: "inj10ecjqkt0hplsyeksysrt8ypfj5tu6vp5d86p6e", Denom: nativeUSDC, IncidentAmount: 148170975077},
	{Recipient: "inj10gt0vfl4xv7c5x9hgedu845j8uddm06udk5508", Denom: nativeUSDC, IncidentAmount: 10069063},
	{Recipient: "inj10qm07mgtqvtd9gxgmjp70nnu6esn7fujzj382f", Denom: nativeUSDC, IncidentAmount: 194435931},
	{Recipient: "inj13cgtdayv6kmy8ns220v4rteuwjeqpmcvx5afxl", Denom: nativeUSDC, IncidentAmount: 11998997778},
	{Recipient: "inj147yzzlprwtzsz2jlcnx8yqm8msd2lmp7pt6lgf", Denom: nativeUSDC, IncidentAmount: 603559838},
	{Recipient: "inj159e4vyhd0r2fhst5uw2xwff9s446vpfh8n9zwu", Denom: nativeUSDC, IncidentAmount: 684622873},
	{Recipient: "inj15rjtvtjmsqwvxcgms97dww6zlwjz443g7anrcs", Denom: nativeUSDC, IncidentAmount: 474707674},
	{Recipient: "inj173c3u4l637f07wlz7qqutg4sdmfm4gxtak86w3", Denom: nativeUSDC, IncidentAmount: 1510401028},
	{Recipient: "inj17gkuet8f6pssxd8nycm3qr9d9y699rupv6397z", Denom: nativeUSDC, IncidentAmount: 19815700},
	{Recipient: "inj17psdthys834uwa9jvd5j8fn5j8c9smxe3jgxnd", Denom: nativeUSDC, IncidentAmount: 1000000000},
	{Recipient: "inj17v92f7ddajutkgylwexnqr9l0q6p6hj4uvkrt3", Denom: nativeUSDC, IncidentAmount: 465620000},
	{Recipient: "inj1859yqmyrqafxc6ru9g9ddgc7uuq8fda3wd5rc0", Denom: nativeUSDC, IncidentAmount: 3919131738},
	{Recipient: "inj18fvdjvf2vw33rj0vqza7860nhqu8dec2nzg3nt", Denom: nativeUSDC, IncidentAmount: 145647940},
	{Recipient: "inj19k6cwnf32mlrrer4pf58xcdttkpuyw20p5e8q6", Denom: nativeUSDC, IncidentAmount: 184039163},
	{Recipient: "inj19sllrf03jz2mt8kajp7atg3ua8zuf8s2sqvc4a", Denom: nativeUSDC, IncidentAmount: 30423158},
	{Recipient: "inj1cxe65zdyly8p89n822zwd0gmquymndchhldq79", Denom: nativeUSDC, IncidentAmount: 1897749875},
	{Recipient: "inj1d46qg9rvn94fve7lnvsx7hw2w7t8w3lvlqavgw", Denom: nativeUSDC, IncidentAmount: 34169482962},
	{Recipient: "inj1e44tvyuwpmarc9qdnxgu6e050ucv57r3dj6s00", Denom: nativeUSDC, IncidentAmount: 186095744},
	{Recipient: "inj1exyaj8xflhqvrs5x30lxms4egya4v527tzu2xq", Denom: nativeUSDC, IncidentAmount: 561612905846},
	{Recipient: "inj1hxx0rl2txwa2r84zu5f348cxhz9wnfxrx2pqfy", Denom: nativeUSDC, IncidentAmount: 16389332},
	{Recipient: "inj1k8txa2875vzlgf8afqn3lz9gv23rv6k9we59zj", Denom: nativeUSDC, IncidentAmount: 25849847690},
	{Recipient: "inj1kmvwyexc3uc2v79l33w59tx7ywhxshtjs08324", Denom: nativeUSDC, IncidentAmount: 22278494501},
	{Recipient: "inj1l4u0q20c0xxtylsxegqz332nnzwz04xwj6l4k7", Denom: nativeUSDC, IncidentAmount: 717997658616},
	{Recipient: "inj1le7uy7a6gcnx43u6pmqe0j8urz75zpg6fzfdw4", Denom: nativeUSDC, IncidentAmount: 842760000},
	{Recipient: "inj1lre9mvd654x4c8ulywqz6mxec5wwsygw3atdx2", Denom: nativeUSDC, IncidentAmount: 1734668549},
	{Recipient: "inj1m0rj0ethfngelcluvsz5c2d5w4t69lxd3fvemu", Denom: nativeUSDC, IncidentAmount: 699089428},
	{Recipient: "inj1ntzp6egl4z6e7gfmvsc63mh8ee5h4m2xqhn3lk", Denom: nativeUSDC, IncidentAmount: 6428743427},
	{Recipient: "inj1plde26u9vvy39atdrnmmazkzey37gplh6f6pnp", Denom: nativeUSDC, IncidentAmount: 97558581},
	{Recipient: "inj1pmrke9w9wqvk3ev8z9zn7jceyzck7pkgchhqsu", Denom: nativeUSDC, IncidentAmount: 86283253},
	{Recipient: "inj1qmc7w099c5dwrh072wnpwfev5gafmu6p296uhh", Denom: nativeUSDC, IncidentAmount: 1913530},
	{Recipient: "inj1suyl0ex5st8tasws005ypvdaf2q6nxw4hhpp3y", Denom: nativeUSDC, IncidentAmount: 140409173},
	{Recipient: "inj1t5224j26ra0s48nv87lwk3jeewc55unntdk7u5", Denom: nativeUSDC, IncidentAmount: 15950068},
	{Recipient: "inj1tfp4vzungc4k6mlufkct6v74pz0h0quns6prdz", Denom: nativeUSDC, IncidentAmount: 133783736},
	{Recipient: "inj1tvxqtjq9kfgl3wwnxpvr824hnxvszp2ya4c7gj", Denom: nativeUSDC, IncidentAmount: 882424889},
	{Recipient: "inj1w587jqdxpygatpl8jt6j74g8hh36tw5vmecuek", Denom: nativeUSDC, IncidentAmount: 685144018117},
	{Recipient: "inj1wh2qzvmxd5kh972rgkl80zuqxccn8u5rg3txu4", Denom: nativeUSDC, IncidentAmount: 24851296325},
	{Recipient: "inj1x6rqcr3exejqrs3q4sj5k6knxd0l48jkpdfd6q", Denom: nativeUSDC, IncidentAmount: 182975285},
	{Recipient: "inj1yh64la2wxxwyhagxcwwpe5she3xclyvp7tf3ue", Denom: nativeUSDC, IncidentAmount: 51989415},
	{Recipient: "inj1zdf549dwfl4m3p3nuq9v5zclsuzfarusue5lyn", Denom: nativeUSDC, IncidentAmount: 18484414510},
	{Recipient: "inj1zxxgn2pfdscdeuhl84na0yjwxgg8xa7urta0l4", Denom: nativeUSDC, IncidentAmount: 284258997},
	{Recipient: "inj12yj3mtjarujkhcp6lg3klxjjfrx2v7v8yswgp9", Denom: peggyUSDT, IncidentAmount: 213354464},
	{Recipient: "inj1cnn83qje9t8p9vhpk9fc8fdd04z5yca8rveucw", Denom: peggyUSDT, IncidentAmount: 145810700},
	{Recipient: "inj1j5mr2hmv7y2z7trazganj75u8km8jvdfuxncsp", Denom: peggyUSDT, IncidentAmount: 10773416},
	{Recipient: "inj1q7lh9zu38kppal96ujmh560s6042r2zxmkv2h0", Denom: peggyUSDT, IncidentAmount: 40400640},
	{Recipient: "inj1wj7apry4gva43khquwaqyxd587cd9g3425tagc", Denom: peggyUSDT, IncidentAmount: 10808779},
}

// DefaultSubaccountRecoveries returns a copy of the mainnet recovery manifest for auditing and tests.
func DefaultSubaccountRecoveries() []DefaultSubaccountRecovery {
	return append([]DefaultSubaccountRecovery(nil), defaultSubaccountRecoveries...)
}

type plannedDefaultSubaccountRecovery struct {
	record       DefaultSubaccountRecovery
	recipient    sdk.AccAddress
	subaccountID common.Hash
	amount       math.Int
}

// RecoverDefaultSubaccountBalancesBestEffort attempts the recovery atomically without making chain
// liveness depend on its success. A skipped recovery remains available for a later remediation.
func RecoverDefaultSubaccountBalancesBestEffort(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	logger log.Logger,
) error {
	recoveryCtx, commitRecovery := ctx.CacheContext()
	if err := RecoverDefaultSubaccountBalances(recoveryCtx, app, logger); err != nil {
		logger.Error("Skipped incident default subaccount recovery", "error", err)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			recoverySkippedEventType,
			sdk.NewAttribute("reason", err.Error()),
		))
		return nil
	}

	commitRecovery()
	return nil
}

// RecoverDefaultSubaccountBalances retries the normal default-subaccount sweep for the verified
// incident recipients. The amount is derived from state at the upgrade height because a recovery
// entry may have already swept, accumulated more funds, or become locked since the incident
// snapshot. The matching exchange deposits are decremented by exactly the amounts sent, so the
// operation does not change the module's solvency gap.
//
// This migration does not recapitalize the exchange module. Its backing must be restored separately
// before the upgrade; the liquidity check below only guarantees that this recovery batch is atomic.
//
// The best-effort upgrade wrapper supplies a cached context. Any error therefore discards all bank
// transfers and deposit writes from this function instead of committing a partial recovery.
func RecoverDefaultSubaccountBalances(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	logger log.Logger,
) error {
	planned, requiredModuleCoins, err := planDefaultSubaccountRecoveries(ctx, app, logger)
	if err != nil {
		return err
	}

	if err := ensureRecoveryLiquidity(ctx, app, requiredModuleCoins); err != nil {
		return err
	}

	if err := executeDefaultSubaccountRecoveries(ctx, app, planned); err != nil {
		return err
	}

	logger.Info(
		"Recovered incident default subaccount balances",
		"accounts", len(planned),
		"skipped_accounts", len(defaultSubaccountRecoveries)-len(planned),
		"amounts", requiredModuleCoins.String(),
	)

	return nil
}

func planDefaultSubaccountRecoveries(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	logger log.Logger,
) ([]plannedDefaultSubaccountRecovery, sdk.Coins, error) {
	exchangeKeeper := app.GetExchangeKeeper()
	planned := make([]plannedDefaultSubaccountRecovery, 0, len(defaultSubaccountRecoveries))
	requiredModuleCoins := sdk.NewCoins()
	seen := make(map[string]struct{}, len(defaultSubaccountRecoveries))

	for _, record := range defaultSubaccountRecoveries {
		if record.IncidentAmount <= 0 {
			return nil, nil, errors.Errorf("invalid recovery amount %d for %s", record.IncidentAmount, record.Recipient)
		}
		if record.Denom != nativeUSDC && record.Denom != peggyUSDT {
			return nil, nil, errors.Errorf("unexpected recovery denom %s for %s", record.Denom, record.Recipient)
		}

		recipient, err := sdk.AccAddressFromBech32(record.Recipient)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "decode recovery recipient %s", record.Recipient)
		}
		subaccountID := exchangetypes.SdkAddressToSubaccountID(recipient)
		key := subaccountID.Hex() + "/" + record.Denom
		if _, duplicate := seen[key]; duplicate {
			return nil, nil, errors.Errorf("duplicate recovery entry %s", key)
		}
		seen[key] = struct{}{}

		deposit := exchangeKeeper.GetDeposit(ctx, subaccountID, record.Denom)
		availableAmount := math.MaxInt(deposit.AvailableBalance.TruncateInt(), math.ZeroInt())
		totalAmount := math.MaxInt(deposit.TotalBalance.TruncateInt(), math.ZeroInt())
		amount := math.MinInt(availableAmount, totalAmount)
		if !amount.IsPositive() {
			continue
		}

		if amount.GT(math.NewInt(record.IncidentAmount)) {
			logger.Warn(
				"Current default subaccount balance exceeds the incident-attributed amount",
				"recipient", record.Recipient,
				"denom", record.Denom,
				"incident_amount", record.IncidentAmount,
				"current_sweep_amount", amount.String(),
			)
		}

		planned = append(planned, plannedDefaultSubaccountRecovery{
			record:       record,
			recipient:    recipient,
			subaccountID: subaccountID,
			amount:       amount,
		})
		requiredModuleCoins = requiredModuleCoins.Add(sdk.NewCoin(record.Denom, amount))
	}

	return planned, requiredModuleCoins, nil
}

func ensureRecoveryLiquidity(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	requiredModuleCoins sdk.Coins,
) error {
	bankKeeper := app.GetBankKeeper()
	moduleAddress := app.GetAccountKeeper().GetModuleAddress(exchangetypes.ModuleName)

	for _, required := range requiredModuleCoins {
		balance := bankKeeper.GetBalance(ctx, moduleAddress, required.Denom)
		if balance.Amount.LT(required.Amount) {
			return errors.Errorf(
				"exchange module must have recovery liquidity before the upgrade: need %s, have %s",
				required,
				balance,
			)
		}
	}

	return nil
}

func executeDefaultSubaccountRecoveries(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	planned []plannedDefaultSubaccountRecovery,
) error {
	bankKeeper := app.GetBankKeeper()
	exchangeKeeper := app.GetExchangeKeeper()

	for _, recovery := range planned {
		coin := sdk.NewCoin(recovery.record.Denom, recovery.amount)
		if err := bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			exchangetypes.ModuleName,
			recovery.recipient,
			sdk.NewCoins(coin),
		); err != nil {
			return errors.Wrapf(err, "send recovery %s to %s", coin, recovery.record.Recipient)
		}

		deposit := exchangeKeeper.GetDeposit(ctx, recovery.subaccountID, recovery.record.Denom)
		amountDec := recovery.amount.ToLegacyDec()
		deposit.AvailableBalance = deposit.AvailableBalance.Sub(amountDec)
		deposit.TotalBalance = deposit.TotalBalance.Sub(amountDec)
		exchangeKeeper.SetDeposit(ctx, recovery.subaccountID, recovery.record.Denom, deposit)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			recoveryEventType,
			sdk.NewAttribute("recipient", recovery.record.Recipient),
			sdk.NewAttribute("subaccount_id", recovery.subaccountID.Hex()),
			sdk.NewAttribute("amount", coin.String()),
			sdk.NewAttribute("incident_amount", math.NewInt(recovery.record.IncidentAmount).String()),
		))
	}

	return nil
}
