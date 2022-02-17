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
	// Max number of peers to protect
	// non-positive means infinite
	MaxProtected int
	// Target percentage of protected peers
	ProtectionRate float32

	lock sync.RWMutex
	// OrderedMap it an associative array that preserves key insertion order
	// which serves a different purpose from SortedMap or PriorityQueue
	// The performance of OrderedMap is not too worse than map + container/list solution
	// so keep using it for now to keep the code simple
	dist2protected map[int]*orderedmap.OrderedMap // OrderedMap types: map[peer.ID]time.Time
	dist2tagged    map[int]*orderedmap.OrderedMap // OrderedMap types: map[peer.ID]time.Time

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

func (p *DHTPeerProtectionPatcher) adjustProtectedThreadUnsafe() {
	for {
		minDistTagged := -1
		for d, m := range p.dist2tagged {
			if m.Len() > 0 {
				if minDistTagged < 0 || d < minDistTagged {
					minDistTagged = d
				}
			}
		}
		if minDistTagged < 0 {
			return
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

		// When max value is set and reached
		// we need to perform a swap here
		if p.isMaxProtectedReachedThreadUnsafe() {
			// Or maybe we can replace oldest protected peer with latest tagged peer
			// When distances are the same
			if minDistTagged >= maxDistProtected {
				return
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
		} else if p.getProtectionRateThreadUnsafe() < p.ProtectionRate {
			// Otherwise just move the selected peer from tagged bucket to protected bucket
			taggedBucket.Delete(bestTagged.Key)
			insertThreadUnsafe(p.dist2protected, minDistTagged, bestTaggedPeerId, bestTaggedTime)
			p.connMgr.Protect(bestTaggedPeerId, kbucketTag)
		} else {
			// TODO: should p.getProtectionRateThreadUnsafe() > p.ProtectionRate case be handled?
			// Not likely needed with current setup

			// Terminate when no operation is performed
			return
		}
	}
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

// Patch the peer protection algorithm of the given dht instance
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
