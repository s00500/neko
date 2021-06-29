package webrtc

import (
	"sync"

	"github.com/pion/webrtc/v3"
)

type WebRTCPeerCtx struct {
	mu          sync.Mutex
	api         *webrtc.API
	connection  *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel
	changeVideo func(videoID string) error
	iceTrickle  bool
}

func (peer *WebRTCPeerCtx) CreateOffer(ICERestart bool) (*webrtc.SessionDescription, error) {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	offer, err := peer.connection.CreateOffer(&webrtc.OfferOptions{
		ICERestart: ICERestart,
	})
	if err != nil {
		return nil, err
	}

	if !peer.iceTrickle {
		// Create channel that is blocked until ICE Gathering is complete
		gatherComplete := webrtc.GatheringCompletePromise(peer.connection)

		if err := peer.connection.SetLocalDescription(offer); err != nil {
			return nil, err
		}

		<-gatherComplete
	} else {
		if err := peer.connection.SetLocalDescription(offer); err != nil {
			return nil, err
		}
	}

	return peer.connection.LocalDescription(), nil
}

func (peer *WebRTCPeerCtx) SignalAnswer(sdp string) error {
	return peer.connection.SetRemoteDescription(webrtc.SessionDescription{
		SDP:  sdp,
		Type: webrtc.SDPTypeAnswer,
	})
}

func (peer *WebRTCPeerCtx) SignalCandidate(candidate webrtc.ICECandidateInit) error {
	return peer.connection.AddICECandidate(candidate)
}

func (peer *WebRTCPeerCtx) SetVideoID(videoID string) error {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	return peer.changeVideo(videoID)
}

func (peer *WebRTCPeerCtx) Destroy() error {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	if peer.connection == nil || peer.connection.ConnectionState() != webrtc.PeerConnectionStateConnected {
		return nil
	}

	return peer.connection.Close()
}
