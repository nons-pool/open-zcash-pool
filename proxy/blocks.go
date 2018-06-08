package proxy

import (
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/jkkgbe/open-zcash-pool/rpc"
	"github.com/jkkgbe/open-zcash-pool/util"
)

const maxBacklog = 3

type heightDiffPair struct {
	diff   *big.Int
	height uint64
}

type BlockTemplate struct {
	sync.RWMutex
	PrevBlockHash   string
	coinbaseTxnData string
	coinbaseTxnHash string
	foundersReward  int
	longpollId      string
	minTime         int
	nonceRange      string
	curtime         int
	bits            string
	height          int
}

type Block struct {
	difficulty         int
	version            string
	prevHashReversed   string
	merkleRootReversed string
	reservedField      string
	nTime              string
	bits               string
	nonce              string
	header             string
}

// func (b Block) Difficulty() *big.Int     { return b.difficulty }
// func (b Block) HashNoNonce() common.Hash { return b.hashNoNonce }
// func (b Block) Nonce() uint64            { return b.nonce }
// func (b Block) MixDigest() common.Hash   { return b.mixDigest }
// func (b Block) NumberU64() uint64        { return b.number }

func (s *ProxyServer) fetchBlockTemplate() {
	rpc := s.rpc()
	t := s.currentBlockTemplate()
	var reply BlockTemplate
	err := rpc.GetBlockTemplate(&reply)
	if err != nil {
		log.Printf("Error while refreshing block template on %s: %s", rpc.Name, err)
		return
	}
	// No need to update, we have fresh job
	if t != nil && t.PrevBlockHash == reply.PrevBlockHash {
		return
	}

	// TODO calc merkle root etc

	// TODO
	newTemplate := Block{
		difficulty:         1,
		version:            "",
		prevHashReversed:   "",
		merkleRootReversed: "",
		reservedField:      "",
		nTime:              "",
		bits:               "",
		nonce:              "",
		header:             "",
	}
	// Copy job backlog and add current one
	newTemplate.headers[reply[0]] = heightDiffPair{
		diff:   util.TargetHexToDiff(reply[2]),
		height: height,
	}
	if t != nil {
		for k, v := range t.headers {
			if v.height > height-maxBacklog {
				newTemplate.headers[k] = v
			}
		}
	}
	s.blockTemplate.Store(&newTemplate)
	log.Printf("New block to mine on %s at height %d / %s", rpc.Name, height, reply[0][0:10])

	// Stratum
	if s.config.Proxy.Stratum.Enabled {
		go s.broadcastNewJobs()
	}
}

func (s *ProxyServer) fetchPendingBlock() (*rpc.GetBlockReplyPart, uint64, int64, error) {
	rpc := s.rpc()
	reply, err := rpc.GetPendingBlock()
	if err != nil {
		log.Printf("Error while refreshing pending block on %s: %s", rpc.Name, err)
		return nil, 0, 0, err
	}
	blockNumber, err := strconv.ParseUint(strings.Replace(reply.Number, "0x", "", -1), 16, 64)
	if err != nil {
		log.Println("Can't parse pending block number")
		return nil, 0, 0, err
	}
	blockDiff, err := strconv.ParseInt(strings.Replace(reply.Difficulty, "0x", "", -1), 16, 64)
	if err != nil {
		log.Println("Can't parse pending block difficulty")
		return nil, 0, 0, err
	}
	return reply, blockNumber, blockDiff, nil
}
