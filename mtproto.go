package mtproto

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/k0kubun/pp"
)

var acc int64

const (
	appId             = 45139
	appHash           = "7e55cea996fe1d94d6d22105258e3579"
	defaultServerAddr = "149.154.167.40:443" // Test
//	defaultServerAddr = "149.154.167.50:443" // Production
)

type MTProto struct {
	ipv6      bool
	addr      string
	conn      *net.TCPConn
	f         *os.File
	connected bool
	queueSend chan packetToSend
	stopRead  chan struct{}
	stopPing  chan struct{}

	authKey     []byte
	authKeyHash []byte
	serverSalt  []byte
	encrypted   bool
	sessionId   int64

	mutex        *sync.Mutex
	lastSeqNo    int32
	msgsIdToAck  map[int64]packetToSend
	msgsIdToResp map[int64]chan TL
	seqNo        int32
	msgId        int64

	dclist       map[int32]string
	accessHashes map[int32]int64
}

type packetToSend struct {
	msg  TL
	resp chan TL
}

func NewMTProto(authkeyfile string, ipv6 bool) (*MTProto, error) {
	var err error
	m := new(MTProto)
	m.ipv6 = ipv6

	m.f, err = os.OpenFile(authkeyfile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	err = m.readData()
	if err == nil {
		m.encrypted = true
	} else {
		m.addr = defaultServerAddr
		m.encrypted = false
	}
	rand.Seed(time.Now().UnixNano())
	m.sessionId = rand.Int63()
	m.connected = false
	m.accessHashes = map[int32]int64{}
	return m, nil
}

func (m *MTProto) Connect() error {
	var err error
	var tcpAddr *net.TCPAddr
	fmt.Println("Connecting: ", m.addr, m.encrypted)

	// connect
	tcpAddr, err = net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return err
	}

	// TODO Make this better
	proxy := os.Getenv("socks5_proxy")

	if proxy != "" {
		var d net.Dialer
		socks5, err := SOCKS5("tcp", proxy, nil, d)
		if err != nil {
			return err
		}
		conn, err := socks5.Dial("tcp", tcpAddr.String())
		if err != nil {
			return err
		}
		m.conn = conn.(*net.TCPConn)
	} else {
		m.conn, err = net.DialTCP("tcp", nil, tcpAddr)
	}

	_, err = m.conn.Write([]byte{0xef})
	if err != nil {
		return err
	}

	// get new authKey if need
	if !m.encrypted {
		err = m.makeAuthKey()
		if err != nil {
			return err
		}
	}

	// start goroutines
	m.queueSend = make(chan packetToSend, 64)
	m.stopRead = make(chan struct{}, 1)
	m.stopPing = make(chan struct{}, 1)
	m.msgsIdToAck = make(map[int64]packetToSend)
	m.msgsIdToResp = make(map[int64]chan TL)
	m.mutex = &sync.Mutex{}
	go m.SendRoutine()
	go m.ReadRoutine(m.stopRead)

	var resp chan TL
	var x TL

	// (help_getConfig)
	resp = make(chan TL, 1)
	m.queueSend <- packetToSend{
		TL_invokeWithLayer{
			layer,
			TL_initConnection{
				appId,
				"Unknown",
				runtime.GOOS + "/" + runtime.GOARCH,
				"0.0.3",
				"en",
				TL_help_getConfig{},
			},
		},
		resp,
	}
	x = <-resp
	switch x.(type) {
	case TL_config:
		fmt.Printf("%#v\n", x.(TL_config))
		m.dclist = make(map[int32]string, 5)
		for _, v := range x.(TL_config).dc_options {
			if v.ipv6 == m.ipv6 {
				m.dclist[v.id] = fmt.Sprintf("%s:%d", v.ip_address, v.port)
			}
		}
	default:
		return fmt.Errorf("Got: %T", x)
	}

	// start keepalive pinging
	m.startPing()
	m.connected = true
	return nil
}

