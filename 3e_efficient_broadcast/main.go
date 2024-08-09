package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/emirpasic/gods/trees/btree"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"log"
	"strconv"
	"sync"
	"time"
)

type server struct {
	n      *maelstrom.Node
	nodeID string
	id     int

	sync.RWMutex

	msgs map[int]any
	tree *btree.Tree
}

func (s *server) getNeighbors() []string {
	n := s.tree.GetNode(s.id)
	var neighbors []string
	if n.Parent != nil {
		neighbors = append(neighbors, n.Parent.Entries[0].Value.(string))
	}
	for _, children := range n.Children {
		for _, entry := range children.Entries {
			neighbors = append(neighbors, entry.Value.(string))
		}
	}

	return neighbors
}

func (s *server) rpc(dst string, body map[string]any) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := s.n.SyncRPC(ctx, dst, body)
	return err
}

func (s *server) initHandler(_ maelstrom.Message) error {
	s.nodeID = s.n.ID()
	id, err := strconv.Atoi(s.nodeID[1:])
	if err != nil {
		return err
	}
	s.id = id
	return nil
}

func (s *server) broadcast(msg maelstrom.Message) error {
	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}
	go func() {
		_ = s.n.Reply(msg, map[string]any{ // Respond immediately with Ok
			"type": "broadcast_ok",
		})
	}()

	message := int(body["message"].(float64))
	s.Lock()
	if _, exists := s.msgs[message]; exists {
		s.Unlock()
		return nil
	}
	s.msgs[message] = "" //Write message to messages list
	s.Unlock()

	s.RLock()
	defer s.RUnlock()
	for _, dst := range s.getNeighbors() {

		if dst == msg.Src || dst == s.n.ID() { //skip if destination is source or current node
			continue
		}

		dst := dst
		go func() { // create a goroutine per neighbor to send messages.

			for j := 1; j <= 100; j++ { //Retry messages for 100 times.
				if err := s.rpc(dst, body); err != nil {
					// Sleep and retry
					continue
				}
				break // break retry on success.
			}

		}()
	}

	return nil
}

func (s *server) read(msg maelstrom.Message) error {

	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.RLock()

	//Need to make and then assign so that we don't lose messages. Easiest way to copy stuff between lists in golang
	m := make([]int, 0) // Create a slice with 0 length
	for message := range s.msgs {
		m = append(m, message)
	}
	s.RUnlock()

	return s.n.Reply(msg, map[string]any{
		"type":     "read_ok",
		"messages": m,
	})
}

func (s *server) topology(msg maelstrom.Message) error {
	s.Lock()
	tree := btree.NewWithIntComparator(len(s.n.NodeIDs()))
	for i := 0; i < len(s.n.NodeIDs()); i++ {
		tree.Put(i, fmt.Sprintf("n%d", i))
	}

	s.tree = tree
	s.Unlock()

	return s.n.Reply(msg, map[string]any{
		"type": "topology_ok",
	})
}

/*
 * Main function, setting up server etc.
 */

func main() {

	n := maelstrom.NewNode()

	S := server{
		n:    n,
		msgs: make(map[int]any),
	}

	n.Handle("init", S.initHandler)
	n.Handle("broadcast", S.broadcast)
	n.Handle("read", S.read)
	n.Handle("topology", S.topology)

	if err := S.n.Run(); err != nil {
		log.Fatal(err)
	}

}
