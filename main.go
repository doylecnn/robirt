package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

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
)

func main() {
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

		var groupsLoaded = false
		for {
			conn, err := ln.Accept()
			if err != nil {
				logger.Printf("Accept Error: %v\n", err)
				continue
			}
			go handleRequest(conn)
			if !groupsLoaded {
				getToken()
			}
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
	}
	s := strings.SplitN(strings.TrimSpace(cmd), ":", 2)
	if len(s) == 2 {
		groupNum, err := strconv.ParseInt(strings.TrimSpace(s[0]), 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}

		if g, ok := groups.getGroup(groupNum); ok {
			sendGroupMessage(g.GroupNum, s[1])
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
	for {
		js := <-requestChan

		if js.Method == "Token" {
			getTokenHandle(js.Params)
			go eventLoop()
			break
		}
	}
}

func eventLoop() {
	for {
		js := <-requestChan

		if js.Method == "Token" {
			go getTokenHandle(js.Params)
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
			getToken()
		case "GroupMemberLeave":
			qqNum, _ := js.Params.getInt64("qqnum")
			groupNum, _ := js.Params.getInt64("groupnum")
			if groupNum == 196656732 {
				continue
			}
			//beingOperateQQ := js.Params.GetInt64("opqqnum")
			if group, ok := groups.getGroup(groupNum); ok {
				members := group.Members
				if member, ok := members.getMember(qqNum); ok {
					if subtype == 1 {
						message := fmt.Sprintf("群员:[%s] 退群了!!!", member.Nickname)
						sendGroupMessage(groupNum, message)
					} else if subtype == 2 {
						message := fmt.Sprintf("群员:[%s] 被 某个管理员 踢了!!!", member.Nickname)
						sendGroupMessage(groupNum, message)
					}
				}
			}
			getToken()
		case "RequestAddFriend":
			responseFlag, _ := js.Params.getString("response_flag")
			addFriend(responseFlag, 1, "")
		// case "RequestAddGroup":
		// 	responseFlag, _ := js.Params.getString("response_flag")
		// 	if subtype == 2 {
		// 		addGroup(responseFlag, 2, 1, "")
		// 	}
		// 	getToken()
		default:
			logger.Printf("未处理：%s\n", js)
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
