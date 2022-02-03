package main

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

func main() {
	m := make(map[int]int)
	a := GenId()
	for i := 0; i < 1000; i++ {
		b := GenId()
		m[kb.CommonPrefixLen(a, b)] += 1
	}
	println(ToJsonString(m))
}

func GenId() kb.ID {
	privkey, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	id, _ := peer.IDFromPrivateKey(privkey)
	return kb.ConvertPeerID(id)
}

func ToJsonString(i interface{}) string {
	bytes, _ := json.Marshal(i)
	return string(bytes)
}
