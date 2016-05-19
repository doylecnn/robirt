package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	"net"

	"os"

	"strings"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"
)

var (
	db              *sql.DB
	config          tomlConfig
	client_conn     *net.TCPConn
	close_sign_chan chan struct{}     = make(chan struct{})
	request_chan    chan Notification = make(chan Notification, 1)
)

func init() {

}

func main() {
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		log.Println(err)
		return
	}
	var err error
	db, err = sql.Open("postgres", config.Database.DBName)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	go eventLoop()
	go func() {
		l_addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:7008")
		if err != nil {
			log.Fatalf("ResolveTCPAddr Error: %v\n", err)
		}
		ln, err := net.ListenTCP("tcp4", l_addr)
		if err != nil {
			log.Fatal("Failed to listening server: %v", err)
		}
		log.Println("Listening server on tcp:127.0.0.1:7008")

		var groups_loaded = false
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("Accept Error: %v\n", err)
				continue
			}
			go handleRequest(conn)
			if !groups_loaded {
				GetToken()
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cmd(scanner.Text())
		}
	}()
	<-close_sign_chan
}

func cmd(cmd string) {
	if cmd == "exit" {
		close(close_sign_chan)
	}
	s := strings.SplitN(strings.TrimSpace(cmd), ":", 2)
	if len(s) == 2 {
		groupnum, err := strconv.ParseInt(strings.TrimSpace(s[0]), 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		var group Group
		var find = false
		for _, g := range groups {
			if g.GroupNo == groupnum {
				group = g
				find = true
				break
			}
		}
		if find {
			sendGroupMessage(group.GroupNo, s[1])
		} else {
			fmt.Println("group not found!")
		}
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	for {
		// Make a buffer to hold incoming data.
		buf := make([]byte, 4096)
		// Read the incoming connection into the buffer.
		reqLen, err := conn.Read(buf)
		if err != nil {
			log.Println("Error reading:", err.Error())
			return
		}
		r := bytes.NewReader(buf[:reqLen])
		b, _ := ioutil.ReadAll(r)
		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			b := bytes.TrimSpace(scanner.Bytes())
			if len(b) == 0 {
				continue
			}
			var js Notification
			err = json.Unmarshal(b, &js)
			if err != nil {
				log.Println(err)
				fmt.Println(string(b))
				continue
			}
			request_chan <- js
		}
	}
}

func eventLoop() {
	for {
		js := <-request_chan

		if js.Method == "Token" {
			go eventToken(js.Params)
			continue
		}

		subtype, _ := js.Params.GetInt64("subtype")
		sendTime, _ := js.Params.GetTime("sendtime")
		if time.Since(sendTime).Seconds() > 30 {
			continue
		}

		switch js.Method {
		case "GroupMessage":
			go event_group_message(js.Params)
		case "PrivateMessage":
			go event_private_message(js.Params)
		case "GroupMemberJoin":
			qqnum, _ := js.Params.GetInt64("qqnum")
			groupnum, _ := js.Params.GetInt64("groupnum")
			beingOperateQQ, _ := js.Params.GetInt64("opqqnum")
			message := "welcomeNewMember(subtype, groupnum, qqnum, beingOperateQQ)"
			sendGroupMessage(groupnum, message)
			GetToken()
		case "GroupMemberLeave":
			qqnum, _ := js.Params.GetInt64("qqnum")
			groupnum, _ := js.Params.GetInt64("groupnum")
			//beingOperateQQ := js.Params.GetInt64("opqqnum")
			if subtype == 1 {
				nickname := members[groupnum][qqnum].Nickname
				message := fmt.Sprintf("群员:[%s] 退群了!!!", nickname)
				sendGroupMessage(groupnum, message)
			} else if subtype == 2 {
				nickname := members[groupnum][qqnum].Nickname
				message := fmt.Sprintf("群员:[%s] 被 某个管理员 踢了!!!", nickname)
				sendGroupMessage(groupnum, message)
			}
			GetToken()
		case "RequestAddFriend":
			responseFlag, _ := js.Params.GetString("response_flag")
			addFriend(responseFlag, 1, "")
		case "RequestAddGroup":
			responseFlag, _ := js.Params.GetString("response_flag")
			if subtype == 2 {
				addGroup(responseFlag, 2, 1, "")
			}
			GetToken()
		default:
			log.Printf("未处理：%s\n", js)
		}
	}
}
