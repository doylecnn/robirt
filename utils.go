package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
)

func (p Params) getInt64(key string) (number int64, err error) {
	err = json.Unmarshal(p[key], &number)
	return
}

func (p Params) getString(key string) (str string, err error) {
	err = json.Unmarshal(p[key], &str)
	return
}

func (p Params) getTime(key string) (t time.Time, err error) {
	var timestamp int64
	err = json.Unmarshal(p[key], &timestamp)
	if err != nil {
		return
	}
	t = time.Unix(timestamp, 0)
	return
}

func reportError(err error) {
	if err != nil {
		errMessage := err.Error()
		logger.Println(errMessage)
		sendPrivateMessage(config.SuperUser.QQNumber, errMessage)
		if err, ok := err.(*pq.Error); ok {
			logger.Println("pq error:", err.Code.Name())
			sendPrivateMessage(config.SuperUser.QQNumber, fmt.Sprintf("Severity:%s\nMessage:%s\nDetail:%s\nHint:%s\nPosition:%s\nInternalPosition:%s\nInternalQuery:%s\nWhere:%s\nSchema:%s\nTable:%s\nColumn:%s\nDataTypeName:%s\nConstraint:%s\nFile:%s\nLine:%s\nRoutine:%s", err.Severity,
				err.Message,
				err.Detail,
				err.Hint,
				err.Position,
				err.InternalPosition,
				err.InternalQuery,
				err.Where,
				err.Schema,
				err.Table,
				err.Column,
				err.DataTypeName,
				err.Constraint,
				err.File,
				err.Line,
				err.Routine))
		}
	}
}

func jsonTrans(json string) string {
	json = strings.Replace(json, "\\", "\\\\", -1)
	json = strings.Replace(json, "\"", "\\\"", -1)
	json = strings.Replace(json, "\n", "\\n", -1)
	json = strings.Replace(json, "\r", "\\r", -1)
	json = strings.Replace(json, "\t", "\\t", -1)
	return json
}

func getGroups(loginQQ int64, cookies string, csrfToken int64) (groups *Groups) {
	if groups == nil {
		groups = &Groups{sync.RWMutex{}, make(map[int64]Group)}
	}

	urlAddr := fmt.Sprintf("http://qun.qzone.qq.com/cgi-bin/get_group_list?groupcount=4&count=4&callbackFun=_GetGroupPortal&uin=%d&g_tk=%d&ua=Mozilla%%2F5.0%%20", loginQQ, csrfToken)
	//logger.Println(url_addr)
	//logger.Println(cookies)

	// proxy, err := url.Parse("http://10.30.3.16:7777")
	// if err != nil {
	// 	logger.Println(err)
	// 	return
	// }

	httpClient := &http.Client{
		Transport: &http.Transport{
			// Proxy:             http.ProxyURL(proxy),
			DisableKeepAlives: true,
		},
	}

	r, err := http.NewRequest("GET", urlAddr, nil)
	if err != nil {
		logger.Println(err)
		return
	}
	r.Header.Set("Cookie", cookies)
	resp, err := httpClient.Do(r)
	if err != nil {
		logger.Println(err)
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Println(err)
		return
	}
	resp.Body.Close()
	groupsResp := string(b)
	if strings.HasPrefix(groupsResp, "_GetGroupPortal_Callback(") && strings.HasSuffix(groupsResp, ");") {
		groupsResp = groupsResp[25 : len(groupsResp)-2]
		groupsjson := GroupsJSON{}
		err = json.Unmarshal([]byte(groupsResp), &groupsjson)
		if err == nil {
			if groupsjson.Data == nil {
				logger.Println(groupsResp)
				return
			}
			for _, g := range groupsjson.Data.Group {
				row := db.QueryRow("select id, name from groups where group_number = $1", g.GroupID)
				var groupName string
				var groupID int64
				err = row.Scan(&groupID, &groupName)
				if err == sql.ErrNoRows {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
					}
					_, err = trans.Exec("insert into groups(group_number, name) values($1, $2)", g.GroupID, g.GroupName)
					if err != nil {
						reportError(err)
						trans.Rollback()
					} else {
						trans.Commit()
					}
				} else if err != nil {
					reportError(err)
				} else if groupName != g.GroupName {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
					}
					_, err = trans.Exec("update groups set name = $1 where group_number = $2", g.GroupName, g.GroupID)
					if err != nil {
						reportError(err)
						trans.Rollback()
					} else {
						trans.Commit()
					}
				}
			}
		} else {
			reportError(err)
			return
		}

		rows, err := db.Query("select id, group_number, name from groups")
		if err != nil {
			panic(err)
		}
		rows.Close()

		i := 0
		groups.RWLocker.Lock()
		for rows.Next() {
			var groupName string
			var groupID int64
			var groupNum int64
			rows.Scan(&groupID, &groupNum, &groupName)
			group := Group{}
			group.ID = groupID
			group.GroupNum = groupNum
			group.GroupName = groupName
			group.Members = getGroupMembers(group, LoginQQ, Cookies, CsrfToken)
			groups.Groups[groupNum] = group
			logger.Printf("%d: %s[%d]", i, group.GroupName, groupNum)
			i++
		}
		groups.RWLocker.Unlock()
	} else {
		logger.Printf(groupsResp)
		logger.Println(urlAddr)
	}
	return
}

