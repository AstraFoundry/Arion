//go:build !darwin || !cgo

package imebridge

import (
	"arion.local/core/domain"
	"net"
)

// Windows obtains its proof from the accepted pipe wrapper. Other transports
// without a native identity implementation cannot authenticate a peer.
func inspectPlatformPeer(net.Conn) (PeerIdentity, func(), error) {
	return PeerIdentity{}, func() {}, domain.Fail(domain.CodeArtifactUnverified, domain.DetailInputMethodArtifactUnverified)
}
