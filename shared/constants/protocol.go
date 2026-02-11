package constants

import "strings"

// Protocol is the transport protocol.
type Protocol string

const (
	// ProtocolHTTP is the HTTP protocol
	ProtocolHTTP Protocol = "http"
	// ProtocolGRPC is the GRPC protocol.
	ProtocolGRPC Protocol = "grpc"
)

func (p Protocol) Normalize() Protocol {
	switch Protocol(strings.ToLower(string(p))) {
	case ProtocolGRPC:
		return ProtocolGRPC
	default:
		return ProtocolHTTP
	}
}

func (p Protocol) IsValid() bool {
	switch p {
	case ProtocolHTTP, ProtocolGRPC:
		return true
	default:
		return false
	}
}

func (p Protocol) EnumValues() []string {
	return []string{string(ProtocolHTTP), string(ProtocolGRPC)}
}
