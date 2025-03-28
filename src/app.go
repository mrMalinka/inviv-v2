package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const port = ":12588"

type State struct {
	Conn *websocket.Conn
	Pub  []byte
	Priv []byte

	PeerPubs [][]byte
}

var state State

func connectToServer(domain string) {
	domain += port

	conn, _, err := websocket.DefaultDialer.Dial(domain, nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer conn.Close()
}

func (a *App) HandleError(err error) {
	if err != nil {
		runtime.WindowExecJS(a.ctx, fmt.Sprintf("displayError(%s)", err.Error()))
	}
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
