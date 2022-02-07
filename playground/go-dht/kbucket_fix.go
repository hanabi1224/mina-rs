package kbucketfix

import (
	"sync"
	"time"

	"github.com/elliotchance/orderedmap"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

const (
	kbucketTag       = "kbucket"
	protectedBuckets = 2
	// BaseConnMgrScore is the base of the score set on the connection
	// manager "kbucket" tag. It is added with the common prefix length
	// between two peer IDs.
	baseConnMgrScore = 5
)

type DHTPeerProtectionPatcher struct {
	MaxProtected   int
	ProtectionRate float32

	lock           sync.RWMutex
	dist2protected map[int]*orderedmap.OrderedMap
	dist2tagged    map[int]*orderedmap.OrderedMap //id to distance: map[peer.ID]int

	dht          *kaddht.IpfsDHT
	host         host.Host
	connMgr      connmgr.ConnManager
	selfId       kb.ID
	routingTable *kb.RoutingTable
}

func (p *DHTPeerProtectionPatcher) getProtectedLenThreadUnsafe() int {
	length := 0
	for _, m := range p.dist2protected {
		length += m.Len()
	}
	return length
}

func (p *DHTPeerProtectionPatcher) getTaggedLenThreadUnsafe() int {
	length := 0
	for _, m := range p.dist2tagged {
		length += m.Len()
	}
	return length
}

func (p *DHTPeerProtectionPatcher) isMaxProtectedReachedThreadUnsafe() bool {
	if p.MaxProtected <= 0 {
		return false
	}
	return p.getProtectedLenThreadUnsafe() >= p.MaxProtected
}

// func (p *DHTPeerProtectionPatcher) getProtectionRate() float32 {
// 	p.lock.RLock()
// 	defer p.lock.RUnlock()
// 	return p.getProtectionRateThreadUnsafe()
// }

func (p *DHTPeerProtectionPatcher) getProtectionRateThreadUnsafe() float32 {
	protectedLen := p.getProtectedLenThreadUnsafe()
	taggedLen := p.getTaggedLenThreadUnsafe()
	return float32(protectedLen) / float32(protectedLen+taggedLen)
}

func (p *DHTPeerProtectionPatcher) adjustProtectedThreadUnsafe() bool {
	minDistTagged := -1
	for d, m := range p.dist2tagged {
		if m.Len() > 0 {
			if minDistTagged < 0 || d < minDistTagged {
				minDistTagged = d
			}
		}
	}
	if minDistTagged < 0 {
		return false
	}
	maxDistProtected := -1
	for d, m := range p.dist2protected {
		if m.Len() > 0 {
			if maxDistProtected < 0 || d > maxDistProtected {
				maxDistProtected = d
			}
		}
	}

	taggedBucket := p.dist2tagged[minDistTagged]
	bestTagged := taggedBucket.Back()
	bestTaggedPeerId := bestTagged.Key.(peer.ID)
	bestTaggedTime := bestTagged.Value.(time.Time)

	if p.isMaxProtectedReachedThreadUnsafe() {
		if minDistTagged >= maxDistProtected {
			return false
		}

		protectedBucket := p.dist2protected[maxDistProtected]
		worstProtected := protectedBucket.Front()
		worstProtectedPeerId := worstProtected.Key.(peer.ID)
		worstProtectedTime := worstProtected.Value.(time.Time)
		// Swap
		taggedBucket.Delete(bestTagged.Key)
		protectedBucket.Delete(worstProtected.Key)
		insertThreadUnsafe(p.dist2tagged, maxDistProtected, worstProtectedPeerId, worstProtectedTime)
		insertThreadUnsafe(p.dist2protected, minDistTagged, bestTaggedPeerId, bestTaggedTime)
		p.connMgr.Unprotect(worstProtectedPeerId, kbucketTag)
		p.connMgr.TagPeer(worstProtectedPeerId, kbucketTag, baseConnMgrScore)
		p.connMgr.Protect(bestTaggedPeerId, kbucketTag)
		return p.adjustProtectedThreadUnsafe()
	} else if p.getProtectionRateThreadUnsafe() < p.ProtectionRate {
		taggedBucket.Delete(bestTagged.Key)
		insertThreadUnsafe(p.dist2protected, minDistTagged, bestTaggedPeerId, bestTaggedTime)
		p.connMgr.Protect(bestTaggedPeerId, kbucketTag)
		return p.adjustProtectedThreadUnsafe()
	}
	return false
}

func NewPatcher() DHTPeerProtectionPatcher {
	return DHTPeerProtectionPatcher{
		MaxProtected:   0,
		ProtectionRate: .5,
		dist2protected: make(map[int]*orderedmap.OrderedMap),
		dist2tagged:    make(map[int]*orderedmap.OrderedMap),
	}
}

func (p *DHTPeerProtectionPatcher) Heartbeat(peerId peer.ID) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	updated := false
	for _, protected := range p.dist2protected {
		if protected.Delete(peerId) {
			protected.Set(peerId, time.Now())
			updated = true
			break
		}
	}
	if !updated {
		for _, tagged := range p.dist2tagged {
			if tagged.Delete(peerId) {
				tagged.Set(peerId, time.Now())
				updated = true
				break
			}
		}
	}
	return updated
}

func (p *DHTPeerProtectionPatcher) Patch(dht *kaddht.IpfsDHT) {
	p.dht = dht
	p.host = dht.Host()
	p.connMgr = p.host.ConnManager()
	p.selfId = kb.ConvertPeerID(dht.PeerID())
	p.routingTable = dht.RoutingTable()

	p.routingTable.PeerAdded = func(pid peer.ID) {
		p.connMgr.TagPeer(pid, kbucketTag, baseConnMgrScore)
		commonPrefixLen := kb.CommonPrefixLen(p.selfId, kb.ConvertPeerID(pid))
		p.lock.Lock()
		defer p.lock.Unlock()
		// TODO: Logic here can be more efficient
		insertThreadUnsafe(p.dist2tagged, commonPrefixLen, pid, time.UnixMicro(0))
		p.adjustProtectedThreadUnsafe()
	}

	peerRemoved := p.routingTable.PeerRemoved
	p.routingTable.PeerRemoved = func(pid peer.ID) {
		peerRemoved(pid)
		p.lock.Lock()
		defer p.lock.Unlock()
		deleted := false
		for _, protected := range p.dist2protected {
			if protected.Delete(pid) {
				deleted = true
				break
			}
		}
		if !deleted {
			for _, tagged := range p.dist2tagged {
				if tagged.Delete(pid) {
					break
				}
			}
		}
		p.adjustProtectedThreadUnsafe()
	}
}

func insertThreadUnsafe(m map[int]*orderedmap.OrderedMap, distance int, id peer.ID, t time.Time) {
	om, ok := m[distance]
	if !ok {
		om = orderedmap.NewOrderedMap()
		m[distance] = om
	}
	om.Set(id, t)
}
