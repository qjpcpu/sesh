package job

import (
    "testing"
)

func TestMember(t *testing.T) {
    mgr, _ := NewManager()
    m1, _ := mgr.NewMember("m1")
    m2, _ := mgr.NewMember("m2")
    mgr.Broadcast("Hello all")
    msg, _ := m1.Receive(-1)
    if msg != "Hello all" {
        t.Error("can't Receive message")
    }
    msg, _ = m2.Receive(-1)
    if msg != "Hello all" {
        t.Error("can't Receive message")
    }
}
