package p2p

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/p2p/encoder"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/consensus-types/wrapper"
	ecdsaprysm "github.com/OffchainLabs/prysm/v7/crypto/ecdsa"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	"github.com/OffchainLabs/prysm/v7/monitoring/tracing"
	"github.com/OffchainLabs/prysm/v7/monitoring/tracing/trace"
	"github.com/OffchainLabs/prysm/v7/network"
	pb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1/metadata"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	corenet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	libp2ptcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
)

// chainHead holds chain head data fetched via REST.
type chainHead struct {
	HeadSlot       primitives.Slot
	HeadRoot       []byte
	FinalizedEpoch primitives.Epoch
	FinalizedRoot  []byte
}

// genesisData holds genesis data fetched via REST.
type genesisData struct {
	GenesisTime           time.Time
	GenesisValidatorsRoot [32]byte
}

// A minimal client for peering with beacon nodes over libp2p and sending p2p RPC requests for data.
type client struct {
	host    host.Host
	meta    metadata.Metadata
	baseURL string // REST API base URL, e.g. "http://localhost:3500"
}

func newClient(beaconEndpoints []string, tcpPort, quicPort uint) (*client, error) {
	ipAdd := ipAddr()
	priv, err := privKey()
	if err != nil {
		return nil, errors.Wrap(err, "could not set up p2p private key")
	}
	meta, err := readMetadata()
	if err != nil {
		return nil, errors.Wrap(err, "could not set up p2p metadata")
	}
	multiaddrs, err := p2p.MultiAddressBuilder(ipAdd, tcpPort, quicPort)
	if err != nil {
		return nil, errors.Wrap(err, "could not set up listening multiaddr")
	}
	options := []libp2p.Option{
		privKeyOption(priv),
		libp2p.ListenAddrs(multiaddrs...),
		libp2p.UserAgent(version.BuildData()),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.Transport(libp2ptcp.NewTCPTransport),
	}
	options = append(options, libp2p.Security(noise.ID, noise.New))
	options = append(options, libp2p.Ping(false))
	h, err := libp2p.New(options...)
	if err != nil {
		return nil, errors.Wrap(err, "could not start libp2p")
	}
	if len(beaconEndpoints) == 0 {
		return nil, errors.New("no specified beacon API endpoints")
	}
	return &client{
		host:    h,
		meta:    meta,
		baseURL: beaconEndpoints[0],
	}, nil
}

func (c *client) Close() {
	if err := c.host.Close(); err != nil {
		panic(err) // lint:nopanic -- The client is closing anyway...
	}
}

func (c *client) Encoding() encoder.NetworkEncoding {
	return &encoder.SszNetworkEncoder{}
}

func (c *client) MetadataSeq() uint64 {
	return c.meta.SequenceNumber()
}

// Send a request to specific peer. The returned stream may be used for reading,
// but has been closed for writing.
// When done, the caller must Close() or Reset() on the stream.
func (c *client) Send(
	ctx context.Context,
	message any,
	baseTopic string,
	pid peer.ID,
) (corenet.Stream, error) {
	ctx, span := trace.StartSpan(ctx, "p2p.Send")
	defer span.End()
	topic := baseTopic + c.Encoding().ProtocolSuffix()
	span.SetAttributes(trace.StringAttribute("topic", topic))

	// Apply max dial timeout when opening a new stream.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	stream, err := c.host.NewStream(ctx, pid, protocol.ID(topic))
	if err != nil {
		tracing.AnnotateError(span, err)
		return nil, errors.Wrap(err, "could not open new stream")
	}
	// do not encode anything if we are sending a metadata request
	if baseTopic != p2p.RPCMetaDataTopicV1 && baseTopic != p2p.RPCMetaDataTopicV2 && baseTopic != p2p.RPCMetaDataTopicV3 {
		castedMsg, ok := message.(ssz.Marshaler)
		if !ok {
			return nil, errors.Errorf("%T does not support the ssz marshaller interface", message)
		}
		if _, err := c.Encoding().EncodeWithMaxLength(stream, castedMsg); err != nil {
			tracing.AnnotateError(span, err)
			_err := stream.Reset()
			_ = _err
			return nil, err
		}
	}
	// Close stream for writing.
	if err := stream.CloseWrite(); err != nil {
		tracing.AnnotateError(span, err)
		_err := stream.Reset()
		_ = _err
		return nil, errors.Wrap(err, "could not close write")
	}

	return stream, nil
}

// fetchJSON performs an HTTP GET and decodes the JSON response into result.
func fetchJSON(ctx context.Context, url string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

// getJSON performs an HTTP GET against the client's beacon node and decodes the JSON response.
func (c *client) getJSON(ctx context.Context, path string, result any) error {
	return fetchJSON(ctx, c.baseURL+path, result)
}

// hexDecode decodes a 0x-prefixed hex string.
func hexDecode(s string) ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(s, "0x"))
}

