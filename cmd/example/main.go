package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/co-in/network"
)

func startNode(ctx context.Context, port uint16) *network.Gossip {
	g, err := network.NewGossip(
		network.WithPort(port),
	)
	if err != nil {
		panic(err)
	}

	if err = g.Subscribe(ctx); err != nil {
		panic(err)
	}

	return g
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var n [4]*network.Gossip

	for i := 0; i < len(n); i++ {
		n[uint16(i)] = startNode(ctx, uint16(9000+i))
	}

	/*
		9000=>9001
		9000=>9002

		9001=>9000
		9001=>9002

		9002=>9003
		9003=>9001
	*/

	if err := n[0].Connect(ctx, fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", n[1].Port(), n[1].ID())); err != nil {
		panic(err)
	}

	if err := n[0].Connect(ctx, fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", n[2].Port(), n[2].ID())); err != nil {
		panic(err)
	}

	if err := n[1].Connect(ctx, fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", n[0].Port(), n[0].ID())); err != nil {
		panic(err)
	}

	if err := n[1].Connect(ctx, fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", n[2].Port(), n[2].ID())); err != nil {
		panic(err)
	}

	if err := n[2].Connect(ctx, fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", n[3].Port(), n[3].ID())); err != nil {
		panic(err)
	}

	if err := n[3].Connect(ctx, fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", n[1].Port(), n[1].ID())); err != nil {
		panic(err)
	}

	//Wait to build a mesh
	time.Sleep(2 * time.Second)

	ctxSender, stopSender := context.WithCancel(ctx)

	if err := n[0].Publish(ctxSender, network.ChannelBlocks, []byte("hello")); err != nil {
		panic(err)
	}

	stopSender()

	//Wait all nodes received
	time.Sleep(5 * time.Second)
}
