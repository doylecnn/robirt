package main

import (
	"fmt"
	"math/rand"

	"strings"
	"time"
)

func delDiscussReplies(key, reply string, discussID, discussNum int64) {
	trans, err := db.Begin()
	if err != nil {
		reportError(err)
		return
	}
	sqlResult, err := trans.Exec("delete from discuss_replies where key = $1 and reply = $2 and discussion_id = $3", key, reply, discussID)
	if err != nil {
		reportError(err)
		trans.Rollback()
		return
	}
	trans.Commit()

	if affectRows, err := sqlResult.RowsAffected(); affectRows > 0 {
		sendDiscussMessage(discussNum, fmt.Sprintf("已删除：key=%s, value=%s", key, reply))
	} else {
		reportError(err)
	}
}

func listDiscussReplies(key string, discussID, discussNum int64) {
	rows, err := db.Query("select reply from discuss_replies where key = $1 and discussion_id = $2", key, discussID)
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
		sendDiscussMessage(discussNum, strings.Join(list, "\n"))
	}
}

func techDiscussReplies(key, reply string, userID, discussID, discussNum int64) {
	if length := len(makeTokens(reply)); length > 600 {
		logger.Println("too long!", length)
		sendDiscussMessage(discussNum, "太长记不住！")
	} else {
		if len([]rune(key)) < 2 && len(makeTokens(reply)) < 2 {
			sendDiscussMessage(discussNum, "触发字太短了！")
		} else {
			row := db.QueryRow("select count(1) from discuss_replies where key = $1 and reply = $2 and discussion_id = $3", key, reply, discussID)
			var count int
			row.Scan(&count)
			if count == 0 {
				trans, err := db.Begin()
				if err != nil {
					reportError(err)
					return
				}
				sqlResult, err := trans.Exec("insert into discuss_replies(key, reply, author_id, discussion_id) values($1, $2, $3, $4)", key, reply, userID, discussID)
				if err != nil {
					reportError(err)
					trans.Rollback()
					return
				}
				if affectRows, err := sqlResult.RowsAffected(); affectRows > 0 {
					sendDiscussMessage(discussNum, fmt.Sprintf("你说 “%s” 我说 “%s”", key, reply))
				} else {
					reportError(err)
					trans.Rollback()
					return
				}
				trans.Commit()
			} else {
				sendDiscussMessage(discussNum, fmt.Sprintf("我早就会这句了！"))
			}
		}
	}
}

func discussMessageHandle(p Params) {
	message, _ := p.getString("msg")
	qqNum, _ := p.getInt64("fromqq")
	discussNum, _ := p.getInt64("fromdiscuss")
	userID, _ := getUserID(qqNum)
	discussID, _ := getDiscussID(discussNum)

	logger.Printf("\n>>> discuss: %d-%d:\n>>> %s\n", discussNum, qqNum, message)

	if "!help" == message || "！help" == message || "/help" == message {
		sendDiscussMessage(discussNum, "!add 触发字=触发内容  添加一条\n!del 触发字=触发内容  删除一条\n!list 触发字  列出该触发字下的所有条目\n没有其他的了……")
	} else if delCmd.MatchString(message) {
		kv := delCmd.FindAllStringSubmatch(message, 1)[0]
		delDiscussReplies(kv[1], kv[2], discussID, discussNum)
	} else if listCmd.MatchString(message) {
		kv := listCmd.FindAllStringSubmatch(message, 1)[0]
		listDiscussReplies(kv[1], discussID, discussNum)
	} else if techCmd.MatchString(message) {
		kv := techCmd.FindAllStringSubmatch(message, 1)[0]
		key := strings.TrimSpace(kv[1])
		reply := kv[2]
		techDiscussReplies(key, reply, userID, discussID, discussNum)
		robirtLastActiveForDiscuss[discussNum] = time.Now().Add(-30 * time.Second)
	} else {
		if lastActive, ok := robirtLastActiveForDiscuss[discussNum]; ok && time.Since(lastActive).Seconds() < 28 {
			return
		}
		keys := spliteN(message)
		keys = append(keys, message)
		var list []string
		args := []interface{}{}
		args = append(args, discussID)
		var sqlParms []string
		for i, key := range keys {
			args = append(args, key)
			sqlParms = append(sqlParms, fmt.Sprintf("$%d", i+2))
		}
		selectStr := fmt.Sprintf("select reply from discuss_replies where key in (%s) and discussion_id = $1", strings.Join(sqlParms, ", "))
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
			sendDiscussMessage(discussNum, message)
			robirtLastActiveForDiscuss[discussNum] = time.Now()
		} else if strings.HasPrefix(message, " ") && strings.HasSuffix(message, "  ") {
			sendDiscussMessage(discussNum, message)
		}
	}
}
