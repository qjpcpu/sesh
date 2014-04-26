package job

import (
    "errors"
    "time"
)

type MessagePool struct {
    queue map[string]chan interface{}
}

func NewMessagePool() *MessagePool {
    mp := &MessagePool{
        make(map[string]chan interface{}),
    }
    return mp
}

func (mp *MessagePool) SetupMailFor(id string) error {
    if mp.queue[id] != nil {
        return errors.New("Duplicate queue.")
    }
    mp.queue[id] = make(chan interface{})
    return nil
}
func (mp *MessagePool) Send(to string, msg interface{}) error {
    if mp.queue[to] == nil {
        return errors.New("Not found mail box for " + to)
    }
    go func() {
        mp.queue[to] <- msg
    }()
    return nil
}
func (mp *MessagePool) Receive(from string, timeout int) (msg interface{}, err error) {
    if mp.queue[from] == nil {
        err = errors.New("Not found mail box for " + from)
        return
    }

    if timeout > 0 {
        select {
        case <-time.After(time.Duration(timeout) * time.Millisecond):
            err = errors.New("timeout")
        case msg = <-mp.queue[from]:
        }
    } else if timeout == 0 {
        select {
        case msg = <-mp.queue[from]:
            return
        default:
            err = errors.New("No new message.")
            return
        }
    } else {
        msg = <-mp.queue[from]
    }
    return
}

type Member struct {
    Id     string
    poster *MessagePool
}

func NewManager() (*Member, error) {
    mp := NewMessagePool()
    if err := mp.SetupMailFor("MASTER"); err != nil {
        return nil, err
    }
    return &Member{"MASTER", mp}, nil
}
func (mgr *Member) NewMember(id string) (*Member, error) {
    m := &Member{
        id,
        mgr.poster,
    }
    if err := mgr.poster.SetupMailFor(id); err != nil {
        return m, err
    }
    return m, nil
}
func (m *Member) Broadcast(data interface{}) {
    for k, _ := range m.poster.queue {
        if k != m.Id {
            m.poster.Send(k, data)
        }
    }
}
func (m *Member) Send(to string, data interface{}) {
    m.poster.Send(to, data)
}
func (m *Member) Receive(timeout int) (msg interface{}, err error) {
    msg, err = m.poster.Receive(m.Id, timeout)
    return
}
