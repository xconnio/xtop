package xtop

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/xconnio/xconn-go"
)

//nolint:gochecknoglobals
var (
	logCache   = make(map[string][]string)
	logCacheMu sync.Mutex
)

func FetchRealms(s *xconn.Session) ([]string, error) {
	resp := s.Call("io.xconn.mgmt.realm.list").Do()
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

func FetchSessions(s *xconn.Session, realm string) (int, error) {
	resp := s.Call("io.xconn.mgmt.session.list").Arg(realm).Do()
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

func FetchSessionDetails(s *xconn.Session, realm string) ([]SessionInfo, error) {
	resp := s.Call("io.xconn.mgmt.session.list").Arg(realm).Do()
	if resp.Err != nil {
		return nil, resp.Err
	}

	var sessions []SessionInfo
	if resp.ArgsLen() > 0 {
		for _, v := range resp.ArgListOr(0, []any{}) {
			if m, ok := v.(map[string]any); ok {
				var si SessionInfo
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

func FetchSessionLogs(s *xconn.Session, realm string, sessionID uint64, onLog func(string)) error {
	cacheKey := fmt.Sprintf("%s:%d", realm, sessionID)

	resp := s.Call("io.xconn.mgmt.session.log.set").
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
				logCacheMu.Lock()
				logCache[cacheKey] = append(logCache[cacheKey], msg)
				logCacheMu.Unlock()
				onLog(msg)
			}
		}
	}

	subResp := s.Subscribe(topic, handler).Do()
	if subResp.Err != nil {
		return fmt.Errorf("failed to subscribe to session logs: %w", subResp.Err)
	}

	return nil
}
