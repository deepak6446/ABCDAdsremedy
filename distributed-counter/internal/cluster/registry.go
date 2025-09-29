package cluster

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	heartbeatInterval = 1 * time.Second
	peerExpiryTimeout = 15 * time.Second
)

// HTTPClient defines the interface our registry needs for communication.
type HTTPClient interface {
	Post(ctx context.Context, url string, body interface{}, responseBody interface{}) error
}

// Peer represents a node in the cluster.
type Peer struct {
	ID       string    `json:"id"` // host:port
	Addr     string    `json:"addr"`
	LastSeen time.Time `json:"-"`
}

// Registry manages the list of peers in the cluster.
type Registry struct {
	mu         sync.RWMutex
	selfID     string
	peers      map[string]Peer
	httpClient HTTPClient // <-- DEPEND ON THE INTERFACE
}

// NewRegistry creates a new registry.
func NewRegistry(selfID string, client HTTPClient) *Registry {
	return &Registry{
		selfID:     selfID,
		peers:      make(map[string]Peer),
		httpClient: client,
	}
}

// Start begins the background tasks for announcing, heartbeating, and peer management.
func (r *Registry) Start(initialPeers []string) {
	// Add self to the peer list
	r.addPeer(Peer{ID: r.selfID, Addr: r.selfID, LastSeen: time.Now()})

	// Announce to initial peers
	go r.announce(initialPeers)

	// Start periodic health checks and heartbeats
	go r.periodicHealthCheck()
}

// GetPeerAddrs returns a list of all known peer addresses, excluding self.
func (r *Registry) GetPeerAddrs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var addrs []string
	for id, peer := range r.peers {
		if id != r.selfID {
			addrs = append(addrs, peer.Addr)
		}
	}
	return addrs
}

// HandleJoinRequest is called when a new node wants to join the cluster.
func (r *Registry) HandleJoinRequest(peerID string) []Peer {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.Printf("Node %s is joining the cluster", peerID)
	r.peers[peerID] = Peer{ID: peerID, Addr: peerID, LastSeen: time.Now()}

	var peerList []Peer
	for _, p := range r.peers {
		peerList = append(peerList, p)
	}
	return peerList
}

// HandleHeartbeat updates the last seen time for a peer.
func (r *Registry) HandleHeartbeat(peerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if peer, exists := r.peers[peerID]; exists {
		peer.LastSeen = time.Now()
		r.peers[peerID] = peer
	} else {
		// If we get a heartbeat from an unknown peer, add them.
		log.Printf("Received heartbeat from unknown peer %s, adding to list", peerID)
		r.peers[peerID] = Peer{ID: peerID, Addr: peerID, LastSeen: time.Now()}
	}
}

func (r *Registry) announce(initialPeers []string) {
	for _, peerAddr := range initialPeers {
		if peerAddr == r.selfID {
			continue
		}
		url := "http://" + peerAddr + "/cluster/join"
		body := map[string]string{"id": r.selfID}
		var responsePeers []Peer

		log.Printf("Announcing self to peer %s", peerAddr)
		err := r.httpClient.Post(context.Background(), url, body, &responsePeers)
		if err != nil {
			log.Printf("Failed to announce to peer %s: %v", peerAddr, err)
			continue
		}
		log.Printf("Successfully announced to %s, received %d peers", peerAddr, len(responsePeers))
		r.syncPeers(responsePeers)
	}
}

func (r *Registry) periodicHealthCheck() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.sendHeartbeats()
		r.removeExpiredPeers()
	}
}

func (r *Registry) sendHeartbeats() {
	for _, peerAddr := range r.GetPeerAddrs() {
		go func(addr string) {
			url := "http://" + addr + "/cluster/heartbeat"
			body := map[string]string{"id": r.selfID}
			err := r.httpClient.Post(context.Background(), url, body, nil)
			if err != nil {
				log.Printf("Failed to send heartbeat to %s: %v", addr, err)
			}
			log.Printf("sendHeartbeats to %s", addr)
		}(peerAddr)
	}
}

func (r *Registry) removeExpiredPeers() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, peer := range r.peers {
		if id == r.selfID {
			continue
		}
		if time.Since(peer.LastSeen) > peerExpiryTimeout {
			log.Printf("Peer %s expired, removing from list", id)
			delete(r.peers, id)
		}
	}
}

func (r *Registry) addPeer(p Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peers[p.ID] = p
}

func (r *Registry) syncPeers(newPeers []Peer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, peer := range newPeers {
		if _, exists := r.peers[peer.ID]; !exists {
			log.Printf("Discovered new peer %s from sync", peer.ID)
			r.peers[peer.ID] = Peer{ID: peer.ID, Addr: peer.Addr, LastSeen: time.Now()}
		}
	}
}