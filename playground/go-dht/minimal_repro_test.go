package kbucketfix

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
)

func TestMinimalRepro(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, hostDHT := makeHost(t, ctx)
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

	host2, _ := makeHost(t, ctx)
	connect(host, host2, ctx)
	for i := 0; i < 3000; i += 1 {
		peerHost, _ := makeHost(t, ctx)
		// connect(host2, peerHost, ctx)
		connect(host, peerHost, ctx)
	}

	hostDHT.RefreshRoutingTable()

	percentage := float32(protected) / float32(added)
	const TARGET float32 = .75
	const BIAS float32 = .03
	if percentage < TARGET-BIAS || percentage > TARGET+BIAS {
		t.Error(percentage)
	}
}
