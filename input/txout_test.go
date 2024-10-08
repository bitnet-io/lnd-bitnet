package input

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/bitnet-io/btcd-bitnet/wire"
)

func TestTxOutSerialization(t *testing.T) {
	txo := wire.TxOut{
		Value: 1e7,
		PkScript: []byte{
			0x41, // OP_DATA_65
			0x04, 0xd6, 0x4b, 0xdf, 0xd0, 0x9e, 0xb1, 0xc5,
			0xfe, 0x29, 0x5a, 0xbd, 0xeb, 0x1d, 0xca, 0x42,
			0x81, 0xbe, 0x98, 0x8e, 0x2d, 0xa0, 0xb6, 0xc1,
			0xc6, 0xa5, 0x9d, 0xc2, 0x26, 0xc2, 0x86, 0x24,
			0xe1, 0x81, 0x75, 0xe8, 0x51, 0xc9, 0x6b, 0x97,
			0x3d, 0x81, 0xb0, 0x1c, 0xc3, 0x1f, 0x04, 0x78,
			0x34, 0xbc, 0x06, 0xd6, 0xd6, 0xed, 0xf6, 0x20,
			0xd1, 0x84, 0x24, 0x1a, 0x6a, 0xed, 0x8b, 0x63,
			0xa6, // 65-byte signature
			0xac, // OP_CHECKSIG
		},
	}

	var buf bytes.Buffer

	if err := writeTxOut(&buf, &txo); err != nil {
		t.Fatalf("unable to serialize txout: %v", err)
	}

	var deserializedTxo wire.TxOut
	if err := readTxOut(&buf, &deserializedTxo); err != nil {
		t.Fatalf("unable to deserialize txout: %v", err)
	}

	if !reflect.DeepEqual(txo, deserializedTxo) {
		t.Fatalf("original and deserialized txouts are different:\n"+
			"original     : %+v\n"+
			"deserialized : %+v\n",
			txo, deserializedTxo)
	}
}
