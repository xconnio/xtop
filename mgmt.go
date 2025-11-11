package xtop

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/xconnio/xconn-go"
)

type ManagementAPI struct {
	session      *xconn.Session
	logCache     map[string][]string
	subscription xconn.SubscribeResponse
	shutdown     chan struct{}
	closed       bool

	sync.RWMutex
}

func NewManagementAPI(session *xconn.Session) *ManagementAPI {
	return &ManagementAPI{
		session:  session,
		logCache: make(map[string][]string),
		shutdown: make(chan struct{}),
	}
}

func (m *ManagementAPI) RequestStats() error {
	resp := m.session.Call(xconn.ManagementProcedureStatsStatusSet).Kwarg("enable", true).Do()
	return resp.Err
}

func (m *ManagementAPI) Realms() ([]string, error) {
	m.RLock()
	closed := m.closed
	m.RUnlock()

	if closed {
		return nil, fmt.Errorf("management API is closed")
	}

	resp := m.session.Call(xconn.ManagementProcedureListRealms).Do()
	if resp.Err != nil {
		return nil, resp.Err
	}

	var realms []string
	for _, realm := range resp.ArgListOr(0, xconn.List{}) {
		rlm, err := realm.String()
		if err == nil {
			realms = append(realms, rlm)
		}
	}

	if len(realms) == 0 {
		return nil, fmt.Errorf("could not find realm list in response")
	}

	return realms, nil
}

func (m *ManagementAPI) SessionsCount(realm string) (uint64, error) {
	m.RLock()
	closed := m.closed
	m.RUnlock()

	if closed {
		return 0, fmt.Errorf("management API is closed")
	}

	resp := m.session.Call(xconn.ManagementProcedureListSession).Arg(realm).Do()
	if resp.Err != nil {
		return 0, resp.Err
	}

	return resp.KwargUInt64Or("total", 0), nil
}

func (m *ManagementAPI) SessionDetailsByRealm(realm string) ([]SessionDetails, error) {
	m.RLock()
	closed := m.closed
	m.RUnlock()

	if closed {
		return nil, fmt.Errorf("management API is closed")
	}

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
		return fmt.Errorf("management API is closed")
	}

	cacheKey := fmt.Sprintf("%s:%d", realm, sessionID)

	resp := m.session.Call(xconn.ManagementProcedureSessionLogSet).
		Arg(realm).
		Arg(sessionID).
		Kwarg("enable", true).
		Do()

	if resp.Err != nil {
		m.Unlock()
		return fmt.Errorf("failed to enable session logs: %w", resp.Err)
	}

	responseDict := resp.ArgDictOr(0, xconn.Dict{})
	topic, err := responseDict.String("topic")
	if err != nil {
		m.Unlock()
		return fmt.Errorf("could not find topic in response")
	}

	handler := func(event *xconn.Event) {
		select {
		case <-m.shutdown:
			return
		default:
		}

		eventMap, err := event.ArgDict(0)
		if err == nil {
			msg, err := eventMap.String("message")
			if err == nil {
				m.Lock()
				if len(m.logCache[cacheKey]) >= 2000 {
					m.logCache[cacheKey] = m.logCache[cacheKey][1:]
				}
				m.logCache[cacheKey] = append(m.logCache[cacheKey], msg)
				m.Unlock()
				onLog(msg)
			}
		}
	}

	subResp := m.session.Subscribe(topic, handler).Do()
	if subResp.Err != nil {
		m.Unlock()
		return fmt.Errorf("failed to subscribe to session logs: %w", subResp.Err)
	}

	m.subscription = subResp
	m.Unlock()
	return nil
}

func (m *ManagementAPI) StopSessionLogs() {
	m.Lock()
	sub := m.subscription
	closed := m.closed
	m.Unlock()

	if !closed {
		_ = sub.Unsubscribe()
	}
}

func (m *ManagementAPI) Close() {
	m.Lock()
	if m.closed {
		m.Unlock()
		return
	}
	m.closed = true
	close(m.shutdown)
	sub := m.subscription
	m.logCache = make(map[string][]string)
	m.Unlock()

	if !m.closed {
		_ = sub.Unsubscribe()
	}
}
