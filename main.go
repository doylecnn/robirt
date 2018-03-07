package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"
)

var (
	db            *sql.DB
	config        tomlConfig
	clientConn    *net.TCPConn
	closeSignChan = make(chan struct{})
	requestChan   = make(chan Notification, 1)
	gLogFile      *os.File
	LoginQQ       int64
)

func main() {
	rand.Seed(42)
	var err error
	logFilename := "robirt.log"
	gLogFile, err = os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		fmt.Printf("open file error=%s\r\n", err.Error())
		os.Exit(-1)
	}
	writers := []io.Writer{
		gLogFile,
		os.Stdout,
	}
	fileAndStdoutWriter := io.MultiWriter(writers...)
	logger = log.New(fileAndStdoutWriter, "", log.Ldate|log.Ltime|log.Lshortfile)

	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		logger.Println(err)
		return
	}

	db, err = sql.Open("postgres", config.Database.DBName)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(60)
	db.SetMaxIdleConns(5)

	go serverStart()
	go func() {
		localAddress, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:7008")
		if err != nil {
			logger.Fatalf("ResolveTCPAddr Error: %v\n", err)
		}
		ln, err := net.ListenTCP("tcp4", localAddress)
		if err != nil {
			logger.Fatalf("Failed to listening server: %v", err)
		}
		logger.Println("Listening server on tcp:127.0.0.1:7008")

		for {
			conn, err := ln.Accept()
			if err != nil {
				logger.Printf("Accept Error: %v\n", err)
				continue
			}
			go handleRequest(conn)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cmd(scanner.Text())
		}
	}()
	<-closeSignChan
}

