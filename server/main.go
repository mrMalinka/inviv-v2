package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"slices"

	ecies "github.com/ecies/go/v2"
	"github.com/gorilla/websocket"
)

const rotateKeysEvery uint8 = 8

var debug = false

func init() {
	if len(os.Args) == 1 {
		return
	}
	if os.Args[1] == "debug" {
		log.Println("Debug on")
		debug = true
	}
}

// -----
// structure
// -----

type uuID [16]byte

type Member struct {
	Name     uuID // only ever used for the users to know who's sending a message
	Conn     *websocket.Conn
	Longterm *ecies.PublicKey // only ever used to send the a new group key (the key used for inviting)

	Shortterm       *ecies.PublicKey
	ShorttermUpdate chan *ecies.PublicKey
}

type Group struct {
	Key     uuID
	Members []*Member
	Counter uint8
}

var (
	groups   []*Group
	groupsMu sync.Mutex
)

// -----
// inviting system
// -----

func authGroup(key uuID) *Group {
	groupsMu.Lock()
	defer groupsMu.Unlock()

	for _, group := range groups {
		if group.Key == key {
			return group
		}
	}
	return nil
}

// generates, switches, encrypts and sends to all members a new invite key for the group
func (g *Group) RotateGroupKey() {
	if debug {
		log.Printf("Rotating group key (%d members)\n", len(g.Members))
	}
	new := generateUUID()
	g.Key = new

	for _, member := range g.Members {
		enc, err := ecies.Encrypt(
			member.Longterm,
			new[:],
		)
		if err != nil {
			if debug {
				log.Println("Invite key encryption error:", err)
			}
			// nuke the member because of their bad key
			member.Nuke()
			continue
		}

		message := MessageNewGroupKey{
			NewGroupKey: enc,
		}
		member.Conn.WriteJSON(makeMessage(
			message, MSG_NewGroupKey,
		))
	}
}

func (m *Member) Nuke() {
	if debug {
		log.Println("Nuking member " + uuidToString(m.Name))
	}

	groupsMu.Lock()
	defer groupsMu.Unlock()

	if m.Conn != nil {
		defer m.Conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "nuked"),
			time.Now().Add(100*time.Millisecond),
		)
		defer m.Conn.Close()
	}

	var myGroup *Group
	// find the group this member is in
	for _, group := range groups {
		for _, member := range group.Members {
			if member.Name == m.Name {
				myGroup = group
				break
			}
		}
		if myGroup != nil {
			break
		}
	}

	if myGroup == nil {
		if debug {
			log.Println("A user was found with no group!")
		}
		return
	}

	for i, member := range myGroup.Members {
		if member == nil {
			if debug {
				log.Printf(
					"WARNING: nil member found at %d in group with key %s\n",
					i, uuidToString(myGroup.Key),
				)
			}
			// delete the nil
			myGroup.Members = slices.Delete(myGroup.Members, i, i+1)
			continue
		}
		if member.Name == m.Name {
			myGroup.Members = slices.Delete(myGroup.Members, i, i+1)
		}
	}

	// nuke the group
	if len(myGroup.Members) == 0 {
		for i, g := range groups {
			if g.Key != myGroup.Key {
				continue
			}

			if debug {
				log.Println("Nuking group")
			}
			groups = slices.Delete(groups, i, i+1)
			break
		}
	}
}

// -----
// main message system
// -----

func packetForMember(m *Member, list map[string]string) map[string]string {
	newList := make(map[string]string)
	for uuid, key := range list {
		if uuid != uuidToString(m.Name) {
			newList[uuid] = key
		}
	}

	return newList
}
func (g *Group) DoRekey() {
	const rekeyTimeout time.Duration = 3 * time.Second

	if debug {
		log.Printf("Rotating member encryption keys (%d members)\n", len(g.Members))
	}
	time.Sleep(1 * time.Millisecond)
	var wg sync.WaitGroup

	for i, member := range g.Members {
		notif := Message{
			Type: MSG_MakeNewKey,
		}

		err := member.Conn.WriteJSON(notif)
		if err != nil {
			if debug {
				log.Println("Error asking user to make a new key:", err)
			}
			member.Nuke()
			continue
		}

		wg.Add(1)
		go func(m *Member) {
			defer wg.Done()
			select {
			case newKey := <-m.ShorttermUpdate:
				if debug {
					log.Println("Received key from " + uuidToString(m.Name))
				}
				m.Shortterm = newKey
			case <-time.After(rekeyTimeout):
				if debug {
					log.Printf("Member %v timed out during rekey!\n", i)
				}
				m.Nuke()
			}
		}(member)
	}

	wg.Wait()

	list := make(map[string]string)
	for _, member := range g.Members {
		list[uuidToString(member.Name)] = serializePublicKey(member.Shortterm)
	}

	for _, member := range g.Members {
		packet := MessageNewPeerKeys{
			packetForMember(member, list),
		}
		if debug {
			log.Printf("Sending %d key(s) to %s\n", len(packet.Keys), uuidToString(member.Name))
		}
		member.Conn.WriteJSON(makeMessage(packet, MSG_NewPeerKeys))
	}
}

