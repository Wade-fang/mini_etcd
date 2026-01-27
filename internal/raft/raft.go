package raft

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"z.cn/RaftImpl/internal/model"
	"z.cn/RaftImpl/internal/store"
	"z.cn/RaftImpl/internal/transport"
)

type Raft struct {
	Me        model.Node
	peers     map[string]model.Node
	store     *store.Store
	transport *transport.Transport

	heartTimeout  chan time.Time
	lastHeartTime time.Time
	mu            sync.RWMutex
}

func New(name, addr string, peers map[string]model.Node, store *store.Store, trans *transport.Transport) *Raft {
	node := model.Node{
		Name:        name,
		Address:     addr,
		Role:        model.Follower,
		State:       0,
		Time:        0,
		CanvassNum:  0,
		CanvassFlag: true,
	}
	r := &Raft{
		Me:           node,
		peers:        peers,
		store:        store,
		transport:    trans,
		heartTimeout: make(chan time.Time, 1),
	}
	return r
}

func (r *Raft) Start() {
	// Connect to peers
	for _, p := range r.peers {
		go r.transport.Connect(p)
	}

	// Start tasks
	go r.listenerTimeOut()
	go r.heartTask()
	go r.canvassTask()

	// Load initial state
	r.store.ReadLogCommand(0)
}

// --- Tasks ---

func (r *Raft) canvassTask() {
	for {
		select {
		case <-time.Tick(r.randomTimeOut(model.Canvass)):
			if r.Me.Role != model.Leader {
				r.requestCanvass()
			}
		}
	}
}

func (r *Raft) heartTask() {
	for {
		select {
		case <-time.Tick(r.randomTimeOut(model.Heart)):
			r.sendHeart()
		}
	}
}

func (r *Raft) listenerTimeOut() {
	for {
		select {
		case lastTime := <-r.heartTimeout:
			r.lastHeartTime = lastTime
			if r.Me.Role == model.Follower && time.Since(lastTime) > r.randomTimeOut(model.Candidate) {
				r.Me.Role = model.Candidate
			}
		default:
			if r.Me.Role == model.Follower && time.Since(r.lastHeartTime) > r.randomTimeOut(model.Candidate) {
				r.Me.Role = model.Candidate
			}
			time.Sleep(time.Millisecond * 300)
		}
	}
}

// --- Logic ---

func (r *Raft) requestCanvass() {
	// Simplified check: if no peers, become leader (for single node test?)
	// But transport pool might be empty if connections failed.
	// Original code checked transport pool size. I will check logic peers size.
	// But actually original code checked `len(r.pool)`.

	// If I'm candidate
	if r.Me.Role == model.Candidate {
		for _, peer := range r.peers {
			// Try to send
			fmt.Printf("Candidate %s request Canvass from %s\n", r.Me.Name, peer.Name)
			res := model.CommandMsg{}
			req := model.CommandMsg{Command: model.Canvass, Node: r.Me}

			err := r.transport.Call(peer.Name, "Raft.Canvass", req, &res)
			if err != nil {
				fmt.Println("failed Canvass:", err)
				r.transport.ReportFailure(peer)
				continue
			}

			if res.CanvassFlag {
				r.Me.CanvassNum++
			}
			fmt.Println(res.Node.Name, " give ", r.Me.Name, res.CanvassFlag, ", count:", r.Me.CanvassNum)

			// Check if won
			activePeers := len(r.peers) // Approximate
			if r.Me.CanvassNum > 0 && r.Me.CanvassNum >= activePeers/2 {
				r.Me.Leader = r.Me.Name
				r.Me.Role = model.Leader
				fmt.Println(">1/2 leader")
				r.Me.Time++
			}
		}
	}
}

func (r *Raft) sendHeart() {
	if r.Me.Leader == r.Me.Name {
		for _, peer := range r.peers {
			res := &model.CommandMsg{} // NewCommandMsg(Heart, "", node, false) logic
			req := model.CommandMsg{Command: model.Heart, Node: r.Me}

			err := r.transport.Call(peer.Name, "Raft.Heart", req, res)
			if err != nil {
				fmt.Println("failed Heart:", err)
				r.transport.ReportFailure(peer)
				continue
			}
			fmt.Printf("%s leader %s send heart to %s \n", time.Now().Format("2009-01-02 03:04:05"), r.Me.Name, peer.Name)
		}
	}
}

// Propose a new log entry (from client)
func (r *Raft) Propose(body model.RequestBody) error {
	// Original logic: "sendLogReplication"
	// Only leader does this
	if r.Me.Leader == r.Me.Name {
		// Local apply
		// Wait, original logic:
		// if single node -> store.Resolve
		// else -> send to others -> store.Resolve

		// Replicate
		for _, peer := range r.peers {
			res := &model.CommandMsg{}
			req := model.CommandMsg{Command: model.LogReplication, LogCommand: body, Node: r.Me}

			err := r.transport.Call(peer.Name, "Raft.LogReplication", req, res)
			if err != nil {
				fmt.Println("failed LogReplication:", err)
				r.transport.ReportFailure(peer)
				return err
			}
		}
		// If we are here, we optimistically assume success (per original bad design)
		r.store.Resolve(body)
		return nil
	}
	return fmt.Errorf("not leader")
}

// --- RPC Handlers ---

func (r *Raft) Heart(req model.CommandMsg, res *model.CommandMsg) error {
	fmt.Println(time.Now().Format("2009-01-02 03:04:05"), "Received Heart...")
	r.update(req.Node)
	r.Me.Leader = req.Node.Name
	r.Me.Role = model.Follower

	if r.Me.Time == req.Node.Time {
		res.Node = r.Me
	} else if r.Me.Time < req.Node.Time {
		r.Me.Time = req.Node.Time
	}

	// Reset timeout
	select {
	case r.heartTimeout <- time.Now():
	default:
	}
	return nil
}

func (r *Raft) Canvass(req model.CommandMsg, res *model.CommandMsg) error {
	res.Command = model.Canvass
	if req.Node.Time < r.Me.Time {
		res.CanvassFlag = false
		return nil
	}
	res.CanvassFlag = r.Me.CanvassFlag
	res.Node = r.Me

	if r.Me.CanvassFlag {
		r.Me.CanvassFlag = !r.Me.CanvassFlag
	}
	return nil
}

func (r *Raft) LogReplication(req model.CommandMsg, res *model.CommandMsg) error {
	res.Command = model.LogReplication
	log := req.LogCommand

	value, err := r.store.Resolve(log)
	if err != nil {
		res.Msg = value
		res.Err = err // Error to string? No, keeping error interface
		return nil
	}
	fmt.Println(r.Me.Name, "Received LogCommand from", req.Node.Name, ":", log)
	return nil
}

// --- Helpers ---

func (r *Raft) update(node model.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Update peer info if we had dynamic peers, but here we just have static map
	// Maybe update their status?
	if p, ok := r.peers[node.Name]; ok {
		p.Role = node.Role
		p.State = node.State
		p.Leader = node.Leader
		r.peers[node.Name] = p

		r.Me.CanvassNum = 0
		r.Me.CanvassFlag = true
	}
}

func (r *Raft) randomTimeOut(command int) time.Duration {
	switch command {
	case model.Candidate:
		return 2 * time.Second
	case model.Heart:
		return time.Duration(randInt(100, 150)) * time.Millisecond
	default:
		return time.Duration(randInt(1500, 3000)) * time.Millisecond
	}
}

func randInt(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return rand.Int63n(max-min) + min
}
