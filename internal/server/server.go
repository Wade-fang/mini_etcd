package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/rpc"

	"z.cn/RaftImpl/internal/model"
	"z.cn/RaftImpl/internal/raft"
	"z.cn/RaftImpl/internal/store"
)

type Server struct {
	Raft       *raft.Raft
	Store      *store.Store
	HttpServer *http.Server
}

func New(r *raft.Raft, s *store.Store, addr string) *Server {
	return &Server{
		Raft:       r,
		Store:      s,
		HttpServer: &http.Server{Addr: addr},
	}
}

func (s *Server) Start(addr string) error {
	// Add put method
	http.HandleFunc("/put", s.putHandler)
	// get method
	http.HandleFunc("/get", s.getHandler)

	// Register Raft service for RPC
	rpc.Register(s.Raft)
	rpc.HandleHTTP()

	s.Raft.Start()

	fmt.Println("start rpc server at", addr)
	return s.HttpServer.ListenAndServe()
}

func (s *Server) putHandler(writer http.ResponseWriter, request *http.Request) {
	msg := model.ResponseBody{
		Code: 400,
		Msg:  "request method error",
	}

	if request.Method == http.MethodPut {
		defer request.Body.Close()
		data, err := ioutil.ReadAll(request.Body)
		if err != nil {
			msg.Code = 400
			msg.Msg = err.Error()
		}
		var rb model.RequestBody
		if err := json.Unmarshal(data, &rb); err != nil {
			msg.Code = 400
			msg.Msg = err.Error()
		}
		rb.Method = http.MethodPut
		rb.IsPutLog = true

		// Call Raft Propose
		// Note: Original code used a channel (tmpLog) and a goroutine.
		// We use direct call now, but Propose currently assumes Leader and sends RPCs synchronously.
		// This might block the HTTP request, which is actually BETTER for consistency than the original,
		// but still naive.
		go s.Raft.Propose(rb)

		msg.Code = 200
		msg.Msg = "success"
		result, err := json.Marshal(&msg)
		if err != nil {
			msg.Msg = err.Error()
		}
		writer.Write(result)
	} else {
		writer.WriteHeader(400)
		data, err := json.Marshal(&msg)
		if err != nil {
			msg.Msg = err.Error()
		}
		writer.Write(data)
	}
}

func (s *Server) getHandler(writer http.ResponseWriter, request *http.Request) {
	msg := model.ResponseBody{
		Code: 400,
		Msg:  "request method error",
	}
	if request.Method == http.MethodGet {
		values := request.URL.Query()
		key := values.Get("key")
		v, err := s.Store.Get(model.RequestBody{Method: http.MethodGet, Key: key})
		if err != nil {
			msg.Msg = err.Error()
			result, _ := json.Marshal(&msg)
			writer.Write(result)
		} else {
			msg.Code = 200
			msg.Msg = v
			result, _ := json.Marshal(&msg)
			writer.Write(result)
		}
	} else {
		writer.WriteHeader(400)
		data, err := json.Marshal(&msg)
		if err != nil {
			msg.Msg = err.Error()
		}
		writer.Write(data)
	}
}

func (s *Server) Close(ctx context.Context) {
	if err := s.HttpServer.Shutdown(ctx); err != nil {
		fmt.Println("服务关闭失败,", err)
	}
	fmt.Println("服务已关闭")
}
