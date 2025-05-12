package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	ecies "github.com/ecies/go/v2"
	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type uuID [16]byte

var (
	LongtermPriv *ecies.PrivateKey
	LongtermPub  *ecies.PublicKey

	PeerPubs      map[uuID]*ecies.PublicKey
	ShorttermPriv *ecies.PrivateKey

	Conn *websocket.Conn
)

const port = ":14194"

const (
	MSG_NewGroupKey uint8 = iota + 1
	MSG_MakeNewKey
	MSG_NewPeerKeys
	MSG_Text
)

type Message struct {
	Type uint8           `json:"type"`
	Data json.RawMessage `json:"data"`
}

type MessageText struct {
	// map of serialized UUID to message encrypted with that UUID's public key
	Contents map[string][]byte `json:"contents"`
}

type MessageTextForward struct {
	// encrypted
	Contents []byte `json:"contents"`
	Sender   uuID   `json:"sender"`
}

type MessageNewGroupKey struct {
	NewGroupKey []byte `json:"newkey"`
}

type MessageNewPeerKeys struct {
	// map of serialized uuID to serialized public key
	Keys map[string]string `json:"keys"`
}

type MessageNewKeyRequestResponse struct {
	SerializedNewKey string `json:"pub"`
}

// -----
// connection
// -----

func (a *App) Connect(host string, makeNew bool, keyStr string) {
	if LongtermPriv == nil || LongtermPub == nil {
		var err error
		LongtermPriv, err = ecies.GenerateKey()
		if err != nil {
			// realistically never happening
			log.Fatalln("Error initializing private key:", err)
		}
		LongtermPub = LongtermPriv.PublicKey
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	header := http.Header{}
	// send it serialized, server will deserialize
	if makeNew {
		header.Add("makenew", "yes")
	} else {
		header.Add("key", keyStr)
	}
	header.Add("longterm", LongtermPub.Hex(false))

	conn, resp, err := dialer.Dial(
		domainPath(host),
		header,
	)
	if err != nil {
		log.Println("Error connecting to WS:", err)
		return
	}
	if resp.StatusCode != 101 {
		log.Fatalf("status code: %v\n", resp.StatusCode)
	}

	Conn = conn
	go receiver(a.ctx)

	runtime.EventsEmit(a.ctx, "connection-change", true)
}
func (a *App) Disconnect() {
	Conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"),
		time.Now().Add(3*time.Millisecond),
	)
	Conn.Close()

	LongtermPriv = nil
	LongtermPub = nil
	ShorttermPriv = nil
	PeerPubs = map[uuID]*ecies.PublicKey{}

	runtime.EventsEmit(a.ctx, "connection-change", false)
}

// main receive function
func receiver(ctx context.Context) {
	for {
		var message Message
		err := Conn.ReadJSON(&message)
		if err != nil {
			log.Printf("Failed to read from WS: '%v'\n", err)
			runtime.EventsEmit(ctx, "connection-change", false)
			return
		}

		switch message.Type {
		case MSG_NewGroupKey:
			var newGroupKey MessageNewGroupKey
			err := json.Unmarshal(message.Data, &newGroupKey)
			if err != nil {
				log.Println("Error unmarshaling new group key:", err)
				continue
			}

			key, err := ecies.Decrypt(LongtermPriv, newGroupKey.NewGroupKey)
			if err != nil {
				log.Println("Error decrypting new group key:", err)
				continue
			}
			runtime.EventsEmit(ctx, "key-update", uuIDToString(uuID(key)))
		case MSG_NewPeerKeys:
			var keysMsg MessageNewPeerKeys
			err := json.Unmarshal(message.Data, &keysMsg)
			if err != nil {
				log.Println("Error unmarshaling new peer key message:", err)
				continue
			}

			PeerPubs = make(map[uuID]*ecies.PublicKey)

			for strUUID, strKey := range keysMsg.Keys {
				deserializedKey, err := deserializePublicKey(string(strKey))
				if err != nil {
					log.Println("Error deserializing peer public key:", err)
					continue
				}
				deserializedUUID, err := stringToUUID(strUUID)
				if err != nil {
					log.Println("Error deserializing peer UUID:", err)
					continue
				}

				PeerPubs[deserializedUUID] = deserializedKey
			}
		case MSG_MakeNewKey:
			ShorttermPriv, err = ecies.GenerateKey()
			if err != nil {
				log.Fatalln(err)
			}

			response := MessageNewKeyRequestResponse{
				SerializedNewKey: serializePublicKey(ShorttermPriv.PublicKey),
			}

			err = Conn.WriteJSON(makeMessage(response, MSG_MakeNewKey))
			if err != nil {
				log.Println("Error sending new public key:", err)
			}
		case MSG_Text:
			var messageText MessageTextForward
			err := json.Unmarshal(message.Data, &messageText)
			if err != nil {
				log.Println("Error unmarshaling text message:", err)
				continue
			}

			plaintext, err := ecies.Decrypt(ShorttermPriv, messageText.Contents)
			if err != nil {
				log.Println("Error decrypting message:", err)
				continue
			}

			runtime.EventsEmit(
				ctx,
				"new-message",
				uuIDToString(messageText.Sender),
				string(plaintext),
				false,
			)
		}
	}
}

func (a *App) SendTextMessage(contents string) {
	msg := new(MessageText)
	msg.Contents = make(map[string][]byte)

	log.Printf("%v", PeerPubs)
	for uuid, key := range PeerPubs {
		ciphertext, err := ecies.Encrypt(key, []byte(contents))
		if err != nil {
			log.Fatalln("Error encrypting message:", err)
		}

		msg.Contents[uuIDToString(uuid)] = ciphertext
	}

	err := Conn.WriteJSON(makeMessage(msg, MSG_Text))
	if err != nil {
		log.Fatalln("Error sending message:", err)
	}

	runtime.EventsEmit(a.ctx, "new-message", "you", contents, true)
}

func makeMessage(msg any, typ uint8) Message {
	embed, err := json.Marshal(msg)
	if err != nil {
		// if this happens, the codes doing something wrong
		log.Fatalln("Error marshaling embed for makeMessage:", err)
	}

	return Message{
		Type: typ,
		Data: embed,
	}
}

// -----
// uuid's, strings, etc
// -----

func serializePublicKey(pub *ecies.PublicKey) string {
	return pub.Hex(true)
}

func deserializePublicKey(encoded string) (*ecies.PublicKey, error) {
	return ecies.NewPublicKeyFromHex(encoded)
}

func uuIDToString(uuid uuID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:],
	)
}
func stringToUUID(s string) (uuID, error) {
	hexStr := ""
	for _, c := range s {
		if c != '-' {
			hexStr += string(c)
		}
	}

	if len(hexStr) != 32 {
		return uuID{}, fmt.Errorf("invalid UUID length")
	}

	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return uuID{}, err
	}

	if decoded[6]&0xf0 != 0x40 {
		return uuID{}, fmt.Errorf("invalid UUID version")
	}
	if decoded[8]&0xc0 != 0x80 {
		return uuID{}, fmt.Errorf("invalid UUID variant")
	}

	var uuid uuID
	copy(uuid[:], decoded)
	return uuid, nil
}

func domainPath(domain string) string {
	if strings.Contains(domain, ":") {
		return fmt.Sprintf("ws://%s/ws", domain)
	}

	return fmt.Sprintf("ws://%s%s/ws", domain, port)
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
