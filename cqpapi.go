package main

import (
	"fmt"
	"net"
)

func getToken() {
	json := "{\"method\":\"GetToken\",\"params\":{}}"
	sendNotification(json)
}

func addFriend(responseFlag string, accept int32, memo string) {
	responseFlag = jsonTrans(responseFlag)
	memo = jsonTrans(memo)
	json := fmt.Sprintf("{\"method\":\"FriendAdd\",\"params\":{\"responseFlag\":\"%s\",\"accept\":%d,\"memo\":\"%s\"}}", responseFlag, accept, memo)
	sendNotification(json)
}

func addGroup(responseFlag string, subtype, accept int32, reason string) {
	responseFlag = jsonTrans(responseFlag)
	reason = jsonTrans(reason)
	json := fmt.Sprintf("{\"method\":\"GroupAdd\",\"params\":{\"responseFlag\":\"%s\",\"subType\":%d,\"accept\":%d,\"reason\":\"%s\"}}", responseFlag, subtype, accept, reason)
	sendNotification(json)
}

func leaveGroup(groupNum, qqNum int64) {
	json := fmt.Sprintf("{\"method\":\"GroupLeave\",\"params\":{\"groupnum\":%d,\"qqnum\":%d}}", groupNum, qqNum)
	sendNotification(json)
}

func groupBan(groupNum, qqNum, seconds int64) {
	json := fmt.Sprintf("{\"method\":\"GroupBan\",\"params\":{\"groupnum\":%d,\"qqnum\":%d,\"seconds\":%d}}", groupNum, qqNum, seconds)
	sendNotification(json)
}

func sendGroupMessage(groupNum int64, message string) {
	if len(message) > 0 {
		message = jsonTrans(message)
		json := fmt.Sprintf("{\"method\":\"SendGroupMessage\",\"params\":{\"groupnum\":%d,\"message\":\"%s\"}}", groupNum, message)
		sendNotification(json)
	}
}

func sendPrivateMessage(qqNum int64, message string) {
	if len(message) > 0 {
		message = jsonTrans(message)
		json := fmt.Sprintf("{\"method\":\"SendPrivateMessage\",\"params\":{\"qqnum\":%d,\"message\":\"%s\"}}", qqNum, message)
		sendNotification(json)
	}
}

func sendDiscussMessage(discussNum int64, message string) {
	if len(message) > 0 {
		message = jsonTrans(message)
		json := fmt.Sprintf("{\"method\":\"SendDiscussionMessage\",\"params\":{\"discussionnum\":%d,\"message\":\"%s\"}}", discussNum, message)
		sendNotification(json)
	}
}

func sendNotification(notification string) {
	logger.Printf("\n<<< %s\n", notification)
	var b = []byte(notification)

	if clientConn == nil {
		remoteAddress, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:7000")
		if err != nil {
			logger.Printf("ResolveTCPAddr Error: %v\n", err)
			return
		}
		clientConn, err = net.DialTCP("tcp4", nil, remoteAddress)
		if err != nil {
			logger.Printf("Net DialTCP Error: %v\n", err)
			return
		}
	}
	count, err := clientConn.Write(b)
	if err != nil {
		logger.Println(err)
	}
	if count != len(b) {
		logger.Println("not all send...")
	}

	clientConn.Close()
	clientConn = nil
}