// getGenesis fetches genesis data from the beacon node REST API.
func (c *client) getGenesis(ctx context.Context) (*genesisData, error) {
	var resp structs.GetGenesisResponse
	if err := c.getJSON(ctx, "/eth/v1/beacon/genesis", &resp); err != nil {
		return nil, errors.Wrap(err, "could not get genesis")
	}
	genesisTimeSec, err := strconv.ParseUint(resp.Data.GenesisTime, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse genesis time")
	}
	valsRoot, err := hexDecode(resp.Data.GenesisValidatorsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode genesis validators root")
	}
	return &genesisData{
		GenesisTime:           time.Unix(int64(genesisTimeSec), 0),
		GenesisValidatorsRoot: bytesutil.ToBytes32(valsRoot),
	}, nil
}

// getChainHead fetches the head slot/root and finality checkpoints from the beacon node REST API.
func (c *client) getChainHead(ctx context.Context) (*chainHead, error) {
	var headerResp structs.GetBlockHeaderResponse
	if err := c.getJSON(ctx, "/eth/v1/beacon/headers/head", &headerResp); err != nil {
		return nil, errors.Wrap(err, "could not get head header")
	}
	headSlot, err := strconv.ParseUint(headerResp.Data.Header.Message.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse head slot")
	}
	headRoot, err := hexDecode(headerResp.Data.Root)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode head root")
	}

	var cpResp structs.GetFinalityCheckpointsResponse
	if err := c.getJSON(ctx, "/eth/v1/beacon/states/head/finality_checkpoints", &cpResp); err != nil {
		return nil, errors.Wrap(err, "could not get finality checkpoints")
	}
	finalizedEpoch, err := strconv.ParseUint(cpResp.Data.Finalized.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse finalized epoch")
	}
	finalizedRoot, err := hexDecode(cpResp.Data.Finalized.Root)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode finalized root")
	}

	return &chainHead{
		HeadSlot:       primitives.Slot(headSlot),
		HeadRoot:       headRoot,
		FinalizedEpoch: primitives.Epoch(finalizedEpoch),
		FinalizedRoot:  finalizedRoot,
	}, nil
}

func (c *client) retrievePeerAddressesViaRPC(ctx context.Context, beaconEndpoints []string) ([]string, error) {
	if len(beaconEndpoints) == 0 {
		return nil, errors.New("no beacon API endpoints specified")
	}
	peers := make([]string, 0)
	for _, endpoint := range beaconEndpoints {
		var resp structs.GetIdentityResponse
		if err := fetchJSON(ctx, endpoint+"/eth/v1/node/identity", &resp); err != nil {
			return nil, err
		}
		if len(resp.Data.P2PAddresses) == 0 {
			continue
		}
		// P2P addresses already include the peer ID in multiaddr format.
		peers = append(peers, resp.Data.P2PAddresses[0])
	}
	return peers, nil
}

func (c *client) initializeMockChainService(ctx context.Context) (*mockChain, error) {
	genesis, err := c.getGenesis(ctx)
	if err != nil {
		return nil, err
	}
	currEpoch := slots.ToEpoch(slots.CurrentSlot(genesis.GenesisTime))
	currFork, err := params.Fork(currEpoch)
	if err != nil {
		return nil, err
	}
	return &mockChain{
		genesisTime:     genesis.GenesisTime,
		currentFork:     currFork,
		genesisValsRoot: genesis.GenesisValidatorsRoot,
	}, nil
}

// Retrieves an external ipv4 address and converts into a libp2p formatted value.
func ipAddr() net.IP {
	ip, err := network.ExternalIP()
	if err != nil {
		panic(err) // lint:nopanic -- Only returns an error when network interfaces are not available. This is a requirement for the application anyway.
	}
	return net.ParseIP(ip)
}

// Determines a private key for p2p networking from the p2p service's
// configuration struct. If no key is found, it generates a new one.
func privKey() (*ecdsa.PrivateKey, error) {
	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	return ecdsaprysm.ConvertFromInterfacePrivKey(priv)
}

// Adds a private key to the libp2p option if the option was provided.
// If the private key file is missing or cannot be read, or if the
// private key contents cannot be marshaled, an exception is thrown.
func privKeyOption(privkey *ecdsa.PrivateKey) libp2p.Option {
	return func(cfg *libp2p.Config) error {
		ifaceKey, err := ecdsaprysm.ConvertToInterfacePrivkey(privkey)
		if err != nil {
			return err
		}
		return cfg.Apply(libp2p.Identity(ifaceKey))
	}
}

func readMetadata() (metadata.Metadata, error) {
	metaData := &pb.MetaDataV1{
		SeqNumber: 0,
		Attnets:   bitfield.NewBitvector64(),
	}
	return wrapper.WrappedMetadataV1(metaData), nil
}
