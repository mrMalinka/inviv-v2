package main

import (
	"context"

	"github.com/gorilla/websocket"
)

const port = ":12588"

type State struct {
	Conn *websocket.Conn
	Pub  []byte
	Priv []byte

	PeerPubs [][]byte
}

func connectToServer(domain string) {
	domain += port
}

// wails boilerplate
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
