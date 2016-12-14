package main

import (
	"regexp"
	"time"
	"sync"
	"log"
)

var groups *Groups = &Groups{sync.RWMutex{},make(map[int64]Group)}
var users *Users = &Users{sync.RWMutex{},make(map[int64]User)}
var timeouts map[string]time.Time = make(map[string]time.Time)
var robirt_last_active map[int64]time.Time = make(map[int64]time.Time)
var robirt_last_active_for_discuss map[int64]time.Time = make(map[int64]time.Time)

var tech_cmd_by_private_message_to_all_groups = regexp.MustCompile("^!addall ((?s:(?:[^=]*\\[CQ:\\w+,\\w+=[\\w\\.]+\\][^=]*)|(?:[^=]+)))=((?s:.+))$")
var tech_cmd_by_private_message_to_one_groups = regexp.MustCompile("^!add (\\d+) ((?s:(?:[^=]*\\[CQ:\\w+,\\w+=[\\w\\.]+\\][^=]*)|(?:[^=]+)))=((?s:.+))$")
var del_cmd_by_private_message_for_all_groups = regexp.MustCompile("^!delall ((?s:(?:[^=]*\\[CQ:\\w+,\\w+=[\\w\\.]+\\][^=]*)|(?:[^=]+)))=((?s:.+))$")
var del_cmd_by_private_message_for_one_groups = regexp.MustCompile("^!add (\\d+) ((?s:(?:[^=]*\\[CQ:\\w+,\\w+=[\\w\\.]+\\][^=]*)|(?:[^=]+)))=((?s:.+))$")

var tech_cmd = regexp.MustCompile("^!add ((?s:(?:[^=]*\\[CQ:\\w+,\\w+=[\\w\\.]+\\][^=]*)|(?:[^=]+)))=((?s:.+))$")
var del_cmd = regexp.MustCompile("^!del ((?s:(?:[^=]*\\[CQ:\\w+,\\w+=[\\w\\.]+\\][^=]*)|(?:[^=]+)))=((?s:.+))$")
var list_cmd = regexp.MustCompile("^!list ((?s:(?:[^=]*\\[CQ:\\w+,\\w+=[\\w\\.]+\\][^=]*)|(?:[^=]+)))$")

var at_regex = regexp.MustCompile("@(\\d+)")

var record = regexp.MustCompile("\\[CQ:record,file=\\w+\\.amr\\]")
var hongbao = regexp.MustCompile("^\\[CQ:hb,id=\\d+,hash=\\w+,title=(.+)\\]$")

var logger *log.Logger = nil