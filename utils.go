package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

func (p Params) UnmarshalGroupList(v *[]Group) error {
	return json.Unmarshal(p["group_list"], v)
}

func (p Params) UnmarshalGroupMemberList(v *[]Member) error {
	return json.Unmarshal(p["member_list"], v)
}

func (p Params) UnmarshalGroupMemberInfo(v *Member) error {
	return json.Unmarshal(p["member"], v)
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
	if group, ok := groups.Load(groupNum); ok {
		groupID = group.(Group).ID
		return
	}
	row := db.QueryRow("select id, name from groups where group_number = $1", groupNum)
	var groupName string
	err = row.Scan(&groupID, &groupName)
	g := Group{}
	g.ID = groupID
	g.GroupNum = groupNum
	g.GroupName = groupName
	groups.Store(groupNum, g)
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
	if user, ok := users.Load(qqNum); ok {
		userID = user.(User).ID
		return
	}
	row := db.QueryRow("select id, qq_name from users where qq_number = $1", qqNum)
	var name sql.NullString
	err = row.Scan(&userID, &name)
	if name.Valid {
		users.Store(qqNum, User{userID, qqNum, name.String})
	} else {
		users.Store(qqNum, User{userID, qqNum, ""})
	}
	return
}
