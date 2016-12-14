package main

import (
	"fmt"
	"net"
)

func GetToken() {
	json := "{\"method\":\"GetToken\",\"params\":{}}"
	sendNotification(json)
}

func addFriend(responseFlag string, accept int32, memo string) {
	responseFlag = json_trans(responseFlag)
	memo = json_trans(memo)
	json := fmt.Sprintf("{\"method\":\"FriendAdd\",\"params\":{\"responseFlag\":\"%s\",\"accept\":%d,\"memo\":\"%s\"}}", responseFlag, accept, memo)
	sendNotification(json)
}

func addGroup(responseFlag string, subtype, accept int32, reason string) {
	responseFlag = json_trans(responseFlag)
	reason = json_trans(reason)
	json := fmt.Sprintf("{\"method\":\"GroupAdd\",\"params\":{\"responseFlag\":\"%s\",\"subType\":%d,\"accept\":%d,\"reason\":\"%s\"}}", responseFlag, subtype, accept, reason)
	sendNotification(json)
}

func leaveGroup(groupnum, qqnum int64) {
	json := fmt.Sprintf("{\"method\":\"GroupLeave\",\"params\":{\"groupnum\":%d,\"qqnum\":%d}}", groupnum, qqnum)
	sendNotification(json)
}

func GroupBan(groupnum, qqnum, seconds int64) {
	json := fmt.Sprintf("{\"method\":\"GroupBan\",\"params\":{\"groupnum\":%d,\"qqnum\":%d,\"seconds\":%d}}", groupnum, qqnum, seconds)
	sendNotification(json)
}

func sendGroupMessage(groupnum int64, message string) {
	if len(message) > 0 {
		message = json_trans(message)
		json := fmt.Sprintf("{\"method\":\"SendGroupMessage\",\"params\":{\"groupnum\":%d,\"message\":\"%s\"}}", groupnum, message)
		sendNotification(json)
	}
}

func sendPrivateMessage(qqnum int64, message string) {
	if len(message) > 0 {
		message = json_trans(message)
		json := fmt.Sprintf("{\"method\":\"SendPrivateMessage\",\"params\":{\"qqnum\":%d,\"message\":\"%s\"}}", qqnum, message)
		sendNotification(json)
	}
}

func sendNotification(notification string) {
	fmt.Println(notification)
	var b = []byte(notification)

	if client_conn == nil {
		r_addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:7000")
		if err != nil {
			logger.Printf("ResolveTCPAddr Error: %v\n", err)
			return
		}
		client_conn, err = net.DialTCP("tcp4", nil, r_addr)
		if err != nil {
			logger.Printf("Net DialTCP Error: %v\n", err)
			return
		}
	}
	count, err := client_conn.Write(b)
	if err != nil {
		fmt.Println(err)
	}
	if count != len(b) {
		fmt.Println("not all send...")
	}

	client_conn.Close()
	client_conn = nil
}
