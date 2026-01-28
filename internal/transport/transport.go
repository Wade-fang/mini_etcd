package transport

import (
	"fmt"
	"net/rpc"
	"sync"
	"time"

	"z.cn/RaftImpl/internal/model"
)

type Transport struct {
	pool       map[string]*rpc.Client
	mu         sync.RWMutex
	failedNode chan model.Node
}

func New() *Transport {
	t := &Transport{
		pool:       make(map[string]*rpc.Client),
		failedNode: make(chan model.Node, 10),
	}
	go t.retryFailedNodes()
	return t
}

func (t *Transport) Connect(node model.Node) {
	client, err := rpc.DialHTTP("tcp", node.Address)
	if err != nil {
		t.failedNode <- node
	} else {
		t.mu.Lock()
		t.pool[node.Name] = client
		t.mu.Unlock()
		fmt.Println("Connected to", node.Address)
	}
}

func (t *Transport) Disconnect(nodeName string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.pool, nodeName)
}

func (t *Transport) Call(nodeName string, method string, args interface{}, reply interface{}) error {
	t.mu.RLock()
	client, ok := t.pool[nodeName]
	t.mu.RUnlock()

	if !ok {
		return fmt.Errorf("node %s not connected", nodeName)
	}
	return client.Call(method, args, reply)
}

func (t *Transport) retryFailedNodes() {
	for {
		select {
		case node := <-t.failedNode:
			time.Sleep(500 * time.Millisecond)
			fmt.Println("trying to reconnect:", node.Name)
			t.Connect(node)
		}
	}
}

// Helper to add a node to retry queue manually if a call fails
func (t *Transport) ReportFailure(node model.Node) {
	t.Disconnect(node.Name)
	t.failedNode <- node
}
