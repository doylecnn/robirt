package main

import (
	"fmt"
	"math/rand"

	"strings"
	"time"
)

func del_disc_replies(key, reply string, discuss_id, discussnum int64) {
	trans, err := db.Begin()
	if err != nil {
		reportError(err)
		return
	}
	sql_result, err := trans.Exec("delete from discuss_replies where key = $1 and reply = $2 and discuss_id = $3", key, reply, discuss_id)
	if err != nil {
		reportError(err)
		trans.Rollback()
		return
	}
	trans.Commit()

	if affect_rows, err := sql_result.RowsAffected(); affect_rows > 0 {
		sendGroupMessage(discussnum, fmt.Sprintf("已删除：key=%s, value=%s", key, reply))
	} else {
		reportError(err)
	}
}

func list_disc_replies(key string, discuss_id, discussnum int64) {
	rows, err := db.Query("select reply from discuss_replies where key = $1 and discuss_id = $2", key, discuss_id)
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
		sendGroupMessage(discussnum, strings.Join(list, "\n"))
	}
}

func tech_disc_replies(key, reply string, userid, groupid, discussnum int64) {
	if length := len(make_tokens(reply)); length > 600 {
		logger.Println("too long!", length)
		sendGroupMessage(discussnum, "太长记不住！")
	} else {
		if len([]rune(key)) < 2 && len(make_tokens(reply)) < 2 {
			sendGroupMessage(discussnum, "触发字太短了！")
		} else {
			row := db.QueryRow("select count(1) from discuss_replies where key = $1 and reply = $2 and discuss_id = $3", key, reply, groupid)
			var count int
			row.Scan(&count)
			if count == 0 {
				trans, err := db.Begin()
				if err != nil {
					reportError(err)
					return
				}
				sql_result, err := trans.Exec("insert into discuss_replies(key, reply, author_id, discuss_id) values($1, $2, $3, $4)", key, reply, userid, groupid)
				if err != nil {
					reportError(err)
					trans.Rollback()
					return
				}
				if affect_rows, err := sql_result.RowsAffected(); affect_rows > 0 {
					sendGroupMessage(discussnum, fmt.Sprintf("你说 “%s” 我说 “%s”", key, reply))
				} else {
					reportError(err)
					trans.Rollback()
					return
				}
				trans.Commit()
			}else{
				sendGroupMessage(discussnum, fmt.Sprintf("我早就会这句了！"))
			}
		}
	}
}

func event_discuss_message(p Params) {
	message, _ := p.GetString("msg")
	qqnum, _ := p.GetInt64("fromqq")
	discussnum, _ := p.GetInt64("fromdiscuss")
	user_id, _ := get_user_id(qqnum)
	discuss_id, _ := get_discuss_id(discussnum)

	fmt.Printf("discuss: %d-%d:\n%s\n", discussnum, qqnum, message)

	if "!help" == message || "！help" == message || "/help" == message {
		sendDiscussMessage(discussnum, "!add 触发字=触发内容  添加一条\n!del 触发字=触发内容  删除一条\n!list 触发字  列出该触发字下的所有条目\n没有其他的了……")
	} else if del_cmd.MatchString(message) {
		kv := del_cmd.FindAllStringSubmatch(message, 1)[0]
		del_replies(kv[1], kv[2], discuss_id, discussnum)
	} else if list_cmd.MatchString(message) {
		kv := list_cmd.FindAllStringSubmatch(message, 1)[0]
		list_replies(kv[1], discuss_id, discussnum)
	} else if tech_cmd.MatchString(message) {
		kv := tech_cmd.FindAllStringSubmatch(message, 1)[0]
		key := strings.TrimSpace(kv[1])
		reply := kv[2]
		tech_replies(key, reply, user_id, discuss_id, discussnum)
		robirt_last_active_for_discuss[discussnum] = time.Now().Add(-30 * time.Second)
	} else {
		if last_active, ok := robirt_last_active_for_discuss[discussnum]; ok && time.Since(last_active).Seconds() < 28 {
			return
		}
		keys := spliteN(message)
		keys = append(keys, message)
		var list []string
		args := []interface{}{}
		args = append(args, discuss_id)
		var sql_parms []string
		for i, key := range keys {
			args = append(args, key)
			sql_parms = append(sql_parms, fmt.Sprintf("$%d", i+2))
		}
		select_str := fmt.Sprintf("select reply from discuss_replies where key in (%s) and discuss_id = $1", strings.Join(sql_parms, ", "))
		rows, err := db.Query(select_str, args...)
		if err == nil {
			for rows.Next() {
				var resp string
				if err := rows.Scan(&resp); err != nil {
					logger.Fatal(err)
				}
				if len(make_tokens(resp)) < 600 {
					list = append(list, resp)
				}
			}
			rows.Close()
		}
		if length := len(list); length > 0 {
			message := list[rand.Intn(length)]
			sendGroupMessage(discussnum, message)
			robirt_last_active_for_discuss[discussnum] = time.Now()
		} else if strings.HasPrefix(message, " ") && strings.HasSuffix(message, "  ") {
			sendGroupMessage(discussnum, message)
		}
	}
}
