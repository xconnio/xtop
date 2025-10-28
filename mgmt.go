package xtop

import (
	"fmt"

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
