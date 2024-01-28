package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/chat/types"
	"github.com/anthdm/hollywood/remote"
)

type client struct {
	username  string
	serverPID *actor.PID //save the PID of the server so we know where to send messages
	logger    *slog.Logger
}

func newClient(username string, serverPID *actor.PID) actor.Producer {
	return func() actor.Receiver {
		return &client{
			username:  username,
			serverPID: serverPID,
			logger:    slog.Default(),
		}
	}
}

func (c *client) Receive(ctx *actor.Context) {
	// switch based on the type of the message
	switch msg := ctx.Message().(type) {
	case actor.Started:
		ctx.Send(c.serverPID, &types.Connect{
			Username: c.username,
		})
	case actor.Stopped:
		_ = msg
	}
}

func main() {
	var (
		listenAt  = flag.String("listenAt", "", "specify address to listen to.  will pick random port if not specified")
		connectTo = flag.String("connect", "127.0.0.1:4000", "addr of server to conn to")
		username  = flag.String("username", os.Getenv("USER"), "")
	)
	flag.Parse()

	if *listenAt == "" {
		*listenAt = fmt.Sprintf("127.0.0.1:%d", rand.Int31n(50000)+10000)
	}

	rem := remote.New(*listenAt, remote.NewConfig())

	e, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(rem)) // actor engine handles actors locally
	if err != nil {
		slog.Error("failed to create engine", "err", err)
		os.Exit(1)
	}

	var (
		serverPID = actor.NewPID(*connectTo, "server")                                          // pass addr and identifier
		clientPID = e.Spawn(newClient(*username, serverPID), "client", actor.WithID(*username)) // spawn it and name it "client" - naming the process
		scanner   = bufio.NewScanner(os.Stdin)
	)

	for scanner.Scan() {
		msg := &types.Message{
			Msg:      scanner.Text(),
			Username: *username,
		}

		if msg.Msg == "quit" {
			break
		}
		e.SendWithSender(serverPID, msg, clientPID)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("failed to read stdin", "err", err)
	}

	e.SendWithSender(serverPID, &types.Disconnect{}, clientPID)
	e.Poison(clientPID).Wait()
	slog.Info("disconnected")
}
