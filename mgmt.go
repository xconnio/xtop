package xtop

import (
	"encoding/json"
	"fmt"
	"log"
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

	for _, arg := range resp.Args() {
		if realms, ok := arg.([]string); ok {
			return realms, nil
		}

		if raw, ok := arg.([]interface{}); ok {
			var realms []string
			for _, item := range raw {
				if str, ok := item.(string); ok {
					realms = append(realms, str)
				}
			}
			if len(realms) > 0 {
				return realms, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find realm list in response")
}

func (m *ManagementAPI) SessionsCount(realm string) (int, error) {
	resp := m.session.Call(xconn.ManagementProcedureListSession).Arg(realm).Do()
	if resp.Err != nil {
		return 0, resp.Err
	}

	var total int
	if len(resp.Args()) > 0 {
		if sessions, ok := resp.Args()[0].([]interface{}); ok {
			total = len(sessions)
		}
	}

	if len(resp.Args()) > 1 {
		if kw, ok := resp.Args()[1].(map[string]any); ok {
			if t, ok := kw["total"]; ok {
				if f, ok := t.(float64); ok {
					total = int(f)
				}
			}
		}
	}

	return total, nil
}

func (m *ManagementAPI) SessionDetailsByRealm(realm string) ([]SessionDetails, error) {
	resp := m.session.Call(xconn.ManagementProcedureListSession).Arg(realm).Do()
	if resp.Err != nil {
		return nil, resp.Err
	}

	var sessions []SessionDetails
	if resp.ArgsLen() > 0 {
		for _, v := range resp.ArgListOr(0, []any{}) {
			if m, ok := v.(map[string]any); ok {
				var si SessionDetails
				b, err := json.Marshal(m)
				if err != nil {
					log.Printf("Error marshaling session info: %v", err)
					continue
				}
				err = json.Unmarshal(b, &si)
				if err != nil {
					log.Printf("Error unmarshaling session info: %v", err)
					continue
				}
				sessions = append(sessions, si)
			} else {
				log.Printf("Unexpected type: %T", v)
			}
		}
	}

	return sessions, nil
}

func (m *ManagementAPI) FetchSessionLogs(realm string, sessionID uint64, onLog func(string)) error {
	cacheKey := fmt.Sprintf("%s:%d", realm, sessionID)

	resp := m.session.Call(xconn.ManagementProcedureSessionLogSet).
		Args(realm, sessionID).
		Kwarg("enable", true).
		Do()

	if resp.Err != nil {
		return fmt.Errorf("failed to enable session logs: %w", resp.Err)
	}

	var topic string
	for _, arg := range resp.Args() {
		if m, ok := arg.(map[string]any); ok {
			if t, ok := m["topic"].(string); ok {
				topic = t
				break
			}
		}
	}

	if topic == "" {
		return fmt.Errorf("no log topic found in response")
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
