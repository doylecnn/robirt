package main

import (
	"encoding/json"
	"strings"
	"sync"
)

type tomlConfig struct {
	SuperUser superUserConfig
	Database  databaseConfig
	Robirt    robirtConfig
}

type superUserConfig struct {
	QQNumber int64
}

type databaseConfig struct {
	DBName string
}

type robirtConfig struct {
	Cdtime float64
}

// Notification json part
type Notification struct {
	// Method
	Method string `json:"method"`

	// Params
	Params Params `json:"params"`
}

// Params json part
type Params map[string]json.RawMessage

// GroupsJSON json info for group
type GroupsJSON struct {
	Code int `json:"code"`
	Data *struct {
		Group []struct {
			Auth      int    `json:"auth"`
			Flag      int    `json:"flag"`
			GroupID   int    `json:"groupid"`
			GroupName string `json:"groupname"`
		} `json:"group"`
		Total int `json:"total"`
	} `json:"data"`
	Default int    `json:"default"`
	Message string `json:"message"`
	Subcode int    `json:"subcode"`
}

// GroupMembersJSON json info for group member
type GroupMembersJSON struct {
	Code int `json:"code"`
	Data *struct {
		Alpha      int    `json:"alpha"`
		BbsCount   int    `json:"bbscount"`
		Class      int    `json:"class"`
		CreateTime int    `json:"create_time"`
		FileCount  int    `json:"filecount"`
		FingerMemo string `json:"finger_memo"`
		GroupMemo  string `json:"group_memo"`
		GroupName  string `json:"group_name"`
		Item       []struct {
			Iscreator int    `json:"iscreator"`
			Ismanager int    `json:"ismanager"`
			Nick      string `json:"nick"`
			Uin       int64  `json:"uin"`
		} `json:"item"`
		Level  int    `json:"level"`
		Nick   string `json:"nick"`
		Option int    `json:"option"`
		Total  int    `json:"total"`
	} `json:"data"`
	Default int    `json:"default"`
	Message string `json:"message"`
	Subcode int    `json:"subcode"`
}

// User user info
type User struct {
	ID     int64
	QQNum  int64
	QQName string
}

// Member member info
type Member struct {
	ID       int64
	UserID   int64
	GroupID  int64
	Nickname string
	Rights   int
}

// Members members map
type Members struct {
	RWLocker sync.RWMutex
	Members  map[int64]Member
}

func (m *Members) getMember(qqNum int64) (member Member, ok bool) {
	m.RWLocker.RLock()
	defer m.RWLocker.RUnlock()
	member, ok = m.Members[qqNum]
	return
}

// Group group info
type Group struct {
	ID        int64
	GroupNum  int64
	GroupName string
	Members   *Members
}

type token struct {
	V []rune
}

func (t *token) String() string {
	return string(t.V)
}

// tokensToString used to get string
func tokensToString(t []token) string {
	r := []string{}
	for _, v := range t {
		r = append(r, string(v.V))
	}
	return strings.Join(r, "")
}

func makeTokens(s string) (result []token) {
	result = []token{}
	s = strings.TrimSpace(s)
	r := []rune(s)
	flag := false
	var startIdx = 0
	var endIdx = 0
	for idx, c := range r {
		if c == '[' {
			flag = true
			startIdx = idx
			continue
		}
		if flag && c != ']' {
			continue
		}
		if c == ']' {
			flag = false
			endIdx = idx + 1
			result = append(result, token{r[startIdx:endIdx]})
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			continue
		}
		result = append(result, token{r[idx : idx+1]})
	}
	return
}
