package main

import (
	"encoding/json"
)

type tomlConfig struct {
	SuperUser superUserConfig
	Database  databaseConfig
}

type superUserConfig struct {
	QQNumber int64
}

type databaseConfig struct {
	DBName string
}

type Notification struct {
	Method string `json:"method"`
	Params Params `json:"params"`
}

type Group struct {
	Id        int64
	GroupNo   int64
	GroupName string
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
