package xtop

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/xconnio/xconn-go"
)

type ManagementAPI struct {
	session    *xconn.Session
	logger     *logrus.Logger
	activeLogs map[string]bool
	mu         sync.Mutex
}

func NewManagementAPI(session *xconn.Session) *ManagementAPI {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		ForceColors:      false,
	})
	file, _ := os.OpenFile("/tmp/xtop.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	logger.SetOutput(file)

	return &ManagementAPI{
		session:    session,
		logger:     logger,
		activeLogs: make(map[string]bool),
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

	return int(resp.KwargUInt64Or("total", 0)), nil //nolint:gosec
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
	logFile := "/tmp/xtop.log"
	_ = os.Remove(logFile)

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	m.logger.SetOutput(file)

	resp := m.session.Call(xconn.ManagementProcedureSessionLogSet).
		Arg(realm).
		Arg(sessionID).
		Kwarg("enable", true).
		Do()

	if resp.Err != nil {
		file.Close()
		os.Remove(logFile)
		return fmt.Errorf("failed to enable session logs: %w", resp.Err)
	}

	responseDict := resp.ArgDictOr(0, map[string]any{})
	topic, ok := responseDict["topic"].(string)
	if !ok {
		file.Close()
		os.Remove(logFile)
		return fmt.Errorf("could not find topic in response")
	}

	m.mu.Lock()
	m.activeLogs[logFile] = true
	m.mu.Unlock()

	handler := func(event *xconn.Event) {
		for _, a := range event.Args() {
			msg, ok := a.(string)
			if !ok {
				continue
			}
			m.logger.Info(msg)
			onLog(msg)
		}
	}

	subResp := m.session.Subscribe(topic, handler).Do()
	if subResp.Err != nil {
		file.Close()
		os.Remove(logFile)
		return fmt.Errorf("failed to subscribe to session logs: %w", subResp.Err)
	}

	return nil
}
