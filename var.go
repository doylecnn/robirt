package main

import (
	"log"
	"regexp"
	"sync"
	"time"
)

var groups = &Groups{sync.RWMutex{}, make(map[int64]Group)}
var users = &Users{sync.RWMutex{}, make(map[int64]User)}
var timeouts = make(map[string]time.Time)
var robirtLastActive = make(map[int64]time.Time)
var robirtLastActiveForDiscuss = make(map[int64]time.Time)

var techCmdByPrivateMessageToAllGroups = regexp.MustCompile(`^!addall ((?s:(?:[^=]*\[CQ:\w+,\w+=[\w\.]+\][^=]*)|(?:[^=]+)))=((?s:.+))$`)
var techCmdByPrivateMessageToGroup = regexp.MustCompile(`^!add (\d+) ((?s:(?:[^=]*\[CQ:\w+,\w+=[\w\\.]+\][^=]*)|(?:[^=]+)))=((?s:.+))$`)
var delCmdByPrivateMessageForAllGroups = regexp.MustCompile(`^!delall ((?s:(?:[^=]*\[CQ:\w+,\w+=[\w\.]+\][^=]*)|(?:[^=]+)))=((?s:.+))$`)
var delCmdByPrivateMessageForGroup = regexp.MustCompile(`^!add (\d+) ((?s:(?:[^=]*\[CQ:\w+,\w+=[\w\.]+\][^=]*)|(?:[^=]+)))=((?s:.+))$`)

var techCmd = regexp.MustCompile(`^!add ((?s:(?:[^=]*\[CQ:\w+,\w+=[\w\.]+\][^=]*)|(?:[^=]+)))=((?s:.+))$`)
var delCmd = regexp.MustCompile(`^!del ((?s:(?:[^=]*\[CQ:\w+,\w+=[\w\.]+\][^=]*)|(?:[^=]+)))=((?s:.+))$`)
var listCmd = regexp.MustCompile(`^!list ((?s:(?:[^=]*\[CQ:\w+,\w+=[\w\.]+\][^=]*)|(?:[^=]+)))$`)

var atRegex = regexp.MustCompile(`@(\d+)`)

// var record = regexp.MustCompile(`\[CQ:record,file=\w+\.amr\]`)
var hongbao = regexp.MustCompile(`^\[CQ:hb,id=\d+,hash=\w+,title=(.+)\]$`)

var logger *log.Logger
