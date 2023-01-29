package controller

import "context"

type WatcherManager struct {
}

func (m *WatcherManager) Start(context.Context) error {
	return nil
}
func (m *WatcherManager) NeedLeaderElection() bool {
	return true
}
