package main

import (
	"fmt"
	"log"
	"math/rand"

	"strings"
	"time"
)

func del_disc_replies(key, reply string, group_id, groupnum int64) {
	trans, err := db.Begin()
	if err != nil {
		reportError(err)
		return
	}
	sql_result, err := trans.Exec("delete from replies where key = $1 and reply = $2 and group_id = $3", key, reply, group_id)
	if err != nil {
		reportError(err)
		trans.Rollback()
		return
	}
	trans.Commit()

	if affect_rows, err := sql_result.RowsAffected(); affect_rows > 0 {
		sendGroupMessage(groupnum, fmt.Sprintf("已删除：key=%s, value=%s", key, reply))
	} else {
		reportError(err)
	}
}

func list_disc_replies(key string, group_id, groupnum int64) {
	rows, err := db.Query("select reply from replies where key = $1 and group_id = $2", key, group_id)
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
		sendGroupMessage(groupnum, strings.Join(list, "\n"))
	}
}

func tech_disc_replies(key, reply string, userid, groupid, groupnum int64) {
	if length := message_length(reply); length > 600 {
		log.Println("too long!", length)
		sendGroupMessage(groupnum, "太长记不住！")
	} else {
		if len(key) < 2 {
			sendGroupMessage(groupnum, "触发字太短了！")
		} else {
			row := db.QueryRow("select count(1) from replies where key = $1 and reply = $2 and group_id = $3", key, reply, groupid)
			var count int
			row.Scan(&count)
			if count == 0 {
				trans, err := db.Begin()
				if err != nil {
					reportError(err)
					return
				}
				sql_result, err := trans.Exec("insert into replies(key, reply, author_id, group_id) values($1, $2, $3, $4)", key, reply, userid, groupid)
				if err != nil {
					reportError(err)
					trans.Rollback()
					return
				}
				if affect_rows, err := sql_result.RowsAffected(); affect_rows > 0 {
					sendGroupMessage(groupnum, fmt.Sprintf("你说 “%s” 我说 “%s”", key, reply))
				} else {
					reportError(err)
					trans.Rollback()
					return
				}
				trans.Commit()
			}
			log.Println(key)
		}
	}
}

func event_discuss_message(p Params) {
	message, _ := p.GetString("message")
	anonymousname, _ := p.GetString("anonymousname")
	qqnum, _ := p.GetInt64("qqnum")
	groupnum, _ := p.GetInt64("groupnum")
	user_id, _ := get_user_id(qqnum)
	group_id, _ := get_group_id(groupnum)
	var nickname string
	if len(anonymousname) > 0 {
		nickname = "[匿名]" + anonymousname
	} else {
		groups.RWLocker.RLock()
		group := groups.Map[groupnum]
		groups.RWLocker.RUnlock()
		members := group.Members
		members.RWLocker.RLock()
		nickname = members.Map[qqnum].Nickname
		members.RWLocker.RUnlock()
	}

	var group Group
	groups.RWLocker.RLock()
	for _, g := range groups.Map {
		if g.GroupNo == groupnum {
			group = g
			break
		}
	}
	groups.RWLocker.RUnlock()

	fmt.Printf("%s(%d)-%s(%d):\n%s\n", group.GroupName, groupnum, nickname, qqnum, message)

	if "!help" == message || "！help" == message || "/help" == message {
		sendGroupMessage(groupnum, "!add 触发字=触发内容  添加一条\n!del 触发字=触发内容  删除一条\n!list 触发字  列出该触发字下的所有条目\n没有其他的了……")
	} else if del_cmd.MatchString(message) {
		kv := del_cmd.FindAllStringSubmatch(message, 1)[0]
		del_replies(kv[1], kv[2], group_id, groupnum)
	} else if list_cmd.MatchString(message) {
		kv := list_cmd.FindAllStringSubmatch(message, 1)[0]
		list_replies(kv[1], group_id, groupnum)
	} else if tech_cmd.MatchString(message) {
		kv := tech_cmd.FindAllStringSubmatch(message, 1)[0]
		key := strings.TrimSpace(kv[1])
		reply := kv[2]
		tech_replies(key, reply, user_id, group_id, groupnum)
		robirt_last_active[groupnum] = time.Now().Add(-30 * time.Second)
	} else if hongbao.MatchString(message) {
		kv := hongbao.FindAllStringSubmatch(message, 1)[0]
		if kv[1] == "我的天哪！土豪发红包啦！大家快抢啊！" {
			sendGroupMessage(groupnum, "我的天哪！！土豪发红包啦！！大家快抢啊！！")
		} else {
			sendGroupMessage(groupnum, "我的天哪！土豪发红包啦！大家快抢啊！")
		}
		d, _ := time.ParseDuration("300s")
		robirt_last_active[groupnum] = time.Now().Add(d)
	} else {
		if groupnum == 171712942 && strings.Contains(message, "[CQ:image,file=0C7D59F95205A8F6B9503FB55212C48D.") {
			go func() {
				time.Sleep(3 * time.Second)
				GroupBan(groupnum, qqnum, 30)
			}()
		} else if last_active, ok := robirt_last_active[groupnum]; ok && time.Since(last_active).Seconds() < 28 {
			return
		}
		keys := spliteN(message)
		keys = append(keys, message)
		var list []string
		args := []interface{}{}
		args = append(args, group_id)
		var sql_parms []string
		for i, key := range keys {
			args = append(args, key)
			sql_parms = append(sql_parms, fmt.Sprintf("$%d", i+2))
		}
		select_str := fmt.Sprintf("select reply from replies where key in (%s) and group_id = $1", strings.Join(sql_parms, ", "))
		rows, err := db.Query(select_str, args...)
		if err == nil {
			for rows.Next() {
				var resp string
				if err := rows.Scan(&resp); err != nil {
					log.Fatal(err)
				}
				if message_length(resp) < 600 {
					list = append(list, resp)
				}
			}
			rows.Close()
		}
		if length := len(list); length > 0 {
			message := list[rand.Intn(length)]
			sendGroupMessage(groupnum, message)
			robirt_last_active[groupnum] = time.Now()
		} else if strings.HasPrefix(message, " ") && strings.HasSuffix(message, "  ") {
			sendGroupMessage(groupnum, message)
		}
	}
}
