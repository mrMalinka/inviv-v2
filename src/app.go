package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	ecies "github.com/ecies/go/v2"
	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type UUID [16]byte

var (
	LongtermPriv *ecies.PrivateKey
	LongtermPub  *ecies.PublicKey

	ShorttermPriv *ecdh.PrivateKey
	SharedKey     [32]byte

	Conn *websocket.Conn
)

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
	// encrypted
	Contents []byte `json:"contents"`
}

type MessageTextForward struct {
	// encrypted
	Contents []byte `json:"contents"`
	Sender   UUID   `json:"sender"`
}

type MessageNewGroupKey struct {
	NewGroupKey []byte `json:"newkey"`
}

type MessageNewPeerKeys struct {
	Keys []string `json:"keys"`
}

type MessageNewKeyRequestResponse struct {
	SerializedNewKey string `json:"pub"`
}

// -----
// connection
// -----

func (a *App) Connect(makeNew bool, keyStr string) {
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
		"ws://localhost:14194/ws",
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
}

// main receive function
func receiver(ctx context.Context) {
	for {
		var message Message
		err := Conn.ReadJSON(&message)
		if err != nil {
			log.Printf("Failed to read from WS: '%v'\n", err)
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
			runtime.EventsEmit(ctx, "key-update", UUIDToString(UUID(key)))
		case MSG_NewPeerKeys:
			var keysMsg MessageNewPeerKeys
			err := json.Unmarshal(message.Data, &keysMsg)
			if err != nil {
				log.Println("Error unmarshaling new peer key message:", err)
				continue
			}

			var list []*ecdh.PublicKey
			for _, serializedKey := range keysMsg.Keys {
				deserializedKey, err := deserializePublicKey(serializedKey, ecdh.X25519())
				if err != nil {
					// if this happens, our "shared" secret would be out of sync so panic anyways
					log.Fatalln("Error deserializing peer public key:", err)
				}
				list = append(list, deserializedKey)
			}

			newKey, err := computeSharedKey(ShorttermPriv, list)
			if err != nil {
				log.Fatalln("Error computing shared key:", err)
			}

			SharedKey = newKey
		case MSG_MakeNewKey:
			ShorttermPriv, err = ecdh.X25519().GenerateKey(rand.Reader)
			if err != nil {
				log.Fatalln(err)
			}

			response := MessageNewKeyRequestResponse{
				SerializedNewKey: serializePublicKey(ShorttermPriv.PublicKey()),
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

			SharedKey = sha256.Sum256(SharedKey[:]) // ratchet forward
			plaintext, err := decrypt(messageText.Contents, SharedKey[:])
			if err != nil {
				log.Fatalln("Error decrypting message:", err)
			}

			runtime.EventsEmit(
				ctx,
				"new-message",
				UUIDToString(messageText.Sender),
				string(plaintext),
				false,
			)
		}
	}
}

func (a *App) SendTextMessage(contents string) {
	// ratchet the key forward
	SharedKey = sha256.Sum256(SharedKey[:])

	// encrypt
	ciphertext, err := encrypt([]byte(contents), SharedKey[:])
	if err != nil {
		log.Fatalln("Error encrypting message:", err)
	}

	msg := MessageText{
		Contents: ciphertext,
	}

	err = Conn.WriteJSON(makeMessage(msg, MSG_Text))
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
// crypto
// -----

func computeSharedKey(localPrivateKey *ecdh.PrivateKey, peerPublicKeys []*ecdh.PublicKey) ([32]byte, error) {
	// sort first
	sortedKeys := make([]*ecdh.PublicKey, len(peerPublicKeys))
	copy(sortedKeys, peerPublicKeys)
	sort.Slice(sortedKeys, func(i, j int) bool {
		return bytes.Compare(sortedKeys[i].Bytes(), sortedKeys[j].Bytes()) < 0
	})

	localPublicKey := localPrivateKey.PublicKey()

	allKeys := append(sortedKeys, localPublicKey)
	sort.Slice(allKeys, func(i, j int) bool {
		return bytes.Compare(allKeys[i].Bytes(), allKeys[j].Bytes()) < 0
	})

	combined := make([]byte, 0, 32*len(allKeys))
	for _, pubKey := range allKeys {
		if !bytes.Equal(pubKey.Bytes(), localPublicKey.Bytes()) {
			secret, err := localPrivateKey.ECDH(pubKey)
			if err != nil {
				return [32]byte{}, err
			}
			combined = append(combined, secret...)
		}
	}

	groupKey := sha256.Sum256(combined)
	return groupKey, nil
}

func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// -----
// uuid's, strings, etc
// -----

func serializePublicKey(pub *ecdh.PublicKey) string {
	return base64.StdEncoding.EncodeToString(pub.Bytes())
}

func deserializePublicKey(encoded string, curve ecdh.Curve) (*ecdh.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	return curve.NewPublicKey(decoded)
}

func UUIDToString(uuid UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:],
	)
}

func stringToUUID(s string) (UUID, error) {
	hexStr := ""
	for _, c := range s {
		if c != '-' {
			hexStr += string(c)
		}
	}

	if len(hexStr) != 32 {
		return UUID{}, fmt.Errorf("invalid UUID length")
	}

	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return UUID{}, err
	}

	// Validate version and variant
	if decoded[6]&0xf0 != 0x40 {
		return UUID{}, fmt.Errorf("invalid UUID version")
	}
	if decoded[8]&0xc0 != 0x80 {
		return UUID{}, fmt.Errorf("invalid UUID variant")
	}

	// Convert slice to UUID ([16]byte)
	var uuid UUID
	copy(uuid[:], decoded)
	return uuid, nil
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
