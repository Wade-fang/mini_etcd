package main

import (
	"fmt"
	"z.cn/RaftImpl/internal/config"
	"z.cn/RaftImpl/internal/model"
	"z.cn/RaftImpl/internal/raft"
	"z.cn/RaftImpl/internal/server"
	"z.cn/RaftImpl/internal/store"
	"z.cn/RaftImpl/internal/transport"
)

func main() {
	// 1. Config
	name, addr, err := config.GetCurrentConfig()
	if err != nil {
		fmt.Println("failed open server.ini ,err", err)
		return
	}
	pNames, pAddrs, err := config.GetClusterConfig()
	if err != nil {
		fmt.Println("failed open server.ini ,err", err)
		return
	}

	// 2. Build Peers Map
	peers := make(map[string]model.Node)
	if len(pNames) == len(pAddrs) {
		for i, n := range pNames {
			peers[n] = model.Node{
				Name:    n,
				Address: pAddrs[i],
				Role:    model.Follower,
			}
		}
	} else {
		fmt.Println("cluster address and name not equal")
		return
	}

	// 3. Store
	s, err := store.NewStore(name + ".log")
	if err != nil {
		fmt.Println("store init err", err)
		return
	}

	// 4. Transport
	trans := transport.New()

	// 5. Raft
	r := raft.New(name, addr, peers, s, trans)

	// 6. Server
	srv := server.New(r, s)

	// 7. Start
	r.Start()

	// Start Server (blocks)
	err = srv.Start(addr)
	if err != nil {
		fmt.Println("server start err", err)
	}
}