func (m *MTProto) Reconnect(newaddr string) error {
	var err error
	//fmt.Println("Reconnecting: ", newaddr)
	// stop ping routine
	m.stopPing <- struct{}{}
	close(m.stopPing)

	// close send routine & close connection
	close(m.queueSend)
	err = m.conn.Close()
	if err != nil {
		return err
	}
	m.connected = false
	// stop read routine
	m.stopRead <- struct{}{}
	close(m.stopRead)

	// renew connection
	m.encrypted = false
	m.addr = newaddr
	err = m.Connect()
	return err
}

func (m *MTProto) Auth(phonenumber string) error {
	var authSentCode TL_auth_sentCode
	// (TL_auth_sendCode)
	flag := true
	for flag {
		resp := make(chan TL, 1)
		m.queueSend <- packetToSend{TL_auth_sendCode{phonenumber, 0, appId, appHash, "en"}, resp}
		x := <-resp
		switch x.(type) {
		case TL_auth_sentCode:
			authSentCode = x.(TL_auth_sentCode)
			flag = false
		case TL_rpc_error:
			x := x.(TL_rpc_error)
			if x.error_code != 303 {
				return fmt.Errorf("RPC error_code: %d, %s", x.error_code, x.error_message)
			}
			var newDc int32
			n, _ := fmt.Sscanf(x.error_message, "PHONE_MIGRATE_%d", &newDc)
			if n != 1 {
				n, _ := fmt.Sscanf(x.error_message, "NETWORK_MIGRATE_%d", &newDc)
				if n != 1 {
					return fmt.Errorf("RPC error_string: %s", x.error_message)
				}
			}

			newDcAddr, ok := m.dclist[newDc]
			if !ok {
				return fmt.Errorf("Wrong DC index: %d", newDc)
			}
			err := m.Reconnect(newDcAddr)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Got: %T", x)
		}

	}

	var code int

	fmt.Print("Enter code: ")
	fmt.Scanf("%d", &code)

	if authSentCode.phone_registered {
		resp := make(chan TL, 1)
		m.queueSend <- packetToSend{
			TL_auth_signIn{phonenumber, authSentCode.phone_code_hash, fmt.Sprintf("%d", code)},
			resp,
		}
		x := <-resp
		auth, ok := x.(TL_auth_authorization)
		if !ok {
			return fmt.Errorf("RPC: %#v", x)
		}
		userSelf := auth.user.(TL_user)
		fmt.Printf("Signed in: id %d name <%s %s>\n", userSelf.id, userSelf.first_name, userSelf.last_name)

	} else {

		return fmt.Errorf("Cannot sign up yet")
	}

	return nil
}

func (m *MTProto) GetContacts() error {
	resp := make(chan TL, 1)
	m.queueSend <- packetToSend{TL_contacts_getContacts{""}, resp}
	x := <-resp
	list, ok := x.(TL_contacts_contacts)
	if !ok {
		return fmt.Errorf("RPC: %#v", x)
	}

	fmt.Printf(
		"\033[33m\033[1m%10s    %10s    %-30s    %-20s\033[0m\n",
		"id", "mutual", "name", "username",
	)

	for _, v := range list.users {
		switch v.(type) {
		case TL_user:
			v := v.(TL_user)
			fmt.Printf(
				"%10d    %10t    %-30s    %-20s\n",
				v.id,
				v.mutual_contact,
				fmt.Sprintf("%s %s", v.first_name, v.last_name),
				v.username,
			)
			m.accessHashes[v.id] = v.access_hash
		}
	}

	return nil
}

