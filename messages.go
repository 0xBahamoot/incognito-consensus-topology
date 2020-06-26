package icnwtopology

import "fmt"

func (topo *TopologyManager) sendMsgRequestProposeBlock(root string) {
	msg := topo.consensus.CreateMsgRequestBlock()
	peerID, err := topo.network.GetPeerIDOfIncognitoID(root)
	if err != nil {
		fmt.Println(err)
		return
	}
	topo.network.SendMessageToPeerID(peerID, msg)
}

func (topo *TopologyManager) sendMsgRequestVote(member, blockHash string) {
	fmt.Println("sendMsgRequestVote", member)
	msg := topo.consensus.CreateMsgRequestVote()
	peerID, err := topo.network.GetPeerIDOfIncognitoID(member)
	if err != nil {
		fmt.Println(err)
		return
	}
	topo.network.SendMessageToPeerID(peerID, msg)
}

func (topo *TopologyManager) sendMsgVote(member string, vote []byte) {
	msg := topo.consensus.CreateMsgVote()
	peerID, err := topo.network.GetPeerIDOfIncognitoID(member)
	if err != nil {
		fmt.Println(err)
		return
	}
	topo.network.SendMessageToPeerID(peerID, msg)
}
