package main

import (
	"projetop2p/internal/chat"
	"projetop2p/internal/localdb"
)

func (a *App) ChatAction(room chat.Room, kind, target, text, emoji string, enabled bool) (string, error) {
	if e := a.initialized(); e != nil {
		return "", e
	}
	return a.core.ChatAction(room, kind, target, text, emoji, enabled)
}
func (a *App) ActivityView(room chat.Room, ids []string, replies bool) (interface{}, error) {
	if e := a.initialized(); e != nil {
		return nil, e
	}
	return a.core.ActivityView(room, ids, replies)
}

func (a *App) NotificationPreferences() (localdb.NotificationPreferences, error) {
	if e := a.initialized(); e != nil {
		return localdb.NotificationPreferences{}, e
	}
	return a.core.NotificationPreferences()
}
func (a *App) SaveNotificationPreferences(p localdb.NotificationPreferences) error {
	if e := a.initialized(); e != nil {
		return e
	}
	return a.core.SaveNotificationPreferences(p)
}

func (a *App) SetChatForeground(visible bool) error {
	if e := a.initialized(); e != nil {
		return e
	}
	a.core.SetChatForeground(visible)
	return nil
}
