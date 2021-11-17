package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p/config"
	"golang.org/x/crypto/blake2b"

	relay "github.com/libp2p/go-libp2p-circuit"

	libp2pmplex "github.com/libp2p/go-libp2p-mplex"
	mplex "github.com/libp2p/go-mplex"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	BUFFER_SIZE = 1024 * 1024
	// mainnet
	CHAIN_ID = "5f704cc0c82e0ed70e873f0893d7e06f148524e3f0bdae2afb02e7819a0c24d1"

	// mainnet seeds: https://storage.googleapis.com/mina-seed-lists/mainnet_seeds.txt
	// devnet seeds:  https://storage.googleapis.com/seed-lists/devnet_seeds.txt
	// MINA_NODE_ADDRESS = "/ip4/127.0.0.1/tcp/8302/p2p/12D3KooWKK3RpV1MWAZk3FJ5xqbVPL2BMDdUEGSfwfQoUprBNZCv"
	MINA_NODE_ADDRESS = "/ip4/95.217.106.189/tcp/8302/p2p/12D3KooWSxxCtzRLfUzoxgRYW9fTKWPUujdvStuwCPSPUN3629mb"
)

var (
	muxer = libp2p.Muxer("/coda/mplex/1.0.0", libp2pmplex.DefaultTransport)
	// https://github.com/MinaProtocol/mina/issues/9043
	// rendezvousString = fmt.Sprintf("/coda/0.0.1/%s", "971ab7c2a9370f90a3e4f5d1e711d6428e4d7ff210cbb7b5563d70843ce95274")
	rendezvousString = fmt.Sprintf("/coda/0.0.1/%s", CHAIN_ID)
	pnetKey          = blake2b.Sum256([]byte(rendezvousString))
	userAgent        = libp2p.UserAgent("github.com/codaprotocol/coda/tree/master/src/app/libp2p_helper")
	privateNetwork   = libp2p.PrivateNetwork(pnetKey[:])
)

func createRelayHost(isRelayNode bool, listenEnabled bool) (host host.Host, err error) {
	var relayOps config.Option
	if isRelayNode {
		_ = relay.OptHop
		relayOps = libp2p.EnableRelay(relay.OptHop)
	} else {
		relayOps = libp2p.EnableRelay()
	}
	var listenAddrs config.Option
	if listenEnabled {
		// listenAddr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
		listenAddrWs, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/23333/ws")
		listenAddrs = libp2p.ListenAddrs(listenAddrWs)
	} else {
		listenAddrs = libp2p.ListenAddrs()
	}
	r := rand.Reader
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, r)
	if err != nil {
		panic(err)
	}
	host, err = libp2p.New(
		context.Background(),
		libp2p.Identity(prvKey),
		listenAddrs,
		relayOps,
		muxer,
		userAgent,
		privateNetwork,
	)
	return
}

func readBytes(s network.Stream) []byte {
	data := make([]byte, BUFFER_SIZE)
	n, _ := s.Read(data)
	if n < len(data) {
		return data[:n]
	} else {
		buffer := make([]byte, BUFFER_SIZE)
		for {
			n, _ := s.Read(buffer)
			if n == 0 {
				break
			} else if n < len(buffer) {
				data = append(data, buffer[:n]...)
				break
			} else {
				data = append(data, buffer...)
			}
		}
		return data
	}
}

func run() {
	relayHost, err := createRelayHost(true, true)
	if err != nil {
		log.Printf("Failed to create relayHost: %v", err)
		return
	}
	relayHost.SetStreamHandler("/mytest", func(s network.Stream) {
		s.Write([]byte("Hello from go peer"))
		s.Close()
	})
	log.Printf("relayHost: %s %s\n", relayHost.Addrs(), relayHost.ID())
	// relayHostInfo := peer.AddrInfo{
	// 	ID:    relayHost.ID(),
	// 	Addrs: relayHost.Addrs(),
	// }

	h1, err := createRelayHost(false, false)
	if err != nil {
		log.Printf("Failed to create h1: %v", err)
		return
	}
	log.Printf("h1: %s %s\n", h1.Addrs(), h1.ID())

	mtaddr, _ := ma.NewMultiaddr(MINA_NODE_ADDRESS)
	minaPeerInfo, err := peer.AddrInfoFromP2pAddr(mtaddr)
	log.Printf("minaPeerInfo: %s %s\n", minaPeerInfo.Addrs, minaPeerInfo.ID)
	if err != nil {
		log.Println(err)
		return
	}
	{
		log.Printf("Connecting relayHost to mina: %s", mtaddr)
		relayHost.Peerstore().AddAddrs(minaPeerInfo.ID, minaPeerInfo.Addrs, peerstore.ConnectedAddrTTL)
		if err := relayHost.Connect(context.Background(), *minaPeerInfo); err != nil {
			log.Printf("Failed to connect relayHost to mina: %v", err)
			return
		}
		s, err := relayHost.NewStream(context.Background(), minaPeerInfo.ID, "/mina/node-status")
		if err != nil {
			log.Println("huh, this should have worked: ", err)
			return
		}
		time.Sleep(time.Second * 1)
		data := readBytes(s)
		// dataStr := string(data)
		fmt.Printf("[relayHost] node status received: %d bytes\n", len(data))
		// fmt.Printf("[relayHost] received:\n\n %s\n\n", dataStr)
		ioutil.WriteFile("status.json", data, 0644)
	}

	log.Printf("relayHost: %s/p2p/%s\n", relayHost.Addrs()[0], relayHost.ID())
}

func main() {
	mplex.MaxMessageSize = 1 << 30
	run()
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