// -----
// raw messages
// -----

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
	// encrypted using a members ecies pubkey
	NewGroupKey []byte `json:"newkey"`
}

type MessageNewPeerKeys struct {
	// map of serialized uuID to serialized public key
	Keys map[string]string `json:"keys"`
}

type MessageNewKeyRequestResponse struct {
	SerializedNewKey string `json:"pub"`
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
// main
// -----

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// main function
func main() {
	const port string = ":14194"

	log.Printf("Starting server on %v", port)

	http.HandleFunc("/ws", handleWebSocket)
	log.Fatalln(http.ListenAndServe(port, nil))
}

// deployed for each user
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	new := r.Header.Get("makenew") == "yes"

	var key uuID
	if new {
		// make a completely new group and initialize it with a new key
		key = generateUUID()
		newGroup := Group{
			Key:     key,
			Members: make([]*Member, 0),
			Counter: 0,
		}
		groupsMu.Lock()
		groups = append(groups, &newGroup)
		groupsMu.Unlock()
	} else {
		// try to get the access key from the header
		var err error
		// will be serialized, so deserialize it
		key, err = stringToUUID(r.Header.Get("key"))
		if err != nil {
			if debug {
				log.Println("Incorrect group key format sent")
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	// get the long term key used for transmitting new group invite keys
	myLongterm, err := ecies.NewPublicKeyFromHex(r.Header.Get("longterm"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// try to access the group
	// (hopefully) will always succeed if the group was created by this user
	myGroup := authGroup(key)
	if myGroup == nil {
		if debug {
			log.Println("Incorrect group key sent")
		}
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// upgrade the connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if debug {
			log.Println("Upgrade failed:", err)
		}
		return
	}
	defer conn.Close()

	// add the new member to the group
	myuuID := generateUUID()
	me := &Member{
		Name: myuuID,
		Conn: conn,

		Longterm: myLongterm,

		// shortterm will be filled by myGroup.DoRekey() soon
		Shortterm:       nil,
		ShorttermUpdate: make(chan *ecies.PublicKey),
	}
	groupsMu.Lock()
	myGroup.Members = append(myGroup.Members, me)
	groupsMu.Unlock()

	defer me.Nuke()

	myGroup.RotateGroupKey()

	go myGroup.DoRekey()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if debug {
				log.Println("Error reading from WS:", err)
			}
			return
		}

		switch msg.Type {
		case MSG_MakeNewKey:
			// the response of the client to a MSG_NewKey (for key rotation)
			// sent by Group.DoRekey()
			var response MessageNewKeyRequestResponse
			err := json.Unmarshal(msg.Data, &response)
			if err != nil {
				if debug {
					log.Println("Error unmarshaling key request response:", err)
				}
				return
			}

			newKey, err := deserializePublicKey(response.SerializedNewKey)
			if err != nil {
				if debug {
					log.Println("Error deserializing new public key:", err)
				}
				return
			}
			me.ShorttermUpdate <- newKey
		case MSG_Text:
			var textMessage MessageText
			err := json.Unmarshal(msg.Data, &textMessage)
			if err != nil {
				if debug {
					log.Println("Error unmarshaling text message:", err)
				}
				return
			}

			for strUUID, contents := range textMessage.Contents {
				uuid, err := stringToUUID(strUUID)
				if err != nil {
					if debug {
						log.Println("Someone sent a message with an invalid peer UUID!")
					}
					return
				}

				for _, member := range myGroup.Members {
					if member.Name == me.Name || member.Name != uuid {
						continue
					}

					toForward := MessageTextForward{
						Contents: contents,
						Sender:   me.Name,
					}

					if debug {
						log.Println("Sending to " + uuidToString(member.Name))
					}

					member.Conn.WriteJSON(makeMessage(
						toForward, MSG_Text,
					)) // forward directly
				}
			}

			myGroup.Counter++
			if myGroup.Counter >= rotateKeysEvery {
				go myGroup.DoRekey()
				myGroup.Counter = 0
			}
		}
	}
}

// -----
// working with strings, uuid, etc
// -----

func serializePublicKey(pub *ecies.PublicKey) string {
	return pub.Hex(true)
}

func deserializePublicKey(encoded string) (*ecies.PublicKey, error) {
	return ecies.NewPublicKeyFromHex(encoded)
}

func generateUUID() uuID {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		// realistically this is never happening
		log.Fatalln(err)
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return uuID(uuid)
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

func uuidToString(uuid uuID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:],
	)
}