func cmd(cmd string) {
	if cmd == "exit" {
		close(closeSignChan)
		return
	} else if cmd == "init" {
		getLoginQQ()
		getGroupList()
		return
	} else if strings.HasPrefix(cmd, "random:") {
		var i int32 = 0
		var target = rand.Int31n(300)
		groups.Range(func(key, value interface{}) bool {
			if i == target {
				sendGroupMessage(value.(Group).GroupNum, cmd[7:])
				return false
			}
			i++
			return true
		})
		return
	} else if strings.HasPrefix(cmd, "level:") {
		groupNum, err := strconv.ParseInt(strings.TrimSpace(cmd[6:]), 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		if _, ok := groups.Load(groupNum); ok {
			leaveGroup(groupNum, LoginQQ)
			getGroupList()
		}
		return
	}
	s := strings.SplitN(strings.TrimSpace(cmd), ":", 2)
	if len(s) == 2 {
		groupNum, err := strconv.ParseInt(strings.TrimSpace(s[0]), 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}

		if g, ok := groups.Load(groupNum); ok {
			sendGroupMessage(g.(Group).GroupNum, s[1])
		} else {
			fmt.Println("group not found!")
		}
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	tmpbyte := make([]byte, 4096)
	tmpbyte = tmpbyte[:0]
	buf := make([]byte, 4096*16)
	for {
		// Read the incoming connection into the buffer.
		reqLen, err := conn.Read(buf)
		if err != nil {
			logger.Println("Error reading:", err.Error())
			return
		}
		if reqLen == 0 {
			continue
		}
		scanner := bufio.NewScanner(bytes.NewReader(buf[:reqLen]))
		for scanner.Scan() {
			b := bytes.TrimSpace(scanner.Bytes())
			if len(b) == 0 {
				logger.Println("len(b)==0", string(b))
				continue
			} else if len(tmpbyte) > 0 && len(tmpbyte)+len(b) < 4096 {
				b = append(tmpbyte, b...)
				logger.Printf("retry b := %s\n", string(b))
			} else {
				tmpbyte = tmpbyte[:0]
			}
			var js Notification
			err = json.Unmarshal(b, &js)
			if err != nil {
				if serr, ok := err.(*json.SyntaxError); ok {
					logger.Printf("%s, %s\n", serr.Error(), string(b))
					if errStr := serr.Error(); strings.HasPrefix(errStr, "invalid character") && strings.HasSuffix(errStr, "in string literal") {
						tmpbyte = tmpbyte[:0]
						continue
					}
					tmpbyte = append(tmpbyte, b...)
				} else {
					logger.Printf("%s, %s\n", err.Error(), string(b))
					tmpbyte = tmpbyte[:0]
				}
				continue
			}
			requestChan <- js
			tmpbyte = tmpbyte[:0]
		}
	}
}

func serverStart() {
	eventLoop()
}

func eventLoop() {
	for {
		js := <-requestChan

		if js.Method == "LoginQq" {
			LoginQQ, _ = js.Params.getInt64("loginqq")
			logger.Printf(">>> %d\n", LoginQQ)
			continue
		}

		subtype, _ := js.Params.getInt64("subtype")

		switch js.Method {
		case "GroupMessage":
			go groupMessageHandle(js.Params)
		case "DiscussMessage":
			go discussMessageHandle(js.Params)
		case "PrivateMessage":
			go privateMessageHandle(js.Params)
		case "GroupMemberJoin":
			qqNum, _ := js.Params.getInt64("qqnum")
			groupNum, _ := js.Params.getInt64("groupnum")
			beingOperateQQ, _ := js.Params.getInt64("opqqnum")
			message := welcomeNewMember(subtype, groupNum, qqNum, beingOperateQQ)
			sendGroupMessage(groupNum, message)
			getGroupMemberList(groupNum)
		case "GroupMemberLeave":
			qqNum, _ := js.Params.getInt64("qqnum")
			groupNum, _ := js.Params.getInt64("groupnum")
			if groupNum == 196656732 {
				continue
			}
			//beingOperateQQ := js.Params.GetInt64("opqqnum")
			if v, ok := groups.Load(groupNum); ok {
				group := v.(Group)
				members := group.Members
				if members == nil {
					continue
				}
				if v, ok := members.Load(qqNum); ok {
					member := v.(Member)
					if subtype == 1 {
						message := fmt.Sprintf("群员:[%s] 退群了!!!", member.Nickname)
						sendGroupMessage(groupNum, message)
					} else if subtype == 2 {
						message := fmt.Sprintf("群员:[%s] 被 某个管理员 踢了!!!", member.Nickname)
						sendGroupMessage(groupNum, message)
					}
				}
			}
		case "RequestAddFriend":
			responseFlag, _ := js.Params.getString("response_flag")
			addFriend(responseFlag, 1, "")
		case "RequestAddGroup":
			responseFlag, _ := js.Params.getString("response_flag")
			if subtype == 2 {
				addGroup(responseFlag, 2, 1, "")
			}
			getGroupList()
		case "GroupMemberList":
			var groupMemberList []Member
			if err := js.Params.UnmarshalGroupMemberList(&groupMemberList); err != nil {
				logger.Printf(">>> get group member list faild: %v", err)
			}
			logger.Printf(">>> %s\n", groupMemberList)
			updateGroupMember(groupMemberList)
		case "GroupList":
			var grouplist []Group
			if err := js.Params.UnmarshalGroupList(&grouplist); err != nil {
				logger.Printf(">>> get group list faild: %v", err)
			}
			logger.Printf(">>> %s\n", grouplist)
			updateGroupList(grouplist)
		case "GroupMemberInfo":
			var memberInfo Member
			if err := js.Params.UnmarshalGroupMemberInfo(&memberInfo); err != nil {
				logger.Printf(">>> get member info faild: %v", err)
			}
			logger.Printf(">>> %s\n", memberInfo)
			updateMemberInfo(memberInfo)
		default:
			logger.Printf("未处理：%s\n", js)
		}
	}
}

func GetGroupListFromDB() {
	rows, err := db.Query("select id, group_number, name from groups")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var groupName string
		var groupID int64
		var groupNum int64
		rows.Scan(&groupID, &groupNum, &groupName)

		if v, ok := groups.Load(groupNum); ok {
			group := v.(Group)
			if groupName != group.GroupName {
				group.GroupName = groupName
			}
		} else {
			group := Group{}
			group.ID = groupID
			group.GroupNum = groupNum
			group.GroupName = groupName
			group.Members = GetGroupMembersFromDB(group.ID, group.GroupNum)
			groups.Store(groupNum, group)
		}
	}
}

func GetGroupMembersFromDB(groupId int64, groupNumbber int64) *sync.Map {
	memberList := new(sync.Map)
	rows, err := db.Query("select m.id, m.user_id, u.qq_number, m.Nickname, m.Rights from group_members m join users u on m.user_id = u.id where m.group_id = $1", groupId)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var userId int64
		var qq_number int64
		var nickname string
		var rights int32
		rows.Scan(&id, &userId, &qq_number, &nickname, &rights)
		m := Member{}
		m.ID = id
		m.UserID = userId
		m.GroupID = groupId
		m.GroupNum = groupNumbber
		m.QQNum = qq_number
		m.Nickname = nickname
		m.Permission = rights
		memberList.Store(qq_number, m)
	}
	return memberList
}

