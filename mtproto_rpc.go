package mtproto

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

type MTProtoRPC struct {
	mtproto *MTProto
	port    int
}

// Response types to GetContacts
type ArgsGetContacts struct {
}

// Response types to ContactAdd
type ArgsContactAdd struct {
	Phone     string
	FirstName string
	LastName  string
}
type ArgsChannelCreate struct {
	Name  string
	About string
}

type ArgsChannelInviteTo struct {
	ChannelId string
	UserId    string
}
type ArgsSendMessage struct {
	PeerId  string
	Message string
}
type Result string

func NewMTProtoRPC(m *MTProto) (*MTProtoRPC, error) {
	m.runningRPCServer = true
	mtProtoRPC := new(MTProtoRPC)
	mtProtoRPC.mtproto = m
	mtProtoRPC.port = 2331
	return mtProtoRPC, nil
}

func (t *MTProtoRPC) ContactList(r *http.Request, args *ArgsGetContacts, result *Result) error {
	tl, err := t.mtproto.GetContacts()
	if err != nil {
		return err
	}
	tl_contacts_contacts := tl.(TL_contacts_contacts)
	s := tl_contacts_contacts.Json_encode()
	*result = Result(s)
	return nil
}

func (t *MTProtoRPC) ChannelInviteTo(r *http.Request, args *ArgsChannelInviteTo, result *Result) error {
	// TODO Make this better
	// stringArgs := make([]string,len(args.Ids) + 1)
	stringArgs := make([]string, 2)
	stringArgs[0] = args.ChannelId
	stringArgs[1] = args.UserId

	// stringArgs = append(stringArgs[1:],  args.Ids...)

	invitedIds, err := t.mtproto.ChannelInviteTo(stringArgs)
	if err != nil {
		return err
	}
	// TODO Do this better
	response := JSON_userListEmpty{
		invitedIds,
	}
	s := response.Json_encode()
	*result = Result(s)
	return nil
}

func (t *MTProtoRPC) SendMessage(r *http.Request, args *ArgsSendMessage, result *Result) error {
	// TODO Make this better
	// stringArgs := make([]string,len(args.Ids) + 1)
	stringArgs := make([]string, 2)
	stringArgs[0] = args.PeerId
	stringArgs[1] = args.Message

	// stringArgs = append(stringArgs[1:],  args.Ids...)

	_, err := t.mtproto.SendMsg(stringArgs)
	if err != nil {
		return err
	}
	// TODO Do this better
	response := JSON_empty{}
	s := response.Json_encode()
	*result = Result(s)
	return nil
}

func (t *MTProtoRPC) ContactAdd(r *http.Request, args *ArgsContactAdd, result *Result) error {
	stringArgs := make([]string, 3)
	stringArgs[0] = args.Phone
	stringArgs[1] = args.FirstName
	stringArgs[2] = args.LastName

	tl, err := t.mtproto.ContactAdd(stringArgs)
	if err != nil {
		return err
	}
	tl_importedContacts := tl.(TL_contacts_importedContacts)
	s := tl_importedContacts.Json_encode()
	*result = Result(s)
	return nil
}

func (t *MTProtoRPC) ChannelCreate(r *http.Request, args *ArgsChannelCreate, result *Result) error {
	stringArgs := make([]string, 2)
	stringArgs[0] = args.Name
	stringArgs[1] = args.About

	tl, err := t.mtproto.ChannelCreate(stringArgs)
	if err != nil {
		return err
	}
	tl_channel := tl.(TL_channel)
	s := tl_channel.Json_encode()
	*result = Result(s)
	return nil
}

func (mtProtoRPC *MTProtoRPC) StartRPCServer() {
	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	s.RegisterService(mtProtoRPC, "")
	r := mux.NewRouter()
	r.Handle("/rpc", s)
	fmt.Printf("Starting RPC Server on port:%d\n", mtProtoRPC.port)
	http.ListenAndServe(fmt.Sprintf(":%d", mtProtoRPC.port), r)
}
