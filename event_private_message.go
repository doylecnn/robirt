package main

import (
	"fmt"
	"log"

	"math/rand"
	"strconv"
	"strings"
	"time"
)

func event_private_message(p Params) {
	subtype, _ := p.GetInt64("subtype")
	if subtype != 11 {
		return
	}
	message, _ := p.GetString("message")
	qqnum, _ := p.GetInt64("qqnum")
	user_id, _ := get_user_id(qqnum)
	fmt.Printf("[私聊]%d:\n%s\n", qqnum, message)
	if message == "help" {
		sendPrivateMessage(qqnum, "私聊调教：\n !add 群号码 触发词=内容\n !del 群号码 触发词=内容")
		return
	}
	if tech_cmd_by_private_message_to_all_groups.MatchString(message) {
		if qqnum != config.SuperUser.QQNumber {
			return
		}
		kv := tech_cmd_by_private_message_to_all_groups.FindAllStringSubmatch(message, 1)[0]
		if length := message_length(kv[2]); length > 600 {
			log.Println("too long!", length)
			sendGroupMessage(qqnum, "太长记不住！")
		} else {
			key := strings.TrimSpace(kv[1])
			if len(key) < 2 {
				sendGroupMessage(qqnum, "触发字太短了！")
			} else {
				value := kv[2]
				trans, err := db.Begin()
				if err != nil {
					reportError(err)
					return
				}
				groups.RWLocker.RLock()
				for _, group := range groups.Map {
					row := trans.QueryRow("select count(1) from replies where key = $1 and reply = $2 and group_id = $3", key, value, group.Id)
					var count int
					row.Scan(&count)
					if count == 0 {
						_, err := trans.Exec("insert into replies(author_id ,key, reply, group_id) values($1, $2, $3, $4)", user_id, key, value, group.Id)
						if err != nil {
							reportError(err)
							//trans.Rollback()
						}
					}
				}
				groups.RWLocker.RUnlock()
				trans.Commit()
				sendPrivateMessage(qqnum, fmt.Sprintf("在 所有 群，有人说 “%s” 我就说 “%s”", key, value))
			}
		}
	} else if tech_cmd_by_private_message_to_one_groups.MatchString(message) {
		tmp := tech_cmd_by_private_message_to_one_groups.FindAllStringSubmatch(message, 1)[0]
		groupnum, err := strconv.ParseInt(tmp[1], 10, 64)
		var group_id int64
		if err != nil {
			sendPrivateMessage(qqnum, err.Error())
			return
		}
		groups.RWLocker.RLock()
		if g, group_exists := groups.Map[groupnum]; group_exists {
			group_id = g.Id
			groups.RWLocker.RUnlock()
		} else {
			groups.RWLocker.RUnlock()
			return
		}
		
		kv := tmp[2:]
		if length := message_length(kv[1]); length > 600 {
			log.Println("too long!", length)
			sendGroupMessage(qqnum, "太长记不住！")
		} else {
			key := strings.TrimSpace(kv[0])
			if len(key) < 2 {
				sendGroupMessage(qqnum, "触发字太短了！")
			} else {
				value := kv[1]
				trans, err := db.Begin()
				if err != nil {
					reportError(err)
					return
				}
				row := trans.QueryRow("select count(1) from replies where key = $1 and reply = $2 and group_id = $3", key, value, group_id)
				var count int
				row.Scan(&count)
				if count == 0 {
					_, err := trans.Exec("insert into replies(author_id ,key, reply, group_id) values($1, $2, $3, $4)", user_id, key, value, group_id)
					if err != nil {
						reportError(err)
						trans.Rollback()
						return
					}
					sendPrivateMessage(qqnum, fmt.Sprintf("在 %d 这个群，有人说 “%s” 我就说 “%s”", groupnum, key, value))
				}
				trans.Commit()
			}
		}
	} else if del_cmd_by_private_message_for_all_groups.MatchString(message) {
		if qqnum != config.SuperUser.QQNumber {
			return
		}
		kv := del_cmd_by_private_message_for_all_groups.FindAllStringSubmatch(message, 1)[0]
		trans, err := db.Begin()
		if err != nil {
			reportError(err)
			return
		}
		sql_result, err := trans.Exec("delete from replies where key = $1 and reply = $2", kv[1], kv[2])
		if err != nil {
			reportError(err)
			trans.Rollback()
			return
		}
		trans.Commit()
		if affect_rows, _ := sql_result.RowsAffected(); affect_rows > 0 {
			sendPrivateMessage(qqnum, fmt.Sprintf("在 所有 群，已删除：key=%s, value=%s", kv[1], kv[2]))
		}
	} else if del_cmd_by_private_message_for_one_groups.MatchString(message) {
		tmp := del_cmd_by_private_message_for_one_groups.FindAllStringSubmatch(message, 1)[0]
		groupnum, err := strconv.ParseInt(tmp[1], 10, 64)
		var group_id int64
		if err != nil {
			sendPrivateMessage(qqnum, err.Error())
			return
		}
		groups.RWLocker.RLock()
		if g, group_exists := groups.Map[groupnum]; group_exists {
			group_id = g.Id
			groups.RWLocker.RUnlock()
		} else {
			groups.RWLocker.RUnlock()
			return
		}

		kv := tmp[2:]
		trans, err := db.Begin()
		if err != nil {
			reportError(err)
			return
		}
		sql_result, err := trans.Exec("delete from replies where key = $1 and reply = $2 and group_id = $3", kv[0], kv[1], group_id)
		if err != nil {
			reportError(err)
			trans.Rollback()
			return
		}
		trans.Commit()
		if affect_rows, _ := sql_result.RowsAffected(); affect_rows > 0 {
			sendPrivateMessage(qqnum, fmt.Sprintf("在 %d 这个群，已删除：key=%s, value=%s", group_id, kv[0], kv[1]))
		}
	} else {
		if qqnum != config.SuperUser.QQNumber {
			return
		}
		switch message {
		case "update groups info":
			GetToken()
		default:
			if strings.HasPrefix(message, "sendto:") {
				s := strings.SplitN(strings.TrimSpace(message[7:]), " ", 2)
				if len(s) == 2 {
					groupnum, err := strconv.ParseInt(strings.TrimSpace(s[0]), 10, 64)
					if err != nil {
						fmt.Println(err)
						sendPrivateMessage(qqnum, err.Error())
						return
					}
					groups.RWLocker.RLock()
					group, group_exists := groups.Map[groupnum]
					groups.RWLocker.RUnlock()
					if group_exists {
						msg := s[1]
						if at_regex.MatchString(msg) {
							msg = at_regex.ReplaceAllString(msg, " [CQ:at,qq=$1]")
						}
						sendGroupMessage(group.GroupNo, msg)
					} else {
						fmt.Println("group not found!")
						sendPrivateMessage(qqnum, "group not found!")
					}
				}
			} else if strings.HasPrefix(message, "leave group:") {
				groupnum, err := strconv.ParseInt(strings.TrimSpace(message[12:]), 10, 64)
				if err != nil {
					fmt.Println(err)
					sendPrivateMessage(qqnum, err.Error())
					return
				}
				groups.RWLocker.RLock()
				group, group_exists := groups.Map[groupnum]
				groups.RWLocker.RUnlock()
				if group_exists {
					sendGroupMessage(group.GroupNo, "我决定离开，再见~")
					leaveGroup(group.GroupNo, LoginQQ)
				} else {
					fmt.Println("group not found!")
					sendPrivateMessage(qqnum, "group not found!")
				}
			} else if strings.HasPrefix(message, "sendtoall:") {
				go func() {
					rand.Seed(time.Now().Unix())

					groups.RWLocker.RLock()
					for _, group := range groups.Map {
						sendGroupMessage(group.GroupNo, strings.TrimSpace(message[10:]))
						d := time.Duration(rand.Intn(3)+3) * time.Second
						time.Sleep(d)
					}
					groups.RWLocker.RUnlock()
				}()
			} else {
				sendPrivateMessage(qqnum, message)
			}
		}
	}
}
