package job

import (
    "testing"
)

func TestMember(t *testing.T) {
    mgr, _ := NewManager()
    m1, _ := mgr.NewMember("m1")
    m2, _ := mgr.NewMember("m2")
    // broadcast to all
    mgr.Broadcast("Hello all")
    // block until received a message
    msg, _ := m1.Receive(-1)
    if msg != "Hello all" {
        t.Error("can't Receive message")
    }
    msg, _ = m2.Receive(-1)
    if msg != "Hello all" {
        t.Error("can't Receive message")
    }
    // block until received a message or 100 milliseconds timeout
    _, err := m2.Receive(100)
    if err == nil || err.Error() != "timeout" {
        t.Error("shouldn't Receive messagg")
    }

    // send message to master
    err = m1.Send(MASTER_ID, "hello master, I'm m1.")
    if err != nil {
        t.Error("send message to master fail.")
    }
    // send message to other member by id
    err = m1.Send("m2", "hello m2, I'm m1.")
    if err != nil {
        t.Error("send message to m2 fail.")
    }

    msg, err = mgr.Receive(-1)
    if err != nil || msg != "hello master, I'm m1." {
        t.Error("Receive message from m1 fail")
    }
    msg, err = m2.Receive(-1)
    if err != nil || msg != "hello m2, I'm m1." {
        t.Error("Receive message from m1 fail")
    }
}
