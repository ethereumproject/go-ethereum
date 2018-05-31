package eth

type PMHandlerAddEvent struct {
	PMPeersLen int
	PMBestPeer *peer
	Peer       *peer
}

type PMHandlerRemoveEvent struct {
	PMPeersLen int
	PMBestPeer *peer
	Peer       *peer
}
