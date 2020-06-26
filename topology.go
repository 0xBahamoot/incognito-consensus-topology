package icnwtopology

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type TopologyManager struct {
	network    NetworkManager
	blockchain Blockchain
	consensus  Consensus
	committee  []string
	memberID   string
	//Tree of each root will be cache here
	trees map[string]*Tree

	//if synchronize is enable all op will have a timeout
	synchronize bool
}

type NetworkManager interface {
	GetPeerIDOfIncognitoID(ID string) (string, error)
	SendMessageToPeerID(peerID string, msg []byte) error
	IsHaveConnection(peerID string) bool
	ConnectToPeerID(peerID string) error
}

type Consensus interface {
	GetCommitteeVoteForBlockHash(member string, blockHash string) ([]byte, error)
	CreateMsgVote() []byte
	CreateMsgBlock() []byte
	CreateMsgRequestBlock() []byte
	CreateMsgRequestVote() []byte
	CombineVotes(votes [][]byte) []byte
}

type Blockchain interface {
	GetBestCommittee() []string
	GetMaxBlockCreateTime() time.Duration
}

func NewTopologyManager(network NetworkManager, bc Blockchain, cs Consensus, memberID string, synchronize bool) *TopologyManager {
	topo := &TopologyManager{
		network:     network,
		blockchain:  bc,
		consensus:   cs,
		memberID:    memberID,
		trees:       make(map[string]*Tree),
		synchronize: synchronize,
	}
	return topo
}

func (topo *TopologyManager) Start() {
	topo.committee = topo.blockchain.GetBestCommittee()
	for _, member := range topo.committee {
		tree, err := CreateTree(member, topo.committee)
		if err != nil {
			fmt.Println(err)
		}
		topo.trees[member] = tree
	}
}

func (topo *TopologyManager) BroadcastProposeBlockMsg(proposer string, proposeMsg []byte) {
	lNode, rNode, err := topo.trees[proposer].GetChild(topo.memberID)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Broadcast block to", lNode, rNode)
	if lNode != "" {
		topo.network.SendMessageToPeerID(lNode, proposeMsg)
	}
	if rNode != "" {
		topo.network.SendMessageToPeerID(rNode, proposeMsg)
	}
}

func (topo *TopologyManager) BroadcastlAggregatedVoteMsg(proposer string, aggVoteMsg []byte) {
	lNode, rNode, err := topo.trees[proposer].GetChild(topo.memberID)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("BroadcastlAggregatedVoteMsg", lNode, rNode)
	if lNode != "" {
		topo.network.SendMessageToPeerID(lNode, aggVoteMsg)
	}
	if rNode != "" {
		topo.network.SendMessageToPeerID(rNode, aggVoteMsg)
	}
}

func (topo *TopologyManager) SendVote(ctx context.Context, proposer string, blockHash string, vote []byte) {
	//will wait for child votes to be collected
	if topo.synchronize {
		waitTime := getWaitTime(topo.trees[proposer].GetNodeHeight(topo.memberID))
		fmt.Println(proposer, topo.memberID, topo.trees[proposer].GetNodeHeight(topo.memberID), waitTime.String())
		t := time.NewTimer(waitTime)
		<-t.C
	}
	votes, err := topo.requestVoteFromChild(ctx, proposer, topo.memberID, blockHash)
	if err != nil {
		fmt.Println(err)
	}
	if topo.memberID != proposer {
		parentNode, err := topo.trees[proposer].GetParent(topo.memberID)
		if err != nil {
			fmt.Println(err)
			return
		}
		votes = append(votes, vote)
		combineVotes := topo.consensus.CombineVotes(votes)
		fmt.Println("Vote combine", combineVotes)
		topo.sendMsgVote(parentNode, combineVotes)
	}

}
func (topo *TopologyManager) requestVoteFromChild(ctx context.Context, proposer, current, blockHash string) (votes [][]byte, err error) {
	lNode, rNode, err := topo.trees[proposer].GetChild(current)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("requestVoteFromChild", lNode, rNode)
	var lVotes, rVotes [][]byte
	var wg sync.WaitGroup
	if lNode != "" {
		wg.Add(1)
		go func() {
			vote := topo.tryGetVoteFromMember(ctx, proposer, lNode, blockHash)
			if vote != nil {
				fmt.Println("right node votes got", lNode)
				lVotes = append([][]byte{}, vote)
			} else {
				lVotes, err = topo.requestVoteFromChild(ctx, proposer, lNode, blockHash)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
			wg.Done()
		}()
	}
	if rNode != "" {
		wg.Add(1)
		go func() {
			vote := topo.tryGetVoteFromMember(ctx, proposer, rNode, blockHash)
			if vote != nil {
				fmt.Println("right node votes got", rNode)
				rVotes = append([][]byte{}, vote)
			} else {
				rVotes, err = topo.requestVoteFromChild(ctx, proposer, rNode, blockHash)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()
	if lVotes != nil {
		votes = append(votes, lVotes...)
	}
	if rVotes != nil {
		votes = append(votes, rVotes...)
	}
	return
}

func (topo *TopologyManager) tryGetVoteFromMember(ctx context.Context, proposer, member, blockHash string) []byte {
	for {
		select {
		default:
			if v, err := topo.consensus.GetCommitteeVoteForBlockHash(member, blockHash); err != nil {
				topo.sendMsgRequestVote(member, blockHash)
				if topo.synchronize { //will get only 1 more time
					t := time.NewTimer(5 * time.Second)
					<-t.C
					fmt.Println("retry 1 more time", member)
					if v, err := topo.consensus.GetCommitteeVoteForBlockHash(member, blockHash); err == nil {
						return v
					}
					return nil
				} else { //will retry until got
					t := time.NewTimer(2 * time.Second)
					<-t.C
					fmt.Println("retry", member)
				}
			} else {
				fmt.Println("got vote", member)
				return v
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func getWaitTime(height int64) time.Duration {
	return time.Duration(height*defaultBlockSendTime)*time.Second + defaultBlockValidationTime
}
