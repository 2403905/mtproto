package mtproto

import (
	"encoding/json"
	"fmt"
	"strings"
)

type JSON interface {
}

type JSON_importedContact struct {
	UserId   int32 `json:"user_id"`
	ClientId int64 `json:"client_id"`
}

type JSON_contact struct {
	UserId int32  `json:"user_id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
}

type JSON_channel_created struct {
	ChannelId  int32  `json:"channel_id"`
	Title      string `json:"title"`
	AccessHash int64  `json:"access_hash"`
	Date       int32  `json:"date"`
}

type JSON_userListEmpty struct {
	UserIds []int32
}
type JSON_empty struct {
}

func json_encode(o JSON) string {
	b, _ := json.Marshal(o)
	s := string(b)
	//fmt.Printf("JSON: %s\n", s)
	return s
}

func (tl *TL_contacts_importedContacts) Json_encode() string {
	var importedContact JSON_importedContact
	if len(tl.imported) > 0 {
		importedContact = JSON_importedContact{
			UserId:   tl.imported[0].user_id,
			ClientId: tl.imported[0].client_id,
		}
	}
	return json_encode(importedContact)
}

func (tl *TL_contacts_contacts) Json_encode() string {
	contacts := make([]JSON_contact, len(tl.users))
	for i, u := range tl.users {
		user := u.(TL_user)
		item := JSON_contact{
			UserId: user.id,
			Name:   strings.TrimSpace(fmt.Sprintf("%s %s", user.first_name, user.last_name)),
			Phone:  user.phone,
		}
		contacts[i] = item
	}
	return json_encode(contacts)
}

func (tl *TL_channel) Json_encode() string {
	channel_created := JSON_channel_created{
		ChannelId:  tl.id,
		AccessHash: tl.access_hash,
		Date:       tl.date,
		Title:      tl.title,
	}
	return json_encode(channel_created)
}

func (json *JSON_userListEmpty) Json_encode() string {
	return json_encode(json)
}
func (json *JSON_empty) Json_encode() string {
	return json_encode(json)
}
