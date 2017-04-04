package p2p_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cfg "github.com/tendermint/go-config"
	p2p "github.com/tendermint/go-p2p"
)

func createMConnection(conn net.Conn) *p2p.MConnection {
	onReceive := func(chID byte, msgBytes []byte) {
	}
	onError := func(r interface{}) {
	}
	return createMConnectionWithCallbacks(conn, onReceive, onError)
}

func createMConnectionWithCallbacks(conn net.Conn, onReceive func(chID byte, msgBytes []byte), onError func(r interface{})) *p2p.MConnection {
	config := cfg.NewMapConfig(map[string]interface{}{"send_rate": 512000, "recv_rate": 512000})
	chDescs := []*p2p.ChannelDescriptor{&p2p.ChannelDescriptor{ID: 0x01, Priority: 1}}

	return p2p.NewMConnection(config, conn, chDescs, onReceive, onError)
}

func TestMConnectionSend(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	mconn := createMConnection(client)
	_, err := mconn.Start()
	require.Nil(err)
	defer mconn.Stop()

	msg := "Ant-Man"
	assert.True(mconn.Send(0x01, msg))
	assert.False(mconn.CanSend(0x01))
	server.Read(make([]byte, len(msg)))
	assert.True(mconn.CanSend(0x01))

	msg = "Spider-Man"
	assert.True(mconn.TrySend(0x01, msg))
	server.Read(make([]byte, len(msg)))
}

func TestMConnectionReceive(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	receivedCh := make(chan []byte)
	errorsCh := make(chan interface{})
	onReceive := func(chID byte, msgBytes []byte) {
		receivedCh <- msgBytes
	}
	onError := func(r interface{}) {
		errorsCh <- r
	}
	mconn1 := createMConnectionWithCallbacks(client, onReceive, onError)
	_, err := mconn1.Start()
	require.Nil(err)
	defer mconn1.Stop()

	mconn2 := createMConnection(server)
	_, err = mconn2.Start()
	require.Nil(err)
	defer mconn2.Stop()

	msg := "Cyclops"
	assert.True(mconn2.Send(0x01, msg))

	select {
	case receivedBytes := <-receivedCh:
		assert.Equal([]byte(msg), receivedBytes[2:]) // first 3 bytes are internal
	case err := <-errorsCh:
		t.Fatalf("Expected %s, got %+v", msg, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Did not receive %s message in 500ms", msg)
	}
}

func TestMConnectionStatus(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	mconn := createMConnection(client)
	_, err := mconn.Start()
	require.Nil(err)
	defer mconn.Stop()

	status := mconn.Status()
	assert.NotNil(status)
	assert.Zero(status.Channels[0].SendQueueSize)
}
