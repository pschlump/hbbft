package main

import (
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/pschlump/hbbft"
)

func main() {
	benchmark(4, 128, 100)
	benchmark(6, 128, 200)
	benchmark(8, 128, 400)
	benchmark(12, 128, 1000)
}

type message struct {
	from    uint64
	payload hbbft.MessageTuple
}

func benchmark(n, txsize, batchSize int) {
	log.Printf("Starting benchmark %d nodes %d tx size %d batch size over 5 seconds...", n, txsize, batchSize)
	var (
		nodes    = makeNodes(n, 10000, txsize, batchSize)
		messages = make(chan message, 1024*1024)
	)

	log.Printf("about to start (%d) nodes\n", n)
	for _, node := range nodes {
		if err := node.Start(); err != nil {
			log.Fatal(err)
		}
		for _, msg := range node.Messages() {
			messages <- message{node.ID, msg}
		}
	}

	log.Printf("running test with (%d) nodes\n", n)
	timer := time.After(5 * time.Second)
	nSeconds := 1
	timer = time.After(time.Duration(nSeconds) * time.Second)
running:
	for {
		select {
		case messag := <-messages:
			fmt.Printf(".")
			node := nodes[messag.payload.To]
			hbmsg := messag.payload.Payload.(hbbft.HBMessage)
			if err := node.HandleMessage(messag.from, hbmsg.Epoch, hbmsg.Payload.(*hbbft.ACSMessage)); err != nil {
				log.Fatal(err)
			}
			for _, msg := range node.Messages() {
				messages <- message{node.ID, msg}
			}
		case <-timer:
			fmt.Printf("x")
			for _, node := range nodes {
				fmt.Printf("X")
				total := 0
				for _, txx := range node.Outputs() {
					total += len(txx)
				}
				log.Printf("\n\nnode (%d) processed a total of (%d) transactions in %d seconds [ %d tx/s ]",
					node.ID, total, nSeconds, total/nSeconds)
			}
			break running
		default:
			fmt.Printf("!")
		}
	}
}

func makeNodes(n, ntx, txsize, batchSize int) []*hbbft.HoneyBadger {
	nodes := make([]*hbbft.HoneyBadger, n)
	for i := 0; i < n; i++ {
		cfg := hbbft.Config{
			N:         n,
			ID:        uint64(i),
			Nodes:     makeids(n),
			BatchSize: batchSize,
		}
		nodes[i] = hbbft.NewHoneyBadger(cfg)
		for ii := 0; ii < ntx; ii++ {
			nodes[i].AddTransaction(newTx(txsize))
		}
	}
	return nodes
}

func makeids(n int) []uint64 {
	ids := make([]uint64, n)
	for i := 0; i < n; i++ {
		ids[i] = uint64(i)
	}
	return ids
}

type tx struct {
	Nonce uint64
	Data  []byte
}

// Size can be used to simulate large transactions in the network.
func newTx(size int) *tx {
	return &tx{
		Nonce: rand.Uint64(),
		Data:  make([]byte, size),
	}
}

// Hash implements the hbbft.Transaction interface.
func (t *tx) Hash() []byte {
	buf := make([]byte, 8) // sizeOf(uint64) + len(data)
	binary.LittleEndian.PutUint64(buf, t.Nonce)
	return buf
}

func init() {
	rand.Seed(time.Now().UnixNano())
	gob.Register(&tx{})
}
