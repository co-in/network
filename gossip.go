package network

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	ChannelBlocks = "blocks"
)

var (
	topics = []string{
		ChannelBlocks,
	}
)

type Gossip struct {
	host          host.Host
	topics        map[string]*pubsub.Topic
	subscriptions map[string]*pubsub.Subscription
}

type Config struct {
	Port uint16
}

type Option func(*Config)

func WithPort(value uint16) Option {
	return func(cfg *Config) { cfg.Port = value }
}

func NewGossip(options ...Option) (*Gossip, error) {
	cfg := &Config{
		Port: 9090,
	}

	for _, option := range options {
		option(cfg)
	}

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Port)),
		libp2p.EnableRelay(), //NAT traversal
	)
	if err != nil {
		return nil, fmt.Errorf("libp2p: %w", err)
	}

	m := &Gossip{
		host:          h,
		topics:        make(map[string]*pubsub.Topic, len(topics)),
		subscriptions: make(map[string]*pubsub.Subscription),
	}

	log.Println(m.Port(), "Started", m.host.ID().ShortString())

	return m, nil
}

func (m *Gossip) ID() string {
	return m.host.ID().String()
}

func portFromAddress(addr multiaddr.Multiaddr) uint16 {
	if tcpComponent, err := addr.ValueForProtocol(multiaddr.P_TCP); err == nil {
		port64, _ := strconv.ParseUint(tcpComponent, 10, 16)

		return uint16(port64)
	}

	return 0
}

func (m *Gossip) Port() uint16 {
	for _, addr := range m.host.Addrs() {
		if port := portFromAddress(addr); port != 0 {
			return port
		}
	}

	return 0
}

func (m *Gossip) Close() {
	_ = m.host.Close()
}

func (m *Gossip) Join(ctx context.Context) error {
	ps, err := pubsub.NewGossipSub(ctx, m.host) //pubsub.WithMessageSigning(false),
	//pubsub.WithStrictSignatureVerification(false),

	if err != nil {
		return fmt.Errorf("pubSub: %w", err)
	}

	for _, topicName := range topics {
		if m.topics[topicName], err = ps.Join(topicName); err != nil {
			return fmt.Errorf("join %q: %w", topicName, err)
		}

	}

	return nil
}

func (m *Gossip) Subscribe(ctx context.Context) error {
	var err error

	if len(m.subscriptions) == 0 {
		if err = m.Join(ctx); err != nil {
			return fmt.Errorf("join: %w", err)
		}
	}

	for _, topicName := range topics {
		if m.subscriptions[topicName], err = m.topics[topicName].Subscribe(); err != nil {
			return fmt.Errorf("subscribe %q: %w", topicName, err)
		}

		go m.handler(ctx, m.subscriptions[topicName])
	}

	return nil
}

func (m *Gossip) Connect(ctx context.Context, addrStr string) error {
	mAddr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("multiaddr.NewMultiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(mAddr)
	if err != nil {
		return fmt.Errorf("peer.AddrInfoFromP2pAddr: %w", err)
	}

	if err = m.host.Connect(ctx, *info); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	if err = m.waitForConnection(ctx, info.ID); err != nil {
		return fmt.Errorf("wait for connection: %w", err)
	}

	log.Printf("%d Connected to [%d]: %s", m.Port(), portFromAddress(mAddr), info.ID.ShortString())

	return nil
}

func (m *Gossip) waitForConnection(ctx context.Context, peerID peer.ID) error {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for connection to peer %s", peerID)
		case <-ticker.C:
			connectedness := m.host.Network().Connectedness(peerID)
			if connectedness == network.Connected {
				if len(m.host.Network().ConnsToPeer(peerID)) > 0 {
					return nil
				}
			}
		}
	}
}

func (m *Gossip) Publish(ctx context.Context, channel string, data []byte) error {
	t, ok := m.topics[channel]
	if !ok {
		return fmt.Errorf("channel not found %q", channel)
	}

	log.Println(m.Port(), "SEND:", string(data))

	if err := t.Publish(ctx, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func (m *Gossip) handler(ctx context.Context, sub *pubsub.Subscription) {
	defer func() {
		log.Println(m.Port(), "SUBSCRIBER CLOSED")

		sub.Cancel()
	}()

	for {
		select {
		case <-ctx.Done():
		default:
			msg, err := sub.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}

				log.Println(m.Port(), "ERR:", err)

				continue
			}

			//Subscribe to self
			if msg.ReceivedFrom == m.host.ID() {
				continue
			}

			log.Printf("%d FROM: %s, RECV: %s",
				m.Port(),
				string(msg.Data),
				msg.ReceivedFrom.ShortString(),
			)
		}
	}
}
