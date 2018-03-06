package main

import (
	"fmt"
	"math/rand"

	"strings"
	"time"
)

func delReplies(key, reply string, groupID, groupNum int64) {
	trans, err := db.Begin()
	if err != nil {
		reportError(err)
		return
	}
	sqlResult, err := trans.Exec("delete from replies where key = $1 and reply = $2 and group_id = $3", key, reply, groupID)
	if err != nil {
		reportError(err)
		trans.Rollback()
	} else {
		trans.Commit()
	}

	if affectRows, err := sqlResult.RowsAffected(); affectRows > 0 {
		sendGroupMessage(groupNum, fmt.Sprintf("已删除：key=%s, value=%s", key, reply))
	} else {
		reportError(err)
	}
}

func listReplies(key string, groupID, groupNum int64) {
	rows, err := db.Query("select reply from replies where key = $1 and group_id = $2", key, groupID)
	if err != nil {
		reportError(err)
		return
	}
	var list []string
	var i = 0
	for rows.Next() {
		var resp string
		err = rows.Scan(&resp)
		if err != nil {
			reportError(err)
			continue
		}

		if i < 5 {
			list = append(list, resp)
		}
		i++
	}
	count := len(list)
	if count > 0 {
		if count < i {
			list = append(list, fmt.Sprintf("之后 %d 项省略....", i-count))
		}
		sendGroupMessage(groupNum, strings.Join(list, "\n"))
	}
}

func techReplies(key, reply string, userID, groupID, groupNum int64) {
	if length := len(makeTokens(reply)); length > 600 {
		logger.Println("too long!", length)
		sendGroupMessage(groupNum, "太长记不住！")
	} else {
		if len([]rune(key)) < 2 && len(makeTokens(reply)) < 2 {
			sendGroupMessage(groupNum, "触发字太短了！")
		} else {
			row := db.QueryRow("select count(1) from replies where key = $1 and reply = $2 and group_id = $3", key, reply, groupID)
			var count int
			row.Scan(&count)
			if count == 0 {
				trans, err := db.Begin()
				if err != nil {
					reportError(err)
					return
				}
				sqlResult, err := trans.Exec("insert into replies(key, reply, author_id, group_id) values($1, $2, $3, $4)", key, reply, userID, groupID)
				if err != nil {
					reportError(err)
					trans.Rollback()
					return
				}
				if affectRows, err := sqlResult.RowsAffected(); affectRows > 0 {
					sendGroupMessage(groupNum, fmt.Sprintf("你说 “%s” 我说 “%s”", key, reply))
				} else {
					reportError(err)
					trans.Rollback()
					return
				}
				trans.Commit()
			} else {
				sendGroupMessage(groupNum, fmt.Sprintf("我早就会这句了！"))
			}
		}
	}
}

func groupMessageHandle(p Params) {
	message, _ := p.getString("message")
	anonymousname, _ := p.getString("anonymousname")
	qqNum, _ := p.getInt64("qqnum")
	groupNum, _ := p.getInt64("groupnum")
	userID, _ := getUserID(qqNum)
	groupID, _ := getGroupID(groupNum)
	var nickname string
	if len(anonymousname) > 0 {
		nickname = "[匿名]" + anonymousname
	} else {
		// if v, ok := groups.Load(groupNum); ok {
		// 	group := v.(Group)
		// 	members := group.Members
		// 	if member, ok := members.getMember(qqNum); ok {
		// 		nickname = member.Nickname
		// 	}
		// }
	}
	var group Group
	if g, ok := groups.Load(groupNum); ok {
		group = g.(Group)
	} else {
		return
	}
	logger.Printf("\n>>> %s(%d)-%s(%d):\n>>> %s\n", group.GroupName, groupNum, nickname, qqNum, message)

	if "!help" == message || "！help" == message || "/help" == message {
		sendGroupMessage(groupNum, "!add 触发字=触发内容  添加一条\n!del 触发字=触发内容  删除一条\n!list 触发字  列出该触发字下的所有条目\n没有其他的了……")
	} else if delCmd.MatchString(message) {
		kv := delCmd.FindAllStringSubmatch(message, 1)[0]
		delReplies(kv[1], kv[2], groupID, groupNum)
	} else if listCmd.MatchString(message) {
		kv := listCmd.FindAllStringSubmatch(message, 1)[0]
		listReplies(kv[1], groupID, groupNum)
	} else if techCmd.MatchString(message) {
		kv := techCmd.FindAllStringSubmatch(message, 1)[0]
		key := strings.TrimSpace(kv[1])
		reply := kv[2]
		techReplies(key, reply, userID, groupID, groupNum)
		robirtLastActive[groupNum] = time.Now().Add(-30 * time.Second)
	} else if hongbao.MatchString(message) {
		kv := hongbao.FindAllStringSubmatch(message, 1)[0]
		if kv[1] == "我的天哪！土豪发红包啦！大家快抢啊！" {
			sendGroupMessage(groupNum, "我的天哪！！土豪发红包啦！！大家快抢啊！！")
		} else {
			sendGroupMessage(groupNum, "我的天哪！土豪发红包啦！大家快抢啊！")
		}
		d, _ := time.ParseDuration("300s")
		robirtLastActive[groupNum] = time.Now().Add(d)
	} else {
		if lastActive, ok := robirtLastActive[groupNum]; ok && time.Since(lastActive).Seconds() < config.Robirt.Cdtime {
			return
		}
		keys := spliteN(message)
		keys = append(keys, message)
		var list []string
		args := []interface{}{}
		args = append(args, groupID)
		var sqlParms []string
		for i, key := range keys {
			args = append(args, key)
			sqlParms = append(sqlParms, fmt.Sprintf("$%d", i+2))
		}
		selectStr := fmt.Sprintf("select reply from replies where key in (%s) and group_id = $1", strings.Join(sqlParms, ", "))
		rows, err := db.Query(selectStr, args...)
		if err == nil {
			for rows.Next() {
				var resp string
				if err := rows.Scan(&resp); err != nil {
					logger.Fatal(err)
				}
				if len(makeTokens(resp)) < 600 {
					list = append(list, resp)
				}
			}
			rows.Close()
		}
		if length := len(list); length > 0 {
			message := list[rand.Intn(length)]
			sendGroupMessage(groupNum, message)
			robirtLastActive[groupNum] = time.Now()
		} else if strings.HasPrefix(message, " ") && strings.HasSuffix(message, "  ") {
			sendGroupMessage(groupNum, message)
		}
	}
}
