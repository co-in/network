package network_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type logConsumer struct {
	log  func(format string, args ...any)
	name string
}

func (m *logConsumer) Accept(log testcontainers.Log) {
	m.log("[%s] > %s", m.name, string(log.Content))
}

func newInstance(ctx context.Context, t *testing.T, req testcontainers.ContainerRequest) (string, func()) {
	container, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
	if err != nil {
		t.Fatal(err)
	}

	id := container.GetContainerID()
	container.FollowOutput(&logConsumer{name: id[:12], log: t.Logf})
	err = container.StartLogProducer(ctx)
	if err != nil {
		t.Fatal(err)
	}

	port, _ := container.MappedPort(ctx, "9090")

	return fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", port), func() { _ = container.Terminate(ctx) }
}

func TestGossip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    ".",
			Dockerfile: "Dockerfile",
		},
		ExposedPorts: []string{"9090/tcp"},
		WaitingFor:   wait.ForExposedPort(),
	}

	addr1, closer1 := newInstance(ctx, t, req)
	addr2, closer2 := newInstance(ctx, t, req)
	addr3, closer3 := newInstance(ctx, t, req)

	defer closer1()
	defer closer2()
	defer closer3()
	fmt.Println(addr1, addr2, addr3)

	//g, err := network.NewGossip()
	//assert.NoError(t, err)
	//
	//err = g.Join(ctx)
	//assert.NoError(t, err)
	//
	//err = g.Connect(ctx, addr1)
	//assert.NoError(t, err)
	//err = g.Connect(ctx, addr3)
	//assert.NoError(t, err)
	//
	//err = g.Publish(ctx, network.ChannelBlocks, []byte("hello"))
	//assert.NoError(t, err)
	//
	//time.Sleep(10 * time.Second)
}
