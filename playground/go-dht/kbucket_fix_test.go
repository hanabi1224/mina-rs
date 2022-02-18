package kbucketfix

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

func TestPatcher1(t *testing.T) {
	testPatcher(t, 0.5, 0)
}

func TestPatcher2(t *testing.T) {
	testPatcher(t, 0.3, 0)
}

func TestPatcher3(t *testing.T) {
	testPatcher(t, 0.8, 0)
}

func TestPatcher4(t *testing.T) {
	testPatcher(t, 0.5, 10)
}

func testPatcher(t *testing.T, targetProtectionRate float32, maxProtected int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, hostDHT := makeHost(t, ctx)

	patcher := NewPatcher()
	patcher.ProtectionRate = targetProtectionRate
	patcher.MaxProtected = maxProtected
	patcher.Patch(hostDHT)

	rt := hostDHT.RoutingTable()
	if rt == nil {
		t.Error("No routing table found")
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
	}

	host2, _ := makeHost(t, ctx)
	connect(host, host2, ctx)
	for i := 0; i < 3000; i += 1 {
		peerHost, _ := makeHost(t, ctx)
		// connect(host2, peerHost, ctx)
		connect(host, peerHost, ctx)
	}

	hostDHT.RefreshRoutingTable()
	log.Println("Done refreshing routing table")
	time.Sleep(time.Second * 2)
	rt.PeerAdded = func(p peer.ID) { log.Println("PeerAdded: " + p.String()) }
	rt.PeerRemoved = func(p peer.ID) { log.Println("PeerRemoved: " + p.String()) }

	// Ensure numbers of active peers matches between the patcher and connectiion manager
	if added-removed != patcher.getProtectedLenThreadUnsafe()+patcher.getTaggedLenThreadUnsafe() {
		t.Error(fmt.Sprintf("%d - %d != %d + %d", added, removed, patcher.getProtectedLenThreadUnsafe(), patcher.getTaggedLenThreadUnsafe()))
	}

	percentage := patcher.getProtectionRateThreadUnsafe()
	if maxProtected > 0 {
		// Ensure MaxProtected (peers) restriction works as expected
		if patcher.getProtectedLenThreadUnsafe() > maxProtected || percentage > targetProtectionRate {
			t.Error(fmt.Sprintf("%d - %f", patcher.getProtectedLenThreadUnsafe(), percentage))
		}
	} else {
		// Ensure the actual peer protection rate approximately matches the target
		// when MaxProtected (peers) is not specified
		const BIAS float32 = .05
		if percentage < targetProtectionRate-BIAS || percentage > targetProtectionRate+BIAS {
			t.Error(percentage)
		}
	}

	// Ensure all peers that are marked as protected in the patcher
	// are actually protected in the connection manager
	for _, m := range patcher.dist2protected {
		for _, k := range m.Keys() {
			pid := k.(peer.ID)
			if !connMgr.IsProtected(pid, kbucketTag) {
				t.Error(fmt.Sprintf("Peer %s should be protected", pid))
			}
		}
	}

	// Ensure all peers that are marked as tagged in the patcher
	// are actually not protected in the connection manager
	for _, m := range patcher.dist2tagged {
		for _, k := range m.Keys() {
			pid := k.(peer.ID)
			if connMgr.IsProtected(pid, kbucketTag) {
				t.Error(fmt.Sprintf("Peer %s should not be protected", pid))
			}
		}
	}
}