func getGroupMembers(group Group, loginQQ int64, cookies string, csrfToken int64) (nicknamesInGroup *Members) {
	nicknamesInGroup = &Members{sync.RWMutex{}, make(map[int64]Member)}

	urlAddr := fmt.Sprintf("http://qun.qzone.qq.com/cgi-bin/get_group_member?callbackFun=_GroupMember&uin=%d&groupid=%d&neednum=1&r=0.5421284231954122&g_tk=%d&ua=Mozilla%%2F4.0%%20(compatible%%3B%%20MSIE%%207.0%%3B%%20Windows%%20NT%%205.1%%3B%%20Trident%%2F4.0)", loginQQ, group.GroupNum, csrfToken)
	//logger.Println(url_addr)
	//logger.Println(cookies)

	// proxy, err := url.Parse("http://10.30.3.16:7777")
	// if err != nil {
	// 	logger.Println(err)
	// 	return
	// }

	httpClient := &http.Client{
		Transport: &http.Transport{
			// Proxy:             http.ProxyURL(proxy),
			DisableKeepAlives: true,
		},
	}

	r, err := http.NewRequest("GET", urlAddr, nil)
	if err != nil {
		logger.Println(err)
		return
	}
	r.Header.Set("Cookie", cookies)
	resp, err := httpClient.Do(r)
	if err != nil {
		logger.Println(err)
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Println(err)
		return
	}
	resp.Body.Close()
	groupMembersResp := string(b)
	if strings.HasPrefix(groupMembersResp, "_GroupMember_Callback(") && strings.HasSuffix(groupMembersResp, ");") {
		groupMembersResp = groupMembersResp[22 : len(groupMembersResp)-2]
		memberjson := GroupMembersJSON{}
		err = json.Unmarshal([]byte(groupMembersResp), &memberjson)
		if err == nil {
			if memberjson.Data == nil {
				logger.Println(groupMembersResp)
				return
			}
			for _, m := range memberjson.Data.Item {
				var userID, memberID int64
				userID, err = getUserID(m.Uin)
				if err == sql.ErrNoRows {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
						continue
					}
					err = trans.QueryRow("insert into users(qq_number) values($1) returning id", m.Uin).Scan(&userID)
					if err != nil {
						logger.Println(err)
						trans.Rollback()
						continue
					} else {
						trans.Commit()
					}
				} else if err != nil {
					logger.Println(err)
					continue
				}
				user := User{userID, m.Uin, ""}

				row := db.QueryRow("SELECT id, nickname, rights FROM group_members where group_id = $1 and user_id = $2", group.ID, user.ID)
				var nickname string
				var rights int
				err = row.Scan(&memberID, &nickname, &rights)
				if err == sql.ErrNoRows {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
						continue
					}
					err = trans.QueryRow("insert into group_members(group_id, user_id, nickname) values($1, $2, $3)  returning id", group.ID, user.ID, m.Nick).Scan(&memberID)
					if err != nil {
						reportError(err)
						trans.Rollback()
					} else {
						trans.Commit()
					}
				} else if err != nil {
					reportError(err)
				} else if nickname != m.Nick {
					trans, err := db.Begin()
					if err != nil {
						reportError(err)
						continue
					}
					_, err = trans.Exec("update group_members set nickname = $1 where group_id = $2 and user_id = $3", m.Nick, group.ID, user.ID)
					if err != nil {
						reportError(err)
						trans.Rollback()
					} else {
						trans.Commit()
					}
				}
			}
		} else {
			reportError(err)
			return
		}

		rows, err := db.Query("SELECT n.id, n.user_id, u.qq_number,n.nickname,n.rights FROM group_members n, users u where n.user_id = u.id and n.group_id=$1", group.ID)
		if err != nil {
			panic(err)
		}
		rows.Close()
		nicknamesInGroup.RWLocker.Lock()
		for rows.Next() {
			var nickname string
			var userQQ, userID, memberID int64
			var rights int
			err = rows.Scan(&memberID, &userID, &userQQ, &nickname, &rights)
			if err != nil {
				panic(err)
			}
			nicknamesInGroup.Members[userQQ] = Member{memberID, userID, group.ID, nickname, rights}
		}
		nicknamesInGroup.RWLocker.Unlock()
	} else {
		logger.Println(groupMembersResp)
		logger.Println(urlAddr)
	}
	return
}

