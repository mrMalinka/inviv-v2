package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/curve25519"
)

const port string = ":12588"

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type MessageText struct {
	Contents string `json:"contents"`
}
type MessageReceiveText struct {
	Contents string `json:"contents"`
	SenderID []byte `json:"sender"`
}

type MessageCreateNewGC struct {
	HostPub [32]byte `json:"hostpub"`
}
type MessageJoinGC struct {
}

type State struct {
	Conn *websocket.Conn
	Pub  *[32]byte
	Priv *[32]byte
}

var state State

func (a *App) Connect(host string, makeNew bool) {
	url := fmt.Sprintf("ws://%s%s/ws", host, port)

	var err error
	state.Conn, _, err = websocket.DefaultDialer.Dial(url, nil)
	if !a.handleError(err) {
		go a.receiver()
	}

	// initialize
	rand.Read(state.Priv[:]) // never expose this
	curve25519.ScalarBaseMult(state.Pub, state.Priv)

	if makeNew {
		newGCRequest := MessageCreateNewGC{
			HostPub: *state.Pub,
		}

		err := state.Conn.WriteJSON(newGCRequest)
		a.handleError(err)
	} else {
		log.Fatalln("not implemented")
	}
}

func (a *App) receiver() {
	defer state.Conn.Close()

	for {
		//err := state.Conn.ReadJSON(&message)
	}
}

func (a *App) SendMessage(contents string) {
	toSend := MessageText{Contents: contents}

	err := state.Conn.WriteJSON(toSend) // FIXME
	a.handleError(err)
}

func (a *App) handleError(err error) bool {
	if err != nil {
		fmt.Printf("error: '%v'\n", err.Error())
		runtime.WindowExecJS(a.ctx, fmt.Sprintf("displayError(%s)", err.Error()))
		return true
	}
	return false
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
