package accounts

import (
	"github.com/OffchainLabs/prysm/v7/cmd/validator/flags"
	"github.com/OffchainLabs/prysm/v7/validator/accounts"
	"github.com/urfave/cli/v2"
)

func accountsList(c *cli.Context) error {
	w, km, err := walletWithKeymanager(c)
	if err != nil {
		return err
	}
	opts := []accounts.Option{
		accounts.WithWallet(w),
		accounts.WithKeymanager(km),
		accounts.WithBeaconRESTApiProvider(c.String(flags.BeaconRESTApiProviderFlag.Name)),
	}
	if c.IsSet(flags.ShowPrivateKeysFlag.Name) {
		opts = append(opts, accounts.WithShowPrivateKeys())
	}
	if c.IsSet(flags.ListValidatorIndices.Name) {
		opts = append(opts, accounts.WithListValidatorIndices())
	}
	acc, err := accounts.NewCLIManager(opts...)
	if err != nil {
		return err
	}
	return acc.List(
		c.Context,
	)
}
