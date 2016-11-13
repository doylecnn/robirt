package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	"net"

	"os"

	"strings"
	//"time"

	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"
)

var (
	db              *sql.DB
	config          tomlConfig
	client_conn     *net.TCPConn
	close_sign_chan chan struct{}     = make(chan struct{})
	request_chan    chan Notification = make(chan Notification, 1)
)

func init() {

}

func main() {
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		log.Println(err)
		return
	}
	var err error
	db, err = sql.Open("postgres", config.Database.DBName)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	go eventLoop()
	go func() {
		l_addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:7008")
		if err != nil {
			log.Fatalf("ResolveTCPAddr Error: %v\n", err)
		}
		ln, err := net.ListenTCP("tcp4", l_addr)
		if err != nil {
			log.Fatal("Failed to listening server: %v", err)
		}
		log.Println("Listening server on tcp:127.0.0.1:7008")

		var groups_loaded = false
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("Accept Error: %v\n", err)
				continue
			}
			go handleRequest(conn)
			if !groups_loaded {
				GetToken()
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cmd(scanner.Text())
		}
	}()
	<-close_sign_chan
}

func cmd(cmd string) {
	if cmd == "exit" {
		close(close_sign_chan)
	}
	s := strings.SplitN(strings.TrimSpace(cmd), ":", 2)
	if len(s) == 2 {
		groupnum, err := strconv.ParseInt(strings.TrimSpace(s[0]), 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		var group Group
		var find = false
		groups.RWLocker.RLock()
		for _, g := range groups.Map {
			if g.GroupNo == groupnum {
				group = g
				find = true
				break
			}
		}
		groups.RWLocker.RUnlock()
		if find {
			sendGroupMessage(group.GroupNo, s[1])
		} else {
			fmt.Println("group not found!")
		}
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	for {
		// Make a buffer to hold incoming data.
		buf := make([]byte, 4096)
		// Read the incoming connection into the buffer.
		reqLen, err := conn.Read(buf)
		if err != nil {
			log.Println("Error reading:", err.Error())
			return
		}
		r := bytes.NewReader(buf[:reqLen])
		b, _ := ioutil.ReadAll(r)
		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			b := bytes.TrimSpace(scanner.Bytes())
			if len(b) == 0 {
				log.Println("b length=0",string(b))
				continue
			}
			var js Notification
			err = json.Unmarshal(b, &js)
			if err != nil {
				log.Println(err)
				fmt.Println(string(b))
				continue
			}
			request_chan <- js
		}
	}
}

func eventLoop() {
	for {
		js := <-request_chan

		if js.Method == "Token" {
			go eventToken(js.Params)
			continue
		}

		subtype, _ := js.Params.GetInt64("subtype")
		//sendTime, _ := js.Params.GetTime("sendtime")
		// span := time.Since(sendTime).Seconds()
		// if span > 30 || span <0 {
		// 	continue
		// }

		switch js.Method {
		case "GroupMessage":
			go event_group_message(js.Params)
		case "DiscussMessage":
			go event_discuss_message(js.Params)
		case "PrivateMessage":
			go event_private_message(js.Params)
		case "GroupMemberJoin":
			qqnum, _ := js.Params.GetInt64("qqnum")
			groupnum, _ := js.Params.GetInt64("groupnum")
			beingOperateQQ, _ := js.Params.GetInt64("opqqnum")
			message := welcomeNewMember(subtype, groupnum, qqnum, beingOperateQQ)
			sendGroupMessage(groupnum, message)
			GetToken()
		case "GroupMemberLeave":
			qqnum, _ := js.Params.GetInt64("qqnum")
			groupnum, _ := js.Params.GetInt64("groupnum")
			if groupnum == 196656732 {
				continue
			}
			//beingOperateQQ := js.Params.GetInt64("opqqnum")
			groups.RWLocker.RLock()
			group:=groups.Map[groupnum]
			groups.RWLocker.RUnlock()
			members := group.Members
			members.RWLocker.RLock()
			if subtype == 1 {
				nickname := members.Map[qqnum].Nickname
				message := fmt.Sprintf("群员:[%s] 退群了!!!", nickname)
				sendGroupMessage(groupnum, message)
			} else if subtype == 2 {
				nickname := members.Map[qqnum].Nickname
				message := fmt.Sprintf("群员:[%s] 被 某个管理员 踢了!!!", nickname)
				sendGroupMessage(groupnum, message)
			}
			members.RWLocker.RUnlock()
			GetToken()
		case "RequestAddFriend":
			responseFlag, _ := js.Params.GetString("response_flag")
			addFriend(responseFlag, 1, "")
		case "RequestAddGroup":
			responseFlag, _ := js.Params.GetString("response_flag")
			if subtype == 2 {
				addGroup(responseFlag, 2, 1, "")
			}
			GetToken()
		default:
			log.Printf("未处理：%s\n", js)
		}
	}
}

func welcomeNewMember(subtype, groupNo, QQNo, operateQQ int64) (message string) {
	var newbe_mission string
	if groupNo == 171712942 {
		newbe_mission = "新手四项任务：\n  1.修改群名片（群名片格式：游戏id + 活动区域）\n  2.上传带游戏id 的游戏内截图（上传到群内新人报道相册）\n  3.完成游戏自带training(游戏主界面右上角ops->training下所有项目)\n  4.阅读 ingress 新手指南: http://mp.weixin.qq.com/s?__biz=MzIxNTI4ODU1OA==&mid=403604670&idx=1&sn=1b74a16225deebefe9fcb81e09a39477&scene=18\n  5.不要轻易相信群里自称托马西的\n欢迎提其它问题"
	} else if groupNo == 147798016 {
		newbe_mission = "新手五项任务：\n  1.修改群名片（群名片格式：游戏等级-游戏id-活动区域）\n  2.上传带游戏id 的游戏内截图（上传到群内新人报道相册）\n  3.完成游戏自带training(游戏主界面右上角ops->training下所有项目)\n  4.阅读ingress 新手指南: http://mp.weixin.qq.com/s?__biz=MzIxNTI4ODU1OA==&mid=403604670&idx=1&sn=1b74a16225deebefe9fcb81e09a39477&scene=18\n  5.汉子请爆照, 妹子自动免除此条\n欢迎提其它问题"
	} else if groupNo == 196656732 {
		newbe_mission = `欢迎加入南京蓝色抵抗军大家庭^_^
建议新人按顺序完成以下事宜：
 1. 修改群名片（格式：游戏id-活动区域）。
 2. 上传带游戏id 的游戏内截图（上传到群内新人报道相册）。
 3. 完成游戏自带training(游戏主界面右上角ops->training下所有项目)。
 4.推荐关注北京蓝军公众号：ingressbeijing。每天都有关于ingress有趣新闻推送。
5.群文件有各种科学上网工具，仍有困难可以咨询群里老司机。
外地agent来访推荐做南大、东大拼图任务。upc以及其他任务攻略，可以咨询群里老司机。`
	} else if groupNo == 292243472 {
		newbe_mission = "新手五项任务：\n  1. 修改群名片（群名片格式：游戏id-活动区域）。\n  2. 上传带游戏id 的游戏内截图（上传到群内新人报道相册）。\n  3. 完成游戏自带training(游戏主界面右上角ops->training下所有项目)。\n  .阅读ingress 新手指南: http://mp.weixin.qq.com/s?__biz=MzIxNTI4ODU1OA==&mid=403604670&idx=1&sn=1b74a16225deebefe9fcb81e09a39477&scene=18\n欢迎提其它问题"
	}else if groupNo == 312848770 {
		newbe_mission = fmt.Sprintf(`[CQ:at,qq=%d], 欢迎加入苏州抵抗军！
教程：
1、【重要】过马路不要玩手机！！！！！
2、基础概念 ( http://t.cn/R5drRQF )
3、升级指南 ( http://t.cn/R5drRQe )
4、进阶数据 ( http://t.cn/R5drRQD )
5、视频教程 ( http://i.youku.com/tomasish )
传教：
1、 Agents ( http://t.cn/R5drRQg )
上海蓝军微信公众号：sh_res
The world around you is not what it seems.`, QQNo)
	}
	if subtype == 1 {
		message = fmt.Sprintf("欢迎新人 [CQ:at,qq=%d]!\n建议玩家使用英文界面方便交流(不要吐槽英文界面哪里方便交流...)\n先右上角目录→设备→语言→english即可\n请务必完成\n%s", QQNo, newbe_mission)
	} else if subtype == 2 {
		message = fmt.Sprintf("欢迎 [CQ:at,qq=%d] 邀请的新人 [CQ:at,qq=%d]!建议玩家使用英文界面方便交流(不要吐槽英文界面哪里方便交流...)\n先右上角目录→设备→语言→english即可", operateQQ, QQNo)
	}
	return
}