func updateGroupList(groupList []Group) {
	if len(groupList) == 0 {
		return
	}
	GetGroupListFromDB()
	var groupNums []int64 = []int64{}
	for _, ng := range groupList {
		groupNums = append(groupNums, ng.GroupNum)
		if v, ok := groups.Load(ng.GroupNum); ok {
			og := v.(Group)
			if og.GroupName != ng.GroupName {
				og.GroupName = ng.GroupName
				trans, err := db.Begin()
				if err != nil {
					//reportError(err)
					continue
				}
				_, err = trans.Exec("update groups set name = $1 where Id = $2", ng.GroupName, og.ID)
				if err != nil {
					//reportError(err)
					trans.Rollback()
					continue
				} else {
					trans.Commit()
				}
			}
		} else {
			var groupID int64
			trans, err := db.Begin()
			if err != nil {
				//reportError(err)
				continue
			}
			err = trans.QueryRow("insert into groups(group_number, name) values($1, $2) returning id", ng.GroupNum, ng.GroupName).Scan(&groupID)
			if err != nil {
				//reportError(err)
				trans.Rollback()
				continue
			} else {
				trans.Commit()
			}
			groups.Store(groupID, Group{ID: groupID, GroupNum: ng.GroupNum, GroupName: ng.GroupName, Members: nil})
			getGroupMemberList(ng.GroupNum)
		}
	}
	var waitForDeleteGroupNums []int64 = []int64{}
	groups.Range(func(key, value interface{}) bool {
		found := false
		for _, num := range groupNums {
			if num == key {
				found = true
				break
			}
		}
		if !found {
			g := value.(Group)
			waitForDeleteGroupNums = append(waitForDeleteGroupNums, g.GroupNum)
			trans, err := db.Begin()
			if err != nil {
				return true
			}
			_, err = trans.Exec("delete from replies where group_id = $1", g.ID)
			if err != nil {
				trans.Rollback()
				return true
			}
			_, err = trans.Exec("delete from group_members where group_id = $1", g.ID)
			if err != nil {
				trans.Rollback()
				return true
			}
			_, err = trans.Exec("delete from groups where id = $1", g.ID)
			if err != nil {
				trans.Rollback()
			} else {
				trans.Commit()
			}
		}
		return true
	})
	for _, num := range waitForDeleteGroupNums {
		groups.Delete(num)
	}
}

func updateGroupMember(groupMemberList []Member) {
	flag := true
	for _, nm := range groupMemberList {
		if v, ok := groups.Load(nm.GroupNum); ok {
			g := v.(Group)
			if flag {
				g.Members = GetGroupMembersFromDB(g.ID, g.GroupNum)
				flag = false
			}
			if _, ok := g.Members.Load(nm.QQNum); !ok {
				g.Members.Store(nm.QQNum, nm)
				trans, err := db.Begin()
				if err != nil {
					//reportError(err)
					continue
				}
				var userID int64
				err = trans.QueryRow("select id from users where qq_numer = $1", nm.QQNum).Scan(&userID)
				if err != nil {
					//reportError(err)
					trans.Rollback()
					continue
				}
				_, err = trans.Exec("insert into group_members(group_id, user_id, nickname, rights) values($1, $2, $3, $4) returning id", g.ID, userID, nm.Nickname, nm.Permission)
				if err != nil {
					//reportError(err)
					trans.Rollback()
					continue
				} else {
					trans.Commit()
				}
			}
		}
	}
}

