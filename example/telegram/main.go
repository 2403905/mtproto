package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/carlosliracl/mtproto"
	"gopkg.in/readline.v1"
)

var commands = map[string]int{
	"hashes":               0,
	"auth":                 1,
	"list":                 0,
	"contact_add":          3,
	"dialogs":              0,
	"get_full_chat":        1,
	"get_full_user":        1,
	"channel_create":       2,
	"channel_invite_to":    3,
	"channel_participants": 2,
	"msg":             2,
	"edit_title":      2,
	"sendmedia":       2,
	"update_username": 1,
	"exit":            0,
}

func main() {
	var err error

	m, err := mtproto.NewMTProto(os.Getenv("HOME")+"/.telegram_go", false, false)
	if err != nil {
		fmt.Printf("Create failed: %s\n", err)
		os.Exit(2)
	}

	err = m.Connect()
	if err != nil {
		fmt.Printf("Connect failed: %s\n", err)
		os.Exit(2)
	}
	// TODO update hash list, maybe using a local db?
	m.GetDialogs()

	if len(os.Args) > 1 && os.Args[1] == "-daemon" {
		mtProtoRCP, err := mtproto.NewMTProtoRPC(m)
		if err != nil {
			fmt.Printf("Create RPC Server error: %s\n", err)
			os.Exit(2)
		}
		mtProtoRCP.StartRPCServer()

	} else {
		// TODO Do this better
		var completer = readline.NewPrefixCompleter(
			readline.PcItem("auth"),
			readline.PcItem("hashes"),
			readline.PcItem("list"),
			readline.PcItem("contact_add"),
			readline.PcItem("dialogs"),
			readline.PcItem("get_full_chat"),
			readline.PcItem("get_full_user"),
			readline.PcItem("update_username"),
			readline.PcItem("edit_title"),
			readline.PcItem("channel_create"),
			readline.PcItem("channel_invite_to"),
			readline.PcItem("channel_participants"),
			readline.PcItem("msg"),
			readline.PcItem("sendmedia"),
			readline.PcItem("resolve_username"),
			readline.PcItem("exit"),
		)

		rl, err := readline.NewEx(&readline.Config{
			Prompt:       "> ",
			AutoComplete: completer,
			HistoryFile:  "/tmp/readline.tmp",
		})

		if err != nil {
			panic(err)
		}
		defer rl.Close()

		shell := true
		for shell {
			input, err := rl.Readline()
			if err != nil {
				break
			}
			if input == "" {
				continue
			}
			commandline := regexp.MustCompile(`(?:(".*?"))|[^\s]+`)
			args := commandline.FindAllString(input, -1)

			for i := range args {
				args[i] = strings.Trim(args[i], `"`)
			}

			// TODO Do this better
			switch args[0] {
			case "help":
				for v, k := range commands {
					fmt.Printf("    %s [%d]\n", v, k)
				}
			case "auth":
				err = m.Auth(args[1])
			case "list":
				_, err = m.GetContacts()
			case "contact_add":
				_, err = m.ContactAdd(args[1:])
			case "dialogs":
				err = m.GetDialogs()
			case "edit_title":
				err = m.EditTitle(args[1], args[2])
			case "get_full_chat":
				chat_id, _ := strconv.Atoi(args[1])
				err = m.GetFullChat(int32(chat_id))
			case "resolve_username":
				if len(args) == 2 {
					fmt.Println(args[1])
					u, err := m.ResolveUsername(args[1])
					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println(u)
					}
				}
			case "msg":
				_, err = m.SendMsg(args[1:])
			case "sendmedia":
				err = m.SendMedia(args[1], args[2])
			case "channel_create":
				_, err = m.ChannelCreate(args[1:])
			case "channel_invite_to":
				_, err = m.ChannelInviteTo(args[1:])
			case "channel_participants":
				_, err = m.ChannelParticipants(args[1:])
			case "hashes":
				err = m.Hasshes()
			case "exit":
				shell = false
			default:
				fmt.Println(args[0], "not found.")
			}
			fmt.Println(err)
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	}
}
