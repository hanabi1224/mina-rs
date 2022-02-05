package kbucketfix

import (
	"context"
	"testing"

	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/stretchr/testify/require"
)

var (
	CONTEXT context.Context
	DHT     *kaddht.IpfsDHT
)

func TestMinimalRepro(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	CONTEXT = ctx

	host, hostDHT := makeHost(t)
	rt := hostDHT.RoutingTable()
	if rt == nil {
		t.Error()
	}
	connMgr := host.ConnManager()

	added := 0
	protected := 0
	peerAdded := rt.PeerAdded
	rt.PeerAdded = func(p peer.ID) {
		peerAdded(p)
		added += 1
		if connMgr.IsProtected(p, "") {
			protected += 1
		}
	}

	peerRemoved := rt.PeerRemoved
	rt.PeerRemoved = func(p peer.ID) {
		peerRemoved(p)
		// log.Println("PeerRemoved: " + p.String())
	}

	host2, _ := makeHost(t)
	connect(host, host2)
	for i := 0; i < 1000; i += 1 {
		peerHost, _ := makeHost(t)
		connect(host, peerHost)
	}

	hostDHT.RefreshRoutingTable()

	percentage := float64(protected) / float64(added)
	const TARGET float64 = .75
	const BIAS float64 = .03
	if percentage < TARGET-BIAS || percentage > TARGET+BIAS {
		t.Error(percentage)
	}
}

func makeHost(t *testing.T) (*bhost.BasicHost, *kaddht.IpfsDHT) {
	connMgr, _ := connmgr.NewConnManager(10, 100)
	dhtOpts := []kaddht.Option{
		kaddht.DisableAutoRefresh(),
		kaddht.Mode(kaddht.ModeServer),
	}
	hostOpt := new(bhost.HostOpts)
	hostOpt.ConnManager = connMgr
	host, err := bhost.NewHost(swarmt.GenSwarm(t, swarmt.OptDisableReuseport), hostOpt)
	require.NoError(t, err)
	hostDHT, err := kaddht.New(CONTEXT, host, dhtOpts...)
	require.NoError(t, err)
	return host, hostDHT
}

func connect(a, b *bhost.BasicHost) {
	hi := peer.AddrInfo{ID: b.ID(), Addrs: b.Addrs()}
	a.Peerstore().AddAddrs(hi.ID, hi.Addrs, peerstore.PermanentAddrTTL)
	a.Connect(CONTEXT, hi)
}
