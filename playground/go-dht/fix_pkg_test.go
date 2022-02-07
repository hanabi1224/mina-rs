package kbucketfix

import (
	"context"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
)

func TestFixPackage1(t *testing.T) {
	testFixPackage(t, 0.5, 0)
}

func TestFixPackage2(t *testing.T) {
	testFixPackage(t, 0.3, 0)
}

func TestFixPackage3(t *testing.T) {
	testFixPackage(t, 0.8, 0)
}

func TestFixPackage4(t *testing.T) {
	testFixPackage(t, 0.5, 10)
}

func testFixPackage(t *testing.T, targetProtectionRate float32, maxProtected int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, hostDHT := makeHost(t, ctx)

	patcher := NewPatcher()
	patcher.ProtectionRate = targetProtectionRate
	patcher.MaxProtected = maxProtected
	patcher.Patch(hostDHT)

	rt := hostDHT.RoutingTable()
	if rt == nil {
		t.Error()
	}
	connMgr := host.ConnManager()
	added := 0
	removed := 0
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
		removed += 1
		// log.Println("PeerRemoved: " + p.String())
	}

	host2, _ := makeHost(t, ctx)
	connect(host, host2, ctx)
	for i := 0; i < 2000; i += 1 {
		peerHost, _ := makeHost(t, ctx)
		connect(host, peerHost, ctx)
	}

	hostDHT.RefreshRoutingTable()

	if added-removed != patcher.getProtectedLenThreadUnsafe()+patcher.getTaggedLenThreadUnsafe() {
		t.Error()
	}

	percentage := patcher.getProtectionRateThreadUnsafe()
	if maxProtected > 0 {
		if patcher.getProtectedLenThreadUnsafe() > maxProtected || percentage > targetProtectionRate {
			t.Error(fmt.Sprintf("%d - %f", patcher.getProtectedLenThreadUnsafe(), percentage))
		}
	} else {
		const BIAS float32 = .03
		if percentage < targetProtectionRate-BIAS || percentage > targetProtectionRate+BIAS {
			t.Error(percentage)
		}
	}
}
