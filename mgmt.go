package xtop

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/xconnio/xconn-go"
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
