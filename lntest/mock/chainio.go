package mock

import (
	"github.com/bitnet-io/btcd-bitnet/chaincfg"
	"github.com/bitnet-io/btcd-bitnet/chaincfg/chainhash"
	"github.com/bitnet-io/btcd-bitnet/wire"
)

// ChainIO is a mock implementation of the BlockChainIO interface.
type ChainIO struct {
	BestHeight int32
}

// GetBestBlock currently returns dummy values.
func (c *ChainIO) GetBestBlock() (*chainhash.Hash, int32, error) {
	return chaincfg.TestNet3Params.GenesisHash, c.BestHeight, nil
}

// GetUtxo currently returns dummy values.
func (c *ChainIO) GetUtxo(op *wire.OutPoint, _ []byte,
	heightHint uint32, _ <-chan struct{}) (*wire.TxOut, error) {

	return nil, nil
}

// GetBlockHash currently returns dummy values.
func (c *ChainIO) GetBlockHash(blockHeight int64) (*chainhash.Hash, error) {
	return nil, nil
}

// GetBlock currently returns dummy values.
func (c *ChainIO) GetBlock(blockHash *chainhash.Hash) (*wire.MsgBlock, error) {
	return nil, nil
}

// GetBlockHeader currently returns dummy values.
func (c *ChainIO) GetBlockHeader(blockHash *chainhash.Hash) (*wire.BlockHeader,
	error) {

	return nil, nil
}
