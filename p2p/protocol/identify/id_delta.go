package identify

import (
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	pb "github.com/torusresearch/go-libp2p/p2p/protocol/identify/pb"

	ggio "github.com/gogo/protobuf/io"
)

const IDDelta = "/p2p/id/delta/1.0.0"

// deltaHandler handles incoming delta updates from peers.
func (ids *IDService) deltaHandler(s network.Stream) {
	c := s.Conn()

	r := ggio.NewDelimitedReader(s, 2048)
	mes := pb.Identify{}
	if err := r.ReadMsg(&mes); err != nil {
		log.Warning("error reading identify message: ", err)
		s.Reset()
		return
	}

	defer helpers.FullClose(s)

	log.Debugf("%s received message from %s %s", s.Protocol(), c.RemotePeer(), c.RemoteMultiaddr())

	delta := mes.GetDelta()
	if delta == nil {
		return
	}

	p := s.Conn().RemotePeer()
	if err := ids.consumeDelta(p, delta); err != nil {
		log.Warningf("delta update from peer %s failed: %s", p, err)
	}
}

// consumeDelta processes an incoming delta from a peer, updating the peerstore
// and emitting the appropriate events.
func (ids *IDService) consumeDelta(id peer.ID, delta *pb.Delta) error {
	err := ids.Host.Peerstore().AddProtocols(id, delta.GetAddedProtocols()...)
	if err != nil {
		return err
	}

	err = ids.Host.Peerstore().RemoveProtocols(id, delta.GetRmProtocols()...)
	if err != nil {
		return err
	}

	evt := event.EvtPeerProtocolsUpdated{
		Peer:    id,
		Added:   protocol.ConvertFromStrings(delta.GetAddedProtocols()),
		Removed: protocol.ConvertFromStrings(delta.GetRmProtocols()),
	}
	ids.emitters.evtPeerProtocolsUpdated.Emit(evt)
	return nil
}
