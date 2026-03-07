package accounts

import (
	"io"
	"time"

	"github.com/OffchainLabs/prysm/v7/api/rest"
	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/cmd/validator/flags"
	"github.com/OffchainLabs/prysm/v7/validator/accounts"
	"github.com/OffchainLabs/prysm/v7/validator/accounts/wallet"
	"github.com/OffchainLabs/prysm/v7/validator/keymanager"
	"github.com/OffchainLabs/prysm/v7/validator/keymanager/local"
	"github.com/OffchainLabs/prysm/v7/validator/node"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func Exit(c *cli.Context, r io.Reader) error {
	var w *wallet.Wallet
	var km keymanager.IKeymanager
	var err error
	beaconApiEndpoint := c.String(flags.BeaconRESTApiProviderFlag.Name)
	if !c.IsSet(flags.Web3SignerURLFlag.Name) && !c.IsSet(flags.WalletDirFlag.Name) && !c.IsSet(flags.InteropNumValidators.Name) {
		return errors.Errorf("No validators found, please provide a prysm wallet directory via flag --%s "+
			"or a remote signer location with corresponding public keys via flags --%s and --%s ",
			flags.WalletDirFlag.Name,
			flags.Web3SignerURLFlag.Name,
			flags.Web3SignerPublicValidatorKeysFlag,
		)
	}
	if c.IsSet(flags.InteropNumValidators.Name) {
		km, err = local.NewInteropKeymanager(c.Context, c.Uint64(flags.InteropStartIndex.Name), c.Uint64(flags.InteropNumValidators.Name))
		if err != nil {
			return errors.Wrap(err, "could not generate interop keys for key manager")
		}
		w = &wallet.Wallet{}
	} else if c.IsSet(flags.Web3SignerURLFlag.Name) {
		// Fetch genesis info via REST API to get genesis_validators_root.
		restProvider, err := rest.NewRestConnectionProvider(
			beaconApiEndpoint,
			rest.WithHttpTimeout(time.Second*30),
		)
		if err != nil {
			return errors.Wrapf(err, "could not create REST connection to %s", beaconApiEndpoint)
		}
		genesisResp := &structs.GetGenesisResponse{}
		if err := restProvider.Handler().Get(c.Context, "/eth/v1/beacon/genesis", genesisResp); err != nil {
			return errors.Wrap(err, "failed to get genesis info")
		}
		if genesisResp.Data == nil {
			return errors.New("genesis data is nil")
		}
		genesisValidatorsRoot, err := hexutil.Decode(genesisResp.Data.GenesisValidatorsRoot)
		if err != nil {
			return errors.Wrap(err, "failed to decode genesis validators root")
		}
		config, err := node.Web3SignerConfig(c)
		if err != nil {
			return errors.Wrapf(err, "could not configure remote signer")
		}
		config.GenesisValidatorsRoot = genesisValidatorsRoot
		w, km, err = walletWithWeb3SignerKeymanager(c, config)
		if err != nil {
			return err
		}
	} else {
		w, km, err = walletWithKeymanager(c)
		if err != nil {
			return err
		}
	}

	opts := []accounts.Option{
		accounts.WithWallet(w),
		accounts.WithKeymanager(km),
		accounts.WithBeaconRESTApiProvider(beaconApiEndpoint),
		accounts.WithExitJSONOutputPath(c.String(flags.VoluntaryExitJSONOutputPathFlag.Name)),
	}
	// Get full set of public keys from the keymanager.
	validatingPublicKeys, err := km.FetchValidatingPublicKeys(c.Context)
	if err != nil {
		return err
	}
	if len(validatingPublicKeys) == 0 {
		return errors.New("wallet is empty, no accounts to delete")
	}
	// Filter keys either from CLI flag or from interactive session.
	rawPubKey, formattedPubKeys, err := accounts.FilterExitAccountsFromUserInput(c, r, validatingPublicKeys, c.Bool(flags.ForceExitFlag.Name))
	if err != nil {
		return errors.Wrap(err, "could not filter public keys for deletion")
	}
	opts = append(opts, accounts.WithRawPubKeys(rawPubKey))
	opts = append(opts, accounts.WithFormattedPubKeys(formattedPubKeys))
	acc, err := accounts.NewCLIManager(opts...)
	if err != nil {
		return err
	}
	return acc.Exit(c.Context)
}
