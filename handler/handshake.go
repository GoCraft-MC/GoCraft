// Package handler contains per-state protocol handlers that read from and
// write to a *network.ClientConn.
package handler

import (
	"errors"
	"fmt"
	"log/slog"

	"GoCraft/config"
	"GoCraft/network"
	"GoCraft/protocol"
)

const (
	// Packet IDs — Handshaking state (client → server)
	packetIDHandshake = 0x00

	// Next-state values encoded in the Handshake packet
	nextStateStatus = 1
	nextStateLogin  = 2
)

// ErrLoginNotImplemented is returned when the client requests login mode.
// Login support is planned for Milestone 2.
var ErrLoginNotImplemented = errors.New("handler: login state is not yet implemented")

// HandshakeData contains the fields decoded from the client's Handshake packet.
type HandshakeData struct {
	ProtocolVersion int32
	ServerAddress   string
	ServerPort      uint16
	NextState       int32
}

// Handshake reads the client's opening Handshake packet and transitions the
// connection into the appropriate next state.
//
// After a successful return the caller should invoke the handler for
// conn.State (HandleStatus for StateStatus, etc.).
func Handshake(conn *network.ClientConn, cfg *config.Config) (*HandshakeData, error) {
	pkt, err := conn.ReadPacket()
	if err != nil {
		return nil, fmt.Errorf("handshake: reading packet: %w", err)
	}

	if pkt.ID != packetIDHandshake {
		return nil, fmt.Errorf("handshake: expected packet 0x00, got 0x%02X", pkt.ID)
	}

	r := pkt.Reader()

	protoVersion, err := protocol.ReadVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("handshake: reading protocol version: %w", err)
	}

	serverAddr, err := protocol.ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("handshake: reading server address: %w", err)
	}

	serverPort, err := protocol.ReadUShort(r)
	if err != nil {
		return nil, fmt.Errorf("handshake: reading server port: %w", err)
	}

	nextState, err := protocol.ReadVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("handshake: reading next state: %w", err)
	}

	data := &HandshakeData{
		ProtocolVersion: protoVersion,
		ServerAddress:   serverAddr,
		ServerPort:      serverPort,
		NextState:       nextState,
	}

	slog.Debug("handshake received",
		"remote", conn.RemoteAddr(),
		"protocolVersion", protoVersion,
		"serverAddr", serverAddr,
		"serverPort", serverPort,
		"nextState", nextState,
	)

	switch nextState {
	case nextStateStatus:
		conn.State = network.StateStatus
	case nextStateLogin:
		conn.State = network.StateLogin
	default:
		return nil, fmt.Errorf("handshake: unknown next state %d", nextState)
	}

	return data, nil
}
