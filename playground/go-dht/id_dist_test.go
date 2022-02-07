package kbucketfix

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

func TestIdDistribution(t *testing.T) {
	m := make(map[int]int)
	a := GenId()
	total := 0
	for i := 0; i < 3000; i++ {
		b := GenId()
		m[kb.CommonPrefixLen(a, b)] += 1
		total += 1
	}

	percentage := float32(m[0]+m[1]) / float32(total)
	const TARGET float32 = .75
	const BIAS float32 = .03
	if percentage < TARGET-BIAS || percentage > TARGET+BIAS {
		t.Error(percentage)
	}
}

func GenId() kb.ID {
	privkey, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	id, _ := peer.IDFromPrivateKey(privkey)
	return kb.ConvertPeerID(id)
}
