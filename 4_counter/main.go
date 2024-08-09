package main

import (
	"context"
	"encoding/json"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"log"
	"strconv"
	"sync"
)

type server struct {
	n      *maelstrom.Node
	nodeID string
	id     int
	kv     *maelstrom.KV

	sync.RWMutex
}

func (s *server) initHandler(_ maelstrom.Message) error {
	s.nodeID = s.n.ID()
	id, err := strconv.Atoi(s.nodeID[1:])
	if err != nil {
		return err
	}
	s.id = id

	// initializing the cache with the current node id so that it does not fail
	ctx := context.Background()
	err = s.kv.Write(ctx, s.nodeID, 0)
	if err != nil {
		return err
	}
	return nil
}

func (s *server) add(msg maelstrom.Message) error {

	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}
	ctx := context.Background()

	delta := int(body["delta"].(float64))

	s.Lock()
	c, err := s.kv.ReadInt(ctx, s.nodeID)
	if err != nil {
		return err
	}

	err = s.kv.Write(ctx, s.nodeID, c+delta)
	if err != nil {
		return err
	}
	s.Unlock()

	return s.n.Reply(msg, map[string]any{
		"type": "add_ok",
	})
}

func (s *server) readLocal(msg maelstrom.Message) error {

	s.RLock()
	ctx := context.Background()

	c, _ := s.kv.ReadInt(ctx, s.nodeID)
	defer s.RUnlock()

	return s.n.Reply(msg, map[string]any{
		"value": c,
	})
}

func (s *server) read(msg maelstrom.Message) error {

	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.RLock()
	ctx := context.Background()
	sum := 0
	for _, nodeId := range s.n.NodeIDs() { // fetch values from every nodeId available

		//if s.nodeID == nodeId {
		var respBody map[string]any
		resp, err := s.n.SyncRPC(ctx, nodeId, map[string]any{
			"type": "readLocal",
		})
		if err != nil {
			return err
		}
		if err = json.Unmarshal(resp.Body, &respBody); err != nil {
			return err
		}
		sum += int(respBody["value"].(float64))

	}

	defer s.RUnlock()

	return s.n.Reply(msg, map[string]any{
		"type":  "read_ok",
		"value": sum,
	})
}

/*
 * Main function, setting up server etc.
 */

func main() {

	n := maelstrom.NewNode()

	s := server{
		n:      n,
		nodeID: n.ID(),
		kv:     maelstrom.NewSeqKV(n),
	}

	n.Handle("add", s.add)
	n.Handle("read", s.read)
	n.Handle("readLocal", s.readLocal)
	n.Handle("init", s.initHandler)

	if err := s.n.Run(); err != nil {
		log.Fatal(err)
	}

}