func (m *MTProto) GetDialogs() error {
	resp := make(chan TL, 1)
	m.queueSend <- packetToSend{TL_messages_getDialogs{0, 0, TL_inputPeerSelf{}, 0}, resp}
	x := <-resp
	list, ok := x.(TL_messages_dialogsSlice)

	if !ok {
		return fmt.Errorf("RPC: %#v", x)
	}

	fmt.Printf(
		"\033[33m\033[1m%10s    %10s    %-10s    %-5s	%-20s\033[0m\n",
		"id", "type", "top_message", "unread_count", "title",
	)

	t := ""
	i := int32(0)
	title := ""
	chat_idx := 0
	user_idx := 0
	for _, v := range list.dialogs {
		switch v.(type) {
		case TL_dialog:
			v := v.(TL_dialog)
			switch v.peer.(type) {
			case TL_peerUser:
				t = "User"
				i = v.peer.(TL_peerUser).user_id
				switch list.users[user_idx].(type) {
				case TL_user:
					u := list.users[user_idx].(TL_user)
					title = fmt.Sprintf("%s %s(%s)", u.first_name, u.last_name, u.username)
					m.accessHashes[u.id] = u.access_hash
				}
				user_idx = user_idx + 1
			case TL_peerChat:
				t = "Chat"
				i = v.peer.(TL_peerChat).chat_id
				switch list.chats[chat_idx].(type) {
				case TL_chatEmpty:
					title = "Empty"
				case TL_chat:
					title = list.chats[chat_idx].(TL_chat).title
				case TL_chatForbidden:
					title = list.chats[chat_idx].(TL_chatForbidden).title
				}
				chat_idx = chat_idx + 1
			}
			fmt.Printf(
				"%10d	%8s	%-10d	%-5d	%-20s\n",
				i, t, v.top_message, v.unread_count, title,
			)
		case TL_dialogChannel:
			v := v.(TL_dialogChannel)
			t = "Channel"
			i = v.peer.(TL_peerChannel).channel_id
			switch list.chats[chat_idx].(type) {
			case TL_chatEmpty:
				title = "Empty"
			case TL_channel:
				ch := list.chats[chat_idx].(TL_channel)
				title = ch.title
				m.accessHashes[ch.id] = ch.access_hash
			case TL_channelForbidden:
				ch := list.chats[chat_idx].(TL_channelForbidden)
				title = ch.title
				m.accessHashes[ch.id] = ch.access_hash
			}
			chat_idx = chat_idx + 1
			fmt.Printf(
				"%10d	%8s	%-10d	%-5d	%-20s\n",
				i, t, v.top_message, v.unread_count, title,
			)
		}
	}
	return nil
}

func (m *MTProto) parsePeerById(str_id string) (peer TL, err error) {
	if str_id[0:1] == "#" {
		if s, err := strconv.Atoi(str_id[1:len(str_id)]); err == nil {
			peer = TL_inputPeerChannel{int32(s), m.accessHashes[int32(s)]}
		}
	} else if len(str_id) > 0 {
		id, err := strconv.Atoi(str_id)
		if str_id[0:1] == "@" {
			// TODO evaluate usernames starting with "@"
		} else {
			if id > 0 {
				if err == nil {
					peer = TL_inputPeerUser{int32(id), m.accessHashes[int32(id)]}
				}
			} else {
				if err == nil {
					peer = TL_inputPeerChat{int32(id) * -1}
				}
			}
		}
	} else {
		peer = TL_inputPeerSelf{}
	}
	return peer, err
}

func (m *MTProto) ResolveUsername(username string) (peer TL, err error) {
	resp := make(chan TL, 1)
	m.queueSend <- packetToSend{TL_contacts_resolveUsername{username}, resp}
	x := <-resp

	resolved, ok := x.(TL_contacts_resolvedPeer)

	if !ok {
		return nil, fmt.Errorf("RPC: %#v", x)
	}

	switch resolved.peer.(type) {
	case TL_peerChannel:
		return resolved.chats[0], nil
	case TL_peerUser:
		return resolved.users[0], nil
	}

	return resolved, nil
}

func (m *MTProto) GetFullChat(chat_id int32) error {
	resp := make(chan TL, 1)
	m.queueSend <- packetToSend{
		TL_messages_getFullChat{
			chat_id,
		},
		resp,
	}
	x := <-resp
	list, ok := x.(TL_messages_chatFull)

	if !ok {
		return fmt.Errorf("RPC: %#v", x)
	}

	fmt.Printf("%#v", list)

	return nil
}

func (m *MTProto) GetState() error {
	resp := make(chan TL, 1)
	m.queueSend <- packetToSend{TL_updates_getState{}, resp}
	x := <-resp
	_, ok := x.(TL_updates_state)
	if !ok {
		return fmt.Errorf("RPC: %#v", x)
	}

	//fmt.Printf("%#v",list)

	return nil
}

