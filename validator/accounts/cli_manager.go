package accounts

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/OffchainLabs/prysm/v7/api/rest"
	"github.com/OffchainLabs/prysm/v7/crypto/bls"
	"github.com/OffchainLabs/prysm/v7/validator/accounts/wallet"
	iface "github.com/OffchainLabs/prysm/v7/validator/client/iface"
	nodeClientFactory "github.com/OffchainLabs/prysm/v7/validator/client/node-client-factory"
	validatorClientFactory "github.com/OffchainLabs/prysm/v7/validator/client/validator-client-factory"
	validatorHelpers "github.com/OffchainLabs/prysm/v7/validator/helpers"
	"github.com/OffchainLabs/prysm/v7/validator/keymanager"
	"github.com/OffchainLabs/prysm/v7/validator/keymanager/derived"
)

// NewCLIManager allows for managing validator accounts via CLI commands.
func NewCLIManager(opts ...Option) (*CLIManager, error) {
	acc := &CLIManager{
		mnemonicLanguage: derived.DefaultMnemonicLanguage,
		inputReader:      os.Stdin,
	}
	for _, opt := range opts {
		if err := opt(acc); err != nil {
			return nil, err
		}
	}
	return acc, nil
}

// CLIManager defines a struct capable of performing various validator
// wallet & account operations via the command line.
type CLIManager struct {
	wallet               *wallet.Wallet
	keymanager           keymanager.IKeymanager
	keymanagerKind       keymanager.Kind
	showPrivateKeys      bool
	listValidatorIndices bool
	deletePublicKeys     bool
	importPrivateKeys    bool
	readPasswordFile     bool
	skipMnemonicConfirm  bool
	walletKeyCount       int
	privateKeyFile       string
	passwordFilePath     string
	keysDir              string
	mnemonicLanguage     string
	backupsDir           string
	backupsPassword      string
	filteredPubKeys      []bls.PublicKey
	rawPubKeys           [][]byte
	formattedPubKeys     []string
	exitJSONOutputPath   string
	walletDir            string
	walletPassword       string
	mnemonic             string
	numAccounts          int
	mnemonic25thWord     string
	beaconApiEndpoint    string
	beaconApiTimeout     time.Duration
	inputReader          io.Reader
}

func (acm *CLIManager) prepareBeaconClients(ctx context.Context) (*iface.ValidatorClient, *iface.NodeClient, error) {
	var connOpts []validatorHelpers.NodeConnectionOption
	if acm.beaconApiEndpoint != "" {
		connOpts = append(connOpts, validatorHelpers.WithREST(acm.beaconApiEndpoint, rest.WithHttpTimeout(acm.beaconApiTimeout)))
	}

	conn, err := validatorHelpers.NewNodeConnection(connOpts...)
	if err != nil {
		return nil, nil, err
	}

	validatorClient := validatorClientFactory.NewValidatorClient(conn)
	nodeClient := nodeClientFactory.NewNodeClient(conn)

	return &validatorClient, &nodeClient, nil
}
