package main

import (
	"flag"
	"log/slog"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/chat/types"
	"github.com/anthdm/hollywood/remote"
)

type clientMap map[string]*actor.PID
type userMap map[string]string

type server struct {
	clients clientMap
	users   userMap
	logger  *slog.Logger
}

func newServer() actor.Receiver {
	return &server{
		clients: make(clientMap),
		users:   make(userMap),
		logger:  slog.Default(),
	}
}

func (s *server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case *types.Message:
		s.logger.Info("message received", "msg", msg.Msg, "from", ctx.Sender())
		s.handleMessage(ctx)
	case *types.Disconnect:
		cAddr := ctx.Sender().GetAddress()
		pid, ok := s.clients[cAddr]
		if !ok {
			s.logger.Warn("uknown user", "client", pid.Address)
			return
		}
		username, ok := s.users[cAddr]
		if !ok {
			s.logger.Warn("unknown user d/c", "client", pid.Address)
			return
		}
		s.logger.Info("client disconnected", "username", username)
		delete(s.clients, cAddr)
		delete(s.users, username)
	case *types.Connect:
		cAddr := ctx.Sender().GetAddress()
		if _, ok := s.clients[cAddr]; ok {
			s.logger.Warn("client already connected", "client", ctx.Sender().GetID())
			return
		}
		if _, ok := s.users[cAddr]; ok {
			s.logger.Warn("user already connected", "client", ctx.Sender().GetID())
			return
		}
		s.clients[cAddr] = ctx.Sender()
		s.users[cAddr] = msg.Username
		slog.Info("new client connected", "id", ctx.Sender().GetID(), "addr", ctx.Sender().GetAddress(), "sender", ctx.Sender(), "username", msg.Username)
	}
}

// handle the incoming msg by broadcasting to all clients
func (s *server) handleMessage(ctx *actor.Context) {
	for _, pid := range s.clients {
		if !pid.Equals(ctx.Sender()) {
			s.logger.Info("fowarding msg", "pid", pid.ID, "addr", pid.Address, "msg", ctx.Message())
			ctx.Forward(pid)
		}
	}
}

func main() {
	var listenAt = flag.String("listenAt", "127.0.0.1:4000", "")
	flag.Parse()

	rem := remote.New(*listenAt, remote.NewConfig())
	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(rem))
	if err != nil {
		panic(err)
	}

	e.Spawn(newServer, "server", actor.WithID("primary"))

	select {}
}