func (m *MTProto) SendMsg(peer_id string, msg string) error {
	peer, _ := m.parsePeerById(peer_id)
	f := uint32(0)
	f |= 1 << 1
	//	f |= 1 << 3
	switch peer.(type) {
	case TL_inputPeerChannel:
		//		f |= 1 << 4
		fmt.Printf("%v : ", peer.(TL_inputPeerChannel).channel_id)
	}
	fmt.Println(f)
	resp := make(chan TL, 1)
	m.queueSend <- packetToSend{
		TL_messages_sendMessage{
			f,
			true,
			false,
			peer,
			0,
			msg,
			rand.Int63(),
			TL_null{},
			[]TL{},
		},
		resp,
	}
	x := <-resp
	_, ok := x.(TL_updateShortSentMessage)
	fmt.Printf("%v, RPC: %#v", ok, x)
	if !ok {
		return fmt.Errorf("RPC: %#v", x)
	}

	return nil
}

func (m *MTProto) SendMedia(peer_id string, file string) (err error) {
	_512k := 512 * 1024
	peer, _ := m.parsePeerById(peer_id)
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("Error to read file: %#v", err)
	}
	md5_hash := fmt.Sprintf("%x", md5.Sum(bytes))
	fileId := rand.Int63()
	parts := int32(len(bytes)/_512k) + 1
	start := 0
	for i := int32(0); i < parts; i++ {
		fmt.Println(i, "/", parts)
		resp := make(chan TL, 1)
		end := start + _512k
		if end > len(bytes) {
			end = len(bytes)
		}
		m.queueSend <- packetToSend{
			TL_upload_saveFilePart{
				fileId,
				i,
				bytes[start:end],
			},
			resp,
		}
		x := <-resp
		_, ok := x.(TL_boolTrue)
		if !ok {
			return fmt.Errorf("upload_saveFilePart RPC: %#v", x)
		}
		start = end
	}

	f := uint32(0)

	resp := make(chan TL, 1)
	m.queueSend <- packetToSend{
		TL_messages_sendMedia{
			f,
			false,
			peer,
			0,
			TL_inputMediaUploadedPhoto{
				TL_inputFile{
					fileId,
					parts,
					file,
					md5_hash,
				},
				"Caption for image",
			},
			rand.Int63(),
			TL_null{},
		},
		resp,
	}
	x := <-resp
	_, ok := x.(TL_updateShortSentMessage)
	if !ok {
		return fmt.Errorf("messages_sendMedia RPC: %#v", x)
	}
	return nil
}

func (m *MTProto) EditTitle(peer_id string, title string) error {
	peer, _ := m.parsePeerById(peer_id)
	switch peer.(type) {
	case TL_inputPeerChannel:
		resp := make(chan TL, 1)
		m.queueSend <- packetToSend{
			TL_channels_editTitle{
				peer,
				title,
			},
			resp,
		}

		x := <-resp
		_, ok := x.(TL_updateShortSentMessage)
		fmt.Printf("%v, RPC: %#v", ok, x)
		if !ok {
			return fmt.Errorf("RPC: %#v", x)
		}
	case TL_inputPeerChat:
		resp := make(chan TL, 1)
		m.queueSend <- packetToSend{
			TL_messages_editChatTitle{
				peer.(TL_inputPeerChat).chat_id,
				title,
			},
			resp,
		}

		x := <-resp
		_, ok := x.(TL_updateShortSentMessage)
		fmt.Printf("%v, RPC: %#v", ok, x)
		if !ok {
			return fmt.Errorf("RPC: %#v", x)
		}
	default:
	}

	return nil
}

func (m *MTProto) processUpdates(t TL) {
	// TODO Process the updates that come, pretty straightforward
}

func (m *MTProto) startPing() {
	// goroutine (TL_ping)
	go func() {
		for {
			select {
			case <-m.stopPing:
				return
			case <-time.Tick(45 * time.Second):
				fmt.Println("Pinged")
				m.queueSend <- packetToSend{TL_ping{0xCADACADA}, nil}
			}
		}
	}()
}

func (m *MTProto) SendRoutine() {
	for x := range m.queueSend {
		err := m.SendPacket(x.msg, x.resp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "SendRoutine:", err)
			os.Exit(2)
		}
	}
}

