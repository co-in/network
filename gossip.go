package network

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

var (
	topics = []string{
		"blocks",
	}
)

type Gossip struct {
	host          host.Host
	topics        map[string]*pubsub.Topic
	subscriptions map[string]*pubsub.Subscription
}

func NewGossip() (*Gossip, error) {
	//TODO random port
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.EnableRelay(), //NAT traversal
	)
	if err != nil {
		return nil, fmt.Errorf("libp2p: %w", err)
	}

	return &Gossip{
		host:          h,
		topics:        make(map[string]*pubsub.Topic, len(topics)),
		subscriptions: make(map[string]*pubsub.Subscription, len(topics)),
	}, nil
}

func (m *Gossip) Close() {
	_ = m.host.Close()
}

func (m *Gossip) Run(ctx context.Context) error {
	ps, err := pubsub.NewGossipSub(ctx, m.host)
	if err != nil {
		return fmt.Errorf("pubSub: %w", err)
	}

	for _, topicName := range topics {
		if m.topics[topicName], err = ps.Join(topicName); err != nil {
			return fmt.Errorf("join topic %q: %w", topicName, err)
		}

		if m.subscriptions[topicName], err = m.topics[topicName].Subscribe(); err != nil {
			return fmt.Errorf("subscribe blocks: %w", err)
		}

		go m.handler(ctx, m.subscriptions[topicName])
	}

	return nil
}

func connectToPeer(ctx context.Context, h host.Host, addrStr string) error {
	mAddr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("multiaddr.NewMultiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(mAddr)
	if err != nil {
		return fmt.Errorf("peer.AddrInfoFromP2pAddr: %w", err)
	}

	if err = h.Connect(ctx, *info); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	return nil
}

func (m *Gossip) handler(ctx context.Context, sub *pubsub.Subscription) {
	defer sub.Cancel()

	for {
		select {
		case <-ctx.Done():
		default:
			msg, err := sub.Next(ctx)
			if err != nil {
				fmt.Println("ERR:", err)
			}

			fmt.Println("MSG:", msg.Data)
		}
	}
}
