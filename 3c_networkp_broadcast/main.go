package main

import (
	"context"
	"encoding/json"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"log"
	"sync"
)

type server struct {
	n  *maelstrom.Node
	br *broadcastWorkers
	sync.Mutex
	msgs []any
}

func (s *server) writeMessage(m map[string]any) {

	//Lock, write and unlock messages
	//Locking elsewhere would choke the system, resulting in lost messages

	s.Lock()
	defer s.Unlock()
	s.msgs = append(s.msgs, m["message"])
}

func (s *server) broadcast(msg maelstrom.Message) error {
	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.writeMessage(body) //Write message to messages list

	for _, dst := range s.n.NodeIDs() {

		if dst == msg.Src || dst == s.n.ID() { //skip if destination is source or current node
			continue
		}

		s.br.broadcast(broadcastMsg{
			dst:  dst,
			body: body,
		})
	}

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
	delete(body, "topology") //Delete key because it is not required
	body["type"] = "topology_ok"

	return s.n.Reply(msg, body)
}

func main() {

	n := maelstrom.NewNode()

	S := server{
		n:  n,
		br: newBroadcastWorkers(n, 10),
	}

	n.Handle("broadcast", S.broadcast)
	n.Handle("read", S.read)
	n.Handle("topology", S.topology)

	if err := S.n.Run(); err != nil {
		log.Fatal(err)
	}

}

type broadcastMsg struct {
	dst  string
	body map[string]any
}

type broadcastWorkers struct {
	cancel context.CancelFunc
	ch     chan broadcastMsg
}

func newBroadcastWorkers(n *maelstrom.Node, worker int) *broadcastWorkers {

	ch := make(chan broadcastMsg) // Channel to be used by workers to receive messages.
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < worker; i++ { //Start workers

		go func() { // create a goroutine per worker
			for {
				select {
				case msg := <-ch: // Receive/Wait on message from channel
					for { //Retry message send until successful
						if err := n.Send(msg.dst, msg.body); err != nil {
							continue
						}
						break // break retry on success.
					}
				case <-ctx.Done():
					return
				}
			}
		}()

	}

	return &broadcastWorkers{
		ch:     ch,
		cancel: cancel,
	}
}

func (b *broadcastWorkers) broadcast(msg broadcastMsg) {
	b.ch <- msg
}

func (b *broadcastWorkers) close() {
	b.cancel()
}
