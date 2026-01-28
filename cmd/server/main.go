package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"z.cn/RaftImpl/internal/config"
	"z.cn/RaftImpl/internal/model"
	"z.cn/RaftImpl/internal/raft"
	"z.cn/RaftImpl/internal/server"
	"z.cn/RaftImpl/internal/store"
	"z.cn/RaftImpl/internal/transport"
)

func main() {
	options := os.Args[1:]
	configNum, err := strconv.Atoi(options[0])
	if err != nil {
		fmt.Println("failed to parse cmd option, err", err)
	}
	var configString string
	switch configNum {
	case 1:
		fmt.Println("node1 starting")
		configString = "./conf/node1/server.ini"
	case 2:
		fmt.Println("node2 starting")
		configString = "./conf/node2/server.ini"
	case 3:
		fmt.Println("node3 starting")
		configString = "./conf/node3/server.ini"
	}
	// 1. Config
	name, addr, err := config.GetCurrentConfig(configString)
	if err != nil {
		fmt.Println("failed open server.ini ,err", err)
		return
	}
	pNames, pAddrs, err := config.GetClusterConfig(configString)
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
	srv := server.New(r, s, addr)

	//优雅关闭http服务
	go func() {
		// Start Server (blocks)
		err = srv.Start(addr)
		if err != nil {
			fmt.Println("server start err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*1)
	defer cancelFunc()
	srv.Close(ctx)
}
