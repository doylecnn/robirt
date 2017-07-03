package main

import (
	"fmt"

	"math/rand"
	"strconv"
	"strings"
	"time"
)

func privateMessageHandle(p Params) {
	subtype, _ := p.getInt64("subtype")
	if subtype != 11 {
		return
	}
	message, _ := p.getString("message")
	qqNum, _ := p.getInt64("qqnum")
	userID, _ := getUserID(qqNum)
	logger.Printf("\n>>> [私聊]%d:\n>>> %s\n", qqNum, message)
	if message == "help" {
		sendPrivateMessage(qqNum, "私聊调教：\n !add 群号码 触发词=内容\n !del 群号码 触发词=内容")
		return
	}
	if techCmdByPrivateMessageToAllGroups.MatchString(message) {
		if qqNum != config.SuperUser.QQNumber {
			return
		}
		kv := techCmdByPrivateMessageToAllGroups.FindAllStringSubmatch(message, 1)[0]
		if length := len(makeTokens(kv[2])); length > 600 {
			logger.Println("too long!", length)
			sendGroupMessage(qqNum, "太长记不住！")
		} else {
			key := strings.TrimSpace(kv[1])
			if len(key) < 2 {
				sendGroupMessage(qqNum, "触发字太短了！")
			} else {
				value := kv[2]
				trans, err := db.Begin()
				if err != nil {
					reportError(err)
					return
				}
				groups.Range(func(k, v interface{}) bool {
					group := v.(Group)
					row := trans.QueryRow("select count(1) from replies where key = $1 and reply = $2 and group_id = $3", key, value, group.ID)
					var count int
					row.Scan(&count)
					if count == 0 {
						_, err := trans.Exec("insert into replies(author_id ,key, reply, group_id) values($1, $2, $3, $4)", userID, key, value, group.ID)
						if err != nil {
							reportError(err)
						}
					}
					return true
				})

				trans.Commit()
				sendPrivateMessage(qqNum, fmt.Sprintf("在 所有 群，有人说 “%s” 我就说 “%s”", key, value))
			}
		}
	} else if techCmdByPrivateMessageToGroup.MatchString(message) {
		tmp := techCmdByPrivateMessageToGroup.FindAllStringSubmatch(message, 1)[0]
		groupNum, err := strconv.ParseInt(tmp[1], 10, 64)
		var groupID int64
		if err != nil {
			sendPrivateMessage(qqNum, err.Error())
			return
		}
		if g, ok := groups.Load(groupNum); ok {
			groupID = g.(Group).ID
		} else {
			return
		}

		kv := tmp[2:]
		if length := len(makeTokens(kv[1])); length > 600 {
			logger.Println("too long!", length)
			sendGroupMessage(qqNum, "太长记不住！")
		} else {
			key := strings.TrimSpace(kv[0])
			if len(key) < 2 {
				sendGroupMessage(qqNum, "触发字太短了！")
			} else {
				value := kv[1]
				trans, err := db.Begin()
				if err != nil {
					reportError(err)
					return
				}
				row := trans.QueryRow("select count(1) from replies where key = $1 and reply = $2 and group_id = $3", key, value, groupID)
				var count int
				row.Scan(&count)
				if count == 0 {
					_, err := trans.Exec("insert into replies(author_id ,key, reply, group_id) values($1, $2, $3, $4)", userID, key, value, groupID)
					if err != nil {
						reportError(err)
						trans.Rollback()
						return
					}
					sendPrivateMessage(qqNum, fmt.Sprintf("在 %d 这个群，有人说 “%s” 我就说 “%s”", groupNum, key, value))
				}
				trans.Commit()
			}
		}
	} else if delCmdByPrivateMessageForAllGroups.MatchString(message) {
		if qqNum != config.SuperUser.QQNumber {
			return
		}
		kv := delCmdByPrivateMessageForAllGroups.FindAllStringSubmatch(message, 1)[0]
		trans, err := db.Begin()
		if err != nil {
			reportError(err)
			return
		}
		sqlResult, err := trans.Exec("delete from replies where key = $1 and reply = $2", kv[1], kv[2])
		if err != nil {
			reportError(err)
			trans.Rollback()
			return
		}
		trans.Commit()
		if affectRows, _ := sqlResult.RowsAffected(); affectRows > 0 {
			sendPrivateMessage(qqNum, fmt.Sprintf("在 所有 群，已删除：key=%s, value=%s", kv[1], kv[2]))
		}
	} else if delCmdByPrivateMessageForGroup.MatchString(message) {
		tmp := delCmdByPrivateMessageForGroup.FindAllStringSubmatch(message, 1)[0]
		groupNum, err := strconv.ParseInt(tmp[1], 10, 64)
		var groupID int64
		if err != nil {
			sendPrivateMessage(qqNum, err.Error())
			return
		}
		if g, ok := groups.Load(groupNum); ok {
			groupID = g.(Group).ID
		} else {
			return
		}

		kv := tmp[2:]
		trans, err := db.Begin()
		if err != nil {
			reportError(err)
			return
		}
		sqlResult, err := trans.Exec("delete from replies where key = $1 and reply = $2 and group_id = $3", kv[0], kv[1], groupID)
		if err != nil {
			reportError(err)
			trans.Rollback()
			return
		}
		trans.Commit()
		if affectRows, _ := sqlResult.RowsAffected(); affectRows > 0 {
			sendPrivateMessage(qqNum, fmt.Sprintf("在 %d 这个群，已删除：key=%s, value=%s", groupID, kv[0], kv[1]))
		}
	} else {
		if qqNum != config.SuperUser.QQNumber {
			return
		}
		switch message {
		case "update groups info":
			getToken()
		default:
			if strings.HasPrefix(message, "sendto:") {
				s := strings.SplitN(strings.TrimSpace(message[7:]), " ", 2)
				if len(s) == 2 {
					groupNum, err := strconv.ParseInt(strings.TrimSpace(s[0]), 10, 64)
					if err != nil {
						fmt.Println(err)
						sendPrivateMessage(qqNum, err.Error())
						return
					}

					if group, ok := groups.Load(groupNum); ok {
						msg := s[1]
						if atRegex.MatchString(msg) {
							msg = atRegex.ReplaceAllString(msg, " [CQ:at,qq=$1]")
						}
						sendGroupMessage(group.(Group).GroupNum, msg)
					} else {
						fmt.Println("group not found!")
						sendPrivateMessage(qqNum, "group not found!")
					}
				}
			} else if strings.HasPrefix(message, "leave group:") {
				groupNum, err := strconv.ParseInt(strings.TrimSpace(message[12:]), 10, 64)
				if err != nil {
					fmt.Println(err)
					sendPrivateMessage(qqNum, err.Error())
					return
				}

				if v, ok := groups.Load(groupNum); ok {
					group := v.(Group)
					sendGroupMessage(group.GroupNum, "我决定离开，再见~")
					leaveGroup(group.GroupNum, LoginQQ)
				} else {
					fmt.Println("group not found!")
					sendPrivateMessage(qqNum, "group not found!")
				}
			} else if strings.HasPrefix(message, "sendtoall:") {
				go func() {
					rand.Seed(time.Now().Unix())

					groups.Range(func(k, v interface{}) bool {
						group := v.(Group)
						sendGroupMessage(group.GroupNum, strings.TrimSpace(message[10:]))
						d := time.Duration(rand.Intn(3)+3) * time.Second
						time.Sleep(d)
						return true
					})
				}()
			} else {
				sendPrivateMessage(qqNum, message)
			}
		}
	}
}
