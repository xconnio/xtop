package xtop

import (
	"encoding/json"
	"fmt"
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
	log.WithField("realms_count", len(realms)).Debug("Realms fetched successfully")
	return realms, nil
}

func (m *ManagementAPI) SessionsCount(realm string) (uint64, error) {
	resp := m.session.Call(xconn.ManagementProcedureListSession).Arg(realm).Do()
	if resp.Err != nil {
		return 0, resp.Err
	}
	count := resp.KwargUInt64Or("total", 0)
	log.WithField("realm", realm).WithField("count", count).Debug("Sessions count fetched")
	return count, nil
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
	log.WithField("realm", realm).WithField("sessions_count", len(sessions)).Debug("Session details fetched successfully")
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
		d, err := ev.ArgDict(0)
		if err != nil {
			log.WithError(err).Debug("Failed to get event dictionary")
			return
		}
		msg, err := d.String("message")
		if err != nil {
			log.WithError(err).Debug("Failed to get message from event")
			return
		}
		onLog(msg)
	}

	sub := m.session.Subscribe(topic, handler).Do()
	if sub.Err != nil {
		return fmt.Errorf("subscribe request timed out")
	}

	go func() {
		<-m.shutdown
		log.WithField("realm", realm).WithField("session_id", sessionID).Debug(
			"Unsubscribing from session logs")
		_ = sub.Unsubscribe()
	}()

	log.WithField("realm", realm).WithField("session_id", sessionID).Debug("Session logs subscription established")
	return nil
}

func (m *ManagementAPI) StopSessionLogs() {
	m.shutdown <- struct{}{}
	resp := m.session.Call(xconn.ManagementProcedureSessionLogSet).
		Kwarg("enable", false).Do()
	if resp.Err != nil {
		log.WithError(resp.Err).Error("Failed to stop session logs")
		return
	}
	log.Debug("Session logs stopped successfully")
}

func (m *ManagementAPI) Close() {
	m.Lock()
	if m.closed {
		m.Unlock()
		return
	}
	m.closed = true
	log.Debug("Closing ManagementAPI")
	close(m.shutdown)
	m.Unlock()
}
