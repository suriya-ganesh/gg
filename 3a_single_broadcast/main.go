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

	body["type"] = "broadcast_ok"
	v := body["message"]
	s.Lock()
	defer s.Unlock()
	s.msgs = append(s.msgs, v)
	delete(body, "message")

	return s.n.Reply(msg, body)
}

func (s *server) read(msg maelstrom.Message) error {

	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "read_ok"
	s.Lock()
	defer s.Unlock()
	m := make([]any, len(s.msgs))
	for i := 0; i < len(s.msgs); i++ {
		m[i] = s.msgs[i]
	}
	body["messages"] = m

	return s.n.Reply(msg, body)
}

func (s *server) topology(msg maelstrom.Message) error {
	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}
	delete(body, "topology") //Delete key because it is not required
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
