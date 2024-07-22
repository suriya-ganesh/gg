package main

import (
	"encoding/json"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"log"
	"sync"
)

type server struct {
	n *maelstrom.Node
	sync.Mutex
	msgs []any
}

func (s *server) broadcast(msg maelstrom.Message) error {
	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	//Lock and unlock
	s.Lock()
	defer s.Unlock()
	s.msgs = append(s.msgs, body["message"])

	return s.n.Reply(msg, map[string]any{
		"type": "broadcast_ok",
	})
}

func (s *server) read(msg maelstrom.Message) error {

	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	//Need to make and then assign so that we don't lose messages. Easiest way to copy stuff between lists in golang

	m := make([]any, len(s.msgs))
	for i := 0; i < len(s.msgs); i++ {
		m[i] = s.msgs[i]
	}

	return s.n.Reply(msg, map[string]any{
		"type":     "read_ok",
		"messages": m,
	})
}

func (s *server) topology(msg maelstrom.Message) error {
	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}
	delete(body, "topology") //Delete key because it is not required (for this exercise)
	body["type"] = "topology_ok"

	return s.n.Reply(msg, body)
}

func main() {

	n := maelstrom.NewNode()

	S := server{
		n: n,
	}

	n.Handle("broadcast", S.broadcast)
	n.Handle("read", S.read)
	n.Handle("topology", S.topology)

	if err := S.n.Run(); err != nil {
		log.Fatal(err)
	}

}
