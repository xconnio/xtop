package xtop

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/xconnio/xconn-go"
)

type ManagementAPI struct {
	session  *xconn.Session
	shutdown chan struct{}
	closed   bool
	sync.RWMutex
}

func NewManagementAPI(session *xconn.Session) *ManagementAPI {
	return &ManagementAPI{
		session:  session,
		shutdown: make(chan struct{}),
	}
}

func (m *ManagementAPI) RequestStats() error {
	resp := m.session.Call(xconn.ManagementProcedureStatsStatusSet).Kwarg("enable", true).Do()
	return resp.Err
}

func (m *ManagementAPI) Realms() ([]string, error) {
	m.RLock()
	if m.closed {
		m.RUnlock()
		return nil, fmt.Errorf("management API closed")
	}
	m.RUnlock()

	resp := m.session.Call(xconn.ManagementProcedureListRealms).Do()
	if resp.Err != nil {
		return nil, resp.Err
	}

	var realms []string
	for _, r := range resp.ArgListOr(0, xconn.List{}) {
		name, err := r.String()
		if err == nil {
			realms = append(realms, name)
		}
	}
	return realms, nil
}

func (m *ManagementAPI) SessionsCount(realm string) (uint64, error) {
	resp := m.session.Call(xconn.ManagementProcedureListSession).Arg(realm).Do()
	if resp.Err != nil {
		return 0, resp.Err
	}
	return resp.KwargUInt64Or("total", 0), nil
}

func (m *ManagementAPI) SessionDetailsByRealm(realm string) ([]SessionDetails, error) {
	resp := m.session.Call(xconn.ManagementProcedureListSession).Arg(realm).Do()
	if resp.Err != nil {
		return nil, resp.Err
	}
	var sessions []SessionDetails
	if resp.ArgsLen() > 0 {
		list := resp.ArgListOr(0, xconn.List{})
		data, err := json.Marshal(list.Raw())
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &sessions); err != nil {
			return nil, err
		}
	}
	return sessions, nil
}

func (m *ManagementAPI) FetchSessionLogs(realm string, sessionID uint64, onLog func(string)) error {
	m.Lock()
	if m.closed {
		m.Unlock()
		return fmt.Errorf("management API closed")
	}
	m.Unlock()

	resp := m.session.Call(xconn.ManagementProcedureSessionLogSet).
		Arg(realm).Arg(sessionID).Kwarg("enable", true).Do()
	if resp.Err != nil {
		return fmt.Errorf("enable session logs failed: %w", resp.Err)
	}

	topic, err := resp.ArgDictOr(0, xconn.Dict{}).String("topic")
	if err != nil {
		return fmt.Errorf("no topic in response")
	}

	handler := func(ev *xconn.Event) {
		select {
		case <-m.shutdown:
			return
		default:
		}
		d, err := ev.ArgDict(0)
		if err != nil {
			return
		}
		msg, err := d.String("message")
		if err != nil {
			return
		}
		defer func() { _ = recover() }()
		onLog(msg)
	}

	sub := m.session.Subscribe(topic, handler).Do()
	if sub.Err != nil {
		return fmt.Errorf("subscribe request timed out")
	}

	go func() {
		<-m.shutdown
		_ = sub.Unsubscribe()
	}()
	return nil
}

func (m *ManagementAPI) StopSessionLogs() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Recovered from panic: %v", err)
			}
		}()
		resp := m.session.Call(xconn.ManagementProcedureSessionLogSet).
			Kwarg("enable", false).Do()
		if resp.Err != nil {
			return
		}
	}()
}

func (m *ManagementAPI) Close() {
	m.Lock()
	if m.closed {
		m.Unlock()
		return
	}
	m.closed = true
	close(m.shutdown)
	m.Unlock()
}
