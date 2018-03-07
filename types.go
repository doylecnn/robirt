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

// User user info
type User struct {
	ID     int64
	QQNum  int64
	QQName string
}

// Member member info
type Member struct {
	ID                      int64
	UserID                  int64
	GroupID                 int64
	GroupNum                int64  `json:"group_id"`
	QQNum                   int64  `json:"qq_id"`
	Nickname                string `json:"nickname"`
	Namecard                string `json:"namecard"`
	Sex                     int32  `json:"sex"`
	Age                     int32  `json:"age"`
	Area                    string `json:"area"`
	JoinTime                int32  `json:"join_time"`
	LastActive              int32  `json:"last_active"`
	LevelName               string `json:"level_name"`
	Permission              int32  `json:"permission"`
	BadRecord               bool   `json:"bad_record"`
	SpecialTitle            string `json:"special_title"`
	SpecialTitleExpressTime int32  `json:"special_title_express_time"`
	AllowModifyNamecard     bool   `json:"allow_modify_namecard"`
}

// Group group info
type Group struct {
	ID        int64
	GroupNum  int64  `json:"id"`
	GroupName string `json:"name"`
	Members   *sync.Map
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
