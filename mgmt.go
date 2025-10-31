package xtop

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/xconnio/xconn-go"
)

type ManagementAPI struct {
	session    *xconn.Session
	logCache   map[string][]string
	logCacheMu sync.Mutex
}

func NewManagementAPI(session *xconn.Session) *ManagementAPI {
	return &ManagementAPI{
		session:    session,
		logCache:   make(map[string][]string),
		logCacheMu: sync.Mutex{},
	}
}

func (m *ManagementAPI) RequestStats() error {
	resp := m.session.Call(xconn.ManagementProcedureStatsStatusSet).Kwarg("enable", true).Do()
	return resp.Err
}

func (m *ManagementAPI) Realms() ([]string, error) {
	resp := m.session.Call(xconn.ManagementProcedureListRealms).Do()
	if resp.Err != nil {
		return nil, resp.Err
	}

	var realms []string
	for _, realm := range resp.ArgListOr(0, []any{}) {
		realms = append(realms, realm.(string))
	}

	if len(realms) == 0 {
		return nil, fmt.Errorf("could not find realm list in response")
	}

	return realms, nil
}

func (m *ManagementAPI) SessionsCount(realm string) (int, error) {
	resp := m.session.Call(xconn.ManagementProcedureListSession).Arg(realm).Do()
	if resp.Err != nil {
		return 0, resp.Err
	}

	return int(resp.KwargUInt64Or("total", 0)), nil //nolint: gosec
}

func (m *ManagementAPI) SessionDetailsByRealm(realm string) ([]SessionDetails, error) {
	resp := m.session.Call(xconn.ManagementProcedureListSession).Arg(realm).Do()
	if resp.Err != nil {
		return nil, resp.Err
	}

	var sessions []SessionDetails
	if resp.ArgsLen() > 0 {
		data, err := json.Marshal(resp.ArgListOr(0, []any{}))
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
	cacheKey := fmt.Sprintf("%s:%d", realm, sessionID)

	resp := m.session.Call(xconn.ManagementProcedureSessionLogSet).
		Arg(realm).
		Arg(sessionID).
		Kwarg("enable", true).
		Do()

	if resp.Err != nil {
		return fmt.Errorf("failed to enable session logs: %w", resp.Err)
	}

	responseDict := resp.ArgDictOr(0, map[string]any{})
	topic, ok := responseDict["topic"].(string)
	if !ok {
		return fmt.Errorf("could not find topic in response")
	}

	handler := func(event *xconn.Event) {
		for _, a := range event.Args() {
			if msg, ok := a.(string); ok {
				m.logCacheMu.Lock()
				m.logCache[cacheKey] = append(m.logCache[cacheKey], msg)
				m.logCacheMu.Unlock()
				onLog(msg)
			}
		}
	}

	subResp := m.session.Subscribe(topic, handler).Do()
	if subResp.Err != nil {
		return fmt.Errorf("failed to subscribe to session logs: %w", subResp.Err)
	}

	return nil
}