func updateMemberInfo(memberInfo Member) {
	if v, ok := groups.Load(memberInfo.GroupNum); ok {
		g := v.(Group)
		if _, ok := g.Members.Load(memberInfo.QQNum); !ok {
			g.Members.Store(memberInfo.QQNum, memberInfo)
		}
	}
}

func welcomeNewMember(subtype, groupNo, QQNum, operateQQ int64) (message string) {
	var newbeMission string
	if groupNo == 171712942 {
		newbeMission = "新手四项任务：\n  1.修改群名片（群名片格式：游戏id + 活动区域）\n  2.上传带游戏id 的游戏内截图（上传到群内新人报道相册）\n  3.完成游戏自带training(游戏主界面右上角ops->training下所有项目)\n  4.阅读 ingress 新手指南: http://mp.weixin.qq.com/s?__biz=MzIxNTI4ODU1OA==&mid=403604670&idx=1&sn=1b74a16225deebefe9fcb81e09a39477&scene=18\n  5.不要轻易相信群里自称托马西的\n欢迎提其它问题"
	} else if groupNo == 147798016 {
		newbeMission = "新手五项任务：\n  1.修改群名片（群名片格式：游戏等级-游戏id-活动区域）\n  2.上传带游戏id 的游戏内截图（上传到群内新人报道相册）\n  3.完成游戏自带training(游戏主界面右上角ops->training下所有项目)\n  4.阅读ingress 新手指南: http://mp.weixin.qq.com/s?__biz=MzIxNTI4ODU1OA==&mid=403604670&idx=1&sn=1b74a16225deebefe9fcb81e09a39477&scene=18\n  5.汉子请爆照, 妹子自动免除此条\n欢迎提其它问题"
	} else if groupNo == 196656732 {
		newbeMission = `欢迎加入南京蓝色抵抗军大家庭^_^
建议新人按顺序完成以下事宜：
 1. 修改群名片（格式：游戏id-活动区域）。
 2. 上传带游戏id 的游戏内截图（上传到群内新人报道相册）。
 3. 完成游戏自带training(游戏主界面右上角ops->training下所有项目)。
 4.推荐关注北京蓝军公众号：ingressbeijing。每天都有关于ingress有趣新闻推送。
5.群文件有各种科学上网工具，仍有困难可以咨询群里老司机。
外地agent来访推荐做南大、东大拼图任务。upc以及其他任务攻略，可以咨询群里老司机。`
	} else if groupNo == 292243472 {
		newbeMission = "新手五项任务：\n  1. 修改群名片（群名片格式：游戏id-活动区域）。\n  2. 上传带游戏id 的游戏内截图（上传到群内新人报道相册）。\n  3. 完成游戏自带training(游戏主界面右上角ops->training下所有项目)。\n  .阅读ingress 新手指南: http://mp.weixin.qq.com/s?__biz=MzIxNTI4ODU1OA==&mid=403604670&idx=1&sn=1b74a16225deebefe9fcb81e09a39477&scene=18\n欢迎提其它问题"
	} else if groupNo == 312848770 {
		newbeMission = fmt.Sprintf(`[CQ:at,qq=%d], 欢迎加入苏州抵抗军！
教程：
1、【重要】过马路不要玩手机！！！！！
2、基础概念 ( http://t.cn/R5drRQF )
3、升级指南 ( http://t.cn/R5drRQe )
4、进阶数据 ( http://t.cn/R5drRQD )
5、视频教程 ( http://i.youku.com/tomasish )
传教：
1、 Agents ( http://t.cn/R5drRQg )
上海蓝军微信公众号：sh_res
The world around you is not what it seems.`, QQNum)
	}
	if subtype == 1 {
		message = fmt.Sprintf("欢迎新人 [CQ:at,qq=%d]!\n建议玩家使用英文界面方便交流(不要吐槽英文界面哪里方便交流...)\n先右上角目录→设备→语言→english即可\n请务必完成\n%s", QQNum, newbeMission)
	} else if subtype == 2 {
		message = fmt.Sprintf("欢迎 [CQ:at,qq=%d] 邀请的新人 [CQ:at,qq=%d]!建议玩家使用英文界面方便交流(不要吐槽英文界面哪里方便交流...)\n先右上角目录→设备→语言→english即可", operateQQ, QQNum)
	}
	return
}
