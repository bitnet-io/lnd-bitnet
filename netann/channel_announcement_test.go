package netann

import (
	"bytes"
	"testing"

	"github.com/bitnet-io/btcd-bitnet/btcutil"
	"github.com/bitnet-io/btcd-bitnet/chaincfg/chainhash"
	"github.com/bitnet-io/btcd-bitnet/wire"
	"github.com/lightningnetwork/lnd/channeldb/models"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateChanAnnouncement(t *testing.T) {
	t.Parallel()

	key := [33]byte{0x1}
	var sig lnwire.Sig
	features := lnwire.NewRawFeatureVector(lnwire.AnchorsRequired)
	var featuresBuf bytes.Buffer
	if err := features.Encode(&featuresBuf); err != nil {
		t.Fatalf("unable to encode features: %v", err)
	}

	expChanAnn := &lnwire.ChannelAnnouncement{
		ChainHash:       chainhash.Hash{0x1},
		ShortChannelID:  lnwire.ShortChannelID{BlockHeight: 1},
		NodeID1:         key,
		NodeID2:         key,
		NodeSig1:        sig,
		NodeSig2:        sig,
		BitcoinKey1:     key,
		BitcoinKey2:     key,
		BitcoinSig1:     sig,
		BitcoinSig2:     sig,
		Features:        features,
		ExtraOpaqueData: []byte{0x1},
	}

	chanProof := &models.ChannelAuthProof{
		NodeSig1Bytes:    expChanAnn.NodeSig1.ToSignatureBytes(),
		NodeSig2Bytes:    expChanAnn.NodeSig2.ToSignatureBytes(),
		BitcoinSig1Bytes: expChanAnn.BitcoinSig1.ToSignatureBytes(),
		BitcoinSig2Bytes: expChanAnn.BitcoinSig2.ToSignatureBytes(),
	}
	chanInfo := &models.ChannelEdgeInfo{
		ChainHash:        expChanAnn.ChainHash,
		ChannelID:        expChanAnn.ShortChannelID.ToUint64(),
		ChannelPoint:     wire.OutPoint{Index: 1},
		Capacity:         btcutil.SatoshiPerBitcoin,
		NodeKey1Bytes:    key,
		NodeKey2Bytes:    key,
		BitcoinKey1Bytes: key,
		BitcoinKey2Bytes: key,
		Features:         featuresBuf.Bytes(),
		ExtraOpaqueData:  expChanAnn.ExtraOpaqueData,
	}
	chanAnn, _, _, err := CreateChanAnnouncement(
		chanProof, chanInfo, nil, nil,
	)
	require.NoError(t, err, "unable to create channel announcement")

	assert.Equal(t, chanAnn, expChanAnn)
}
