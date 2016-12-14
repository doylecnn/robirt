package main

import (
	"encoding/json"
	"sync"
	"strings"
)

type tomlConfig struct {
	SuperUser superUserConfig
	Database  databaseConfig
	Robirt	  robirtConfig
}

type superUserConfig struct {
	QQNumber int64
}

type databaseConfig struct {
	DBName string
}

type robirtConfig struct{
	Cdtime float64
}

type Notification struct {
	Method string `json:"method"`
	Params Params `json:"params"`
}

type Params map[string]json.RawMessage

type GetGroupsJson struct {
	Code int `json:"code"`
	Data struct {
		Group []struct {
			Auth      int    `json:"auth"`
			Flag      int    `json:"flag"`
			Groupid   int    `json:"groupid"`
			Groupname string `json:"groupname"`
		} `json:"group"`
		Total int `json:"total"`
	} `json:"data"`
	Default int    `json:"default"`
	Message string `json:"message"`
	Subcode int    `json:"subcode"`
}

type GetGroupMembersJson struct {
	Code int `json:"code"`
	Data struct {
		Alpha      int    `json:"alpha"`
		Bbscount   int    `json:"bbscount"`
		Class      int    `json:"class"`
		CreateTime int    `json:"create_time"`
		Filecount  int    `json:"filecount"`
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

type User struct {
	Id     int64
	QQNo   int64
	QQName string
}

type Member struct {
	Id       int64
	User_Id  int64
	Group_Id int64
	Nickname string
	Rights   int
}

type Users struct{
	RWLocker sync.RWMutex
	Map map[int64]User
}

type Members struct{
	RWLocker sync.RWMutex
	Map map[int64]Member
}

type Group struct {
	Id        int64
	GroupNo   int64
	GroupName string
	Members *Members
}

type Groups struct{
	RWLocker sync.RWMutex
	Map map[int64]Group
}

type token struct {
	V []rune
}

func (t *token) String() string {
	return string(t.V)
}

func TokensToString(t []token) string {
	r := []string{}
	for _, v := range t {
		r = append(r, string(v.V))
	}
	return strings.Join(r,"")
}

func make_tokens(s string) (result []token) {
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