func (m *MTProto) ReadRoutine(stop <-chan struct{}) {
	for true {
		data, err := m.Read(stop)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ReadRoutine:", err)
			os.Exit(2)
		}
		if data == nil {
			return
		}

		m.Process(m.msgId, m.seqNo, data)
	}

}

//func (m *MTProto) Process(msgId int64, seqNo int32, data interface{}) (interface{}, error) {
func (m *MTProto) Process(msgId int64, seqNo int32, data interface{}) interface{} {
	fmt.Fprintln(os.Stderr, "Received: ", reflect.TypeOf(data))
	switch data.(type) {
	case TL_msg_container:
		data := data.(TL_msg_container).messages
		for _, v := range data {
			m.Process(v.msg_id, v.seqno, v.body)
		}

	case TL_bad_server_salt:
		data := data.(TL_bad_server_salt)
		m.serverSalt = data.new_server_salt
		_ = m.saveData()
		m.mutex.Lock()
		for k, v := range m.msgsIdToAck {
			delete(m.msgsIdToAck, k)
			m.queueSend <- v
		}
		m.mutex.Unlock()

	case TL_new_session_created:
		data := data.(TL_new_session_created)
		m.serverSalt = data.server_salt
		_ = m.saveData()

	case TL_ping:
		data := data.(TL_ping)
		m.queueSend <- packetToSend{TL_pong{msgId, data.ping_id}, nil}

	case TL_pong:
		// (ignore)

	case TL_msgs_ack:
		data := data.(TL_msgs_ack)
		m.mutex.Lock()
		for _, v := range data.msg_ids {
			delete(m.msgsIdToAck, v)
		}
		m.mutex.Unlock()

	case TL_rpc_result:
		data := data.(TL_rpc_result)
		x := m.Process(msgId, seqNo, data.result)
		m.mutex.Lock()
		v, ok := m.msgsIdToResp[data.req_msg_id]
		if ok {
			v <- x.(TL)
			close(v)
			delete(m.msgsIdToResp, data.req_msg_id)
		}
		delete(m.msgsIdToAck, data.req_msg_id)
		m.mutex.Unlock()

	case TL_rpc_error:
		if data == nil {
			break
		}
		data := data.(TL_rpc_error)
		fmt.Println(data.Error())
		return data

	// Call update parsing function
	case TL_updatesTooLong:
		// Too many updates, it is necessary to execute updates.getDifference
	case TL_updateShortMessage:
		// Shortened constructor containing info on one new incoming message from a contact
	case TL_updateShortChatMessage:
		// Shortened constructor containing info on one new incoming text message from a chat
	case TL_updateShort:
		// Shortened constructor containing info on one update not requiring auxiliaty data
	case TL_updatesCombined:
		// Constructor for a group of updates
	case TL_updates:
		// Full constructor of updates
	default:
		return data

	}

	if (seqNo&1) == 1 && m.connected {
		m.queueSend <- packetToSend{TL_msgs_ack{[]int64{msgId}}, nil}
	}

	return nil
}

func (m *MTProto) saveData() (err error) {
	m.encrypted = true

	b := NewEncodeBuf(1024)
	b.StringBytes(m.authKey)
	b.StringBytes(m.authKeyHash)
	b.StringBytes(m.serverSalt)
	b.String(m.addr)

	err = m.f.Truncate(0)
	if err != nil {
		return err
	}

	_, err = m.f.WriteAt(b.buf, 0)
	if err != nil {
		return err
	}

	return nil
}

func (m *MTProto) readData() (err error) {
	b := make([]byte, 1024*4)
	n, err := m.f.ReadAt(b, 0)
	if n <= 0 {
		return fmt.Errorf("New session")
	}

	d := NewDecodeBuf(b)
	m.authKey = d.StringBytes()
	m.authKeyHash = d.StringBytes()
	m.serverSalt = d.StringBytes()
	m.addr = d.String()

	if d.err != nil {
		return d.err
	}

	return nil
}

func (m *MTProto) Halt() {
	select {}
}

func dump(x interface{}) {
	_, _ = pp.Println(x)
}