func spliteN(s string) (result []string) {
	max := 1000
	s = strings.TrimSpace(s)
	r := makeTokens(s)
	if len(r) < 2 || len(r) > 30 {
		return result
	}
	var c = 0
	for i := 1; i <= len(r); i++ {
		for j := 0; j < len(r) && j < i; j++ {
			key := tokensToString(r[j:i])
			if (i-j > 1 || len([]rune(key)) > 1) && i-j < 10 {
				contain := false
				for _, item := range result {
					if item == key {
						contain = true
						break
					}
				}
				if !contain {
					result = append(result, key)
					c++
					if c > max {
						return
					}
				}
			}
		}
	}
	return result
}

func getGroupID(groupNum int64) (groupID int64, err error) {
	if group, ok := groups.getGroup(groupNum); ok {
		groupID = group.ID
		return
	}
	row := db.QueryRow("select id, name from groups where group_number = $1", groupNum)
	var groupName string
	err = row.Scan(&groupID, &groupName)
	g := Group{}
	g.ID = groupID
	g.GroupNum = groupNum
	g.GroupName = groupName
	g.Members = getGroupMembers(g, LoginQQ, Cookies, CsrfToken)
	if groups == nil {
		groups = &Groups{}
	}
	groups.RWLocker.Lock()
	groups.Groups[groupNum] = g
	groups.RWLocker.Unlock()
	return
}

func getDiscussID(discussNum int64) (discussID int64, err error) {
	row := db.QueryRow("select id from discusses where discuss_number = $1", discussNum)
	err = row.Scan(&discussID)
	if err != nil {
		err = db.QueryRow("insert into discusses (discuss_number) values ($1) returning id", discussNum).Scan(&discussID)
	}
	return
}

func getUserID(qqNum int64) (userID int64, err error) {
	if user, ok := users.getUser(qqNum); ok {
		userID = user.ID
		return
	}
	row := db.QueryRow("select id, qq_name from users where qq_number = $1", qqNum)
	var name sql.NullString
	err = row.Scan(&userID, &name)
	if users == nil {
		users = &Users{}
	}
	users.RWLocker.Lock()
	if name.Valid {
		users.Users[qqNum] = User{userID, qqNum, name.String}
	} else {
		users.Users[qqNum] = User{userID, qqNum, ""}
	}
	users.RWLocker.Unlock()
	return
}
