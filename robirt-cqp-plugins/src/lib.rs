#![feature(custom_derive, plugin)]
extern crate serde;
extern crate serde_json;

extern crate encoding;
extern crate libc;

#[macro_use]
extern crate lazy_static;

#[macro_use]
extern crate cqpsdk;

use serde_json::Value;

use std::ffi::CString;
use std::ffi::CStr;
use std::str;

use std::sync::Mutex;
use std::thread;
use std::io::prelude::*;
use std::net::{TcpStream, TcpListener, Shutdown};

use encoding::{Encoding, EncoderTrap, DecoderTrap};
use encoding::all::{GBK};

use libc::c_char;

use cqpsdk::cqpapi;

struct PUPURIUM {
    is_initialized: bool,
    auth_code: i32
}

static mut PUPURIUM: PUPURIUM = PUPURIUM {
    is_initialized: false,
    auth_code: 0
};

lazy_static! {
    static ref TCP_CLIENT: Mutex<TcpStream> = Mutex::new(TcpStream::connect("127.0.0.1:7008").unwrap());
}

#[export_name="\x01_AppInfo"]
pub extern "stdcall" fn app_info() -> *const c_char {
    return CString::new("9,me.robirt.rust.welcome").unwrap().into_raw();
}

#[export_name="\x01_Initialize"]
pub extern "stdcall" fn initialize(auth_code: i32) -> i32 {
    unsafe {
        PUPURIUM.auth_code = auth_code;
        PUPURIUM.is_initialized = true;
    }
    return 0;
}

// Type=1001 酷Q启动
// 无论本应用是否被启用，本函数都会在酷Q启动后执行一次，请在这里执行应用初始化代码。
// 如非必要，不建议在这里加载窗口。（可以添加菜单，让用户手动打开窗口）
#[export_name="\x01_CQPStartupHandler"]
pub extern "stdcall" fn cqp_startup_handler()->i32{
    return 0;
}

// Type=1002 酷Q退出
// 无论本应用是否被启用，本函数都会在酷Q退出前执行一次，请在这里执行插件关闭代码。
// 本函数调用完毕后，酷Q将很快关闭，请不要再通过线程等方式执行其他代码。
#[export_name="\x01_CQPExitHandler"]
pub extern "stdcall" fn cqp_exit_handler()->i32{
    return 0;
}

// Type=1003 应用已被启用
// 当应用被启用后，将收到此事件。
// 如果酷Q载入时应用已被启用，则在_eventStartup(Type=1001,酷Q启动)被调用后，本函数也将被调用一次。
// 如非必要，不建议在这里加载窗口。（可以添加菜单，让用户手动打开窗口）
#[export_name="\x01_EnableHandler"]
pub extern "stdcall" fn cqp_enable_handler()->i32{
    thread::spawn(||{
        match TcpListener::bind("127.0.0.1:7000"){
            Ok(listener) =>{
                unsafe{
                    cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_DEBUG,CString::new("rust json rpc").unwrap().as_ptr(),CString::new("listening started, ready to accept").unwrap().as_ptr())
                };
                for stream in listener.incoming() {
                    match stream {
                        Ok(stream) => {
                            // unsafe{
                            //     cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_INFO,CString::new("rust json rpc").unwrap().as_ptr(),gbk!("收到接入"))
                            // };
                            thread::spawn(move|| {
                                handle_client(stream);
                            });
                        }
                        Err(e) => {
                            let error_msg = format!("{:?}", e);
                            unsafe{
                                cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_ERROR,CString::new("rust json rpc").unwrap().as_ptr(),gbk!(error_msg.as_str()))
                            };
                        }
                    }
                }
                drop(listener);
            }
            Err(e) => {
                let error_msg = format!("{:?}", e);
                unsafe{
                    cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_ERROR,CString::new("rust json rpc").unwrap().as_ptr(),gbk!(error_msg.as_str()))
                };
            }
        }
    });
    return 0;
}

// Type=1004 应用将被停用
// 当应用被停用前，将收到此事件。
// 如果酷Q载入时应用已被停用，则本函数*不会*被调用。
// 无论本应用是否被启用，酷Q关闭前本函数都*不会*被调用。
#[export_name="\x01_DisableHandler"]
pub extern "stdcall" fn cqp_disable_handler()->i32{
    return 0;
}

// Type=21 私聊消息
// subType:11:来自好友 1:来自在线状态 2:来自群 3:来自讨论组
#[export_name="\x01_PrivateMessageHandler"]
pub extern "stdcall" fn private_message_handler(sub_type: i32, send_time: i32, qq_num: i64, msg: *const c_char, font: i32) -> i32 {
    let msg = unsafe{
        json_trans(utf8!(msg).to_owned())
    };

    let notification = format!(r#"{{"method":"PrivateMessage","params":{{"subtype":{},"sendtime":{},"qqnum":{},"message":"{}","font":{}}}}}"#, sub_type, send_time, qq_num, msg, font);
    send_notification(notification);
    return cqpapi::EVENT_IGNORE;
}

// Type=2 群消息  subType固定为1
// fromQQ == 80000000 && strlen(fromAnonymous)>0 为 匿名消息
#[export_name="\x01_GroupMessageHandler"]
pub extern "stdcall" fn group_message_handler(sub_type: i32, send_time: i32, group_num: i64, qq_num: i64, anonymous_name: *const c_char, msg: *const c_char, font: i32) -> i32 {
    let msg = unsafe{
        json_trans(utf8!(msg).to_owned())
    };
    let anonymous_name = unsafe{
        json_trans(utf8!(anonymous_name).to_owned())
    };
    let notification = format!(r#"{{"method":"GroupMessage","params":{{"subtype":{},"sendtime":{},"groupnum":{},"qqnum":{},"anonymousname":"{}","message":"{}","font":{}}}}}"#,sub_type,send_time,group_num,qq_num, anonymous_name, msg, font);
    send_notification(notification);
    return cqpapi::EVENT_IGNORE;
}

// Type=102 群事件-群成员减少
// subType 1/群员离开 2/群员被踢 3/自己(即登录号)被踢
// fromQQ, 操作者QQ(仅子类型为2、3时存在)
#[export_name="\x01_GroupMemberLeaveHandler"]
pub extern "stdcall" fn group_member_leave_handler(sub_type: i32, send_time: i32, group_num: i64, opqq_num: i64, qq_num: i64) -> i32 {
    let notification = format!(r#"{{"method":"GroupMemberLeave","params":{{"subtype":{},"sendtime":{},"groupnum":{},"opqqnum":{},"qqnum":{}}}}}"#, sub_type, send_time, group_num, opqq_num, qq_num);
    send_notification(notification);
    return cqpapi::EVENT_IGNORE;
}

// Type=103 群事件-群成员增加
// subType 1/管理员已同意 2/管理员邀请
#[export_name="\x01_GroupMemberJoinHandler"]
pub extern "stdcall" fn group_member_join_handler(sub_type: i32, send_time: i32, group_num: i64, opqq_num: i64, qq_num: i64) -> i32 {
    let notification = format!(r#"{{"method":"GroupMemberJoin","params":{{"subtype":{},"sendtime":{},"groupnum":{},"opqqnum":{},"qqnum":{}}}}}"#, sub_type, send_time, group_num, opqq_num, qq_num);
    send_notification(notification);
    return cqpapi::EVENT_IGNORE;
}

// Type=301 请求-好友添加
#[export_name="\x01_RequestAddFriendHandler"]
pub extern "stdcall" fn request_add_friend_handler(sub_type: i32, send_time: i32, from_qq: i64, msg: *const c_char, response_flag: *const c_char) -> i32 {
    let msg = unsafe{
        json_trans(utf8!(msg).to_owned())
    };
    let response_flag = unsafe{
        json_trans(utf8!(response_flag).to_owned())
    };
    let notification = format!(r#"{{"method":"RequestAddFriend","params":{{"subtype":{},"sendtime":{},"fromqq":{},"msg":"{}","response_flag":"{}"}}}}"#, sub_type, send_time, from_qq, msg, response_flag);
    send_notification(notification);
    return cqpapi::EVENT_IGNORE;
}

// Type=302 请求-群添加
#[export_name="\x01_RequestAddGroupHandler"]
pub extern "stdcall" fn request_add_group_handler(sub_type: i32, send_time: i32, group_num: i64, from_qq: i64, msg: *const c_char, response_flag: *const c_char) -> i32 {
    let msg = unsafe{
        json_trans(utf8!(msg).to_owned())
    };
    let response_flag = unsafe{
        json_trans(utf8!(response_flag).to_owned())
    };
    let notification = format!(r#"{{"method":"RequestAddGroup","params":{{"subtype":{},"sendtime":{},"groupnum":{},"fromqq":{},"msg":"{}","response_flag":"{}"}}}}"#, sub_type, send_time, group_num, from_qq, msg, response_flag);
    send_notification(notification);
    return cqpapi::EVENT_IGNORE;
}

// Type=4 讨论组消息处理
#[export_name="\x01_DiscussMessageHandler"]
pub extern "stdcall" fn discuss_message_handler(sub_type: i32, send_time: i32, from_discuss: i64, qq_num: i64, msg: *const c_char, font: i32) -> i32 {
    let msg = unsafe{
        json_trans(utf8!(msg).to_owned())
    };
    let notification = format!(r#"{{"method":"DiscussMessage","params":{{"subtype":{},"sendtime":{},"fromdiscuss":{},"fromqq":{},"msg":"{}","font":"{}"}}}}"#, sub_type, send_time, from_discuss, qq_num, msg, font);
    send_notification(notification);
    return cqpapi::EVENT_IGNORE;
}

//
// ========== 分割线 ==========
//

fn json_trans(json :String)->String{
    let json = json.replace("\\","\\\\");
    let json = json.replace("\"","\\\"");
    let json = json.replace("\n","\\n");
    let json = json.replace("\r","\\r");
    json.replace("\t","\\t")
}

fn send_notification(notification: String){
    let notification = format!("{}\n",notification);
    match TCP_CLIENT.lock(){
        Ok(mut client)=>{
            match client.write_all(notification.as_bytes()){
                Ok(_)=>{
                    // unsafe{
                    //     cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_INFO,CString::new("rust json rpc").unwrap().as_ptr(),gbk!(notification.as_str()))
                    // };
                }
                Err(e)=>{
                    let error_msg = format!("{:?}", e);
                    unsafe{
                        cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_ERROR,CString::new("rust json rpc").unwrap().as_ptr(),gbk!(error_msg.as_str()))
                    };
                }
            }
        }
        Err(e)=>{
            let error_msg = format!("{:?}", e);
            unsafe{
                cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_ERROR,CString::new("rust json rpc").unwrap().as_ptr(),gbk!(error_msg.as_str()))
            };
        }
    }
}

fn handle_client(mut stream :TcpStream){
    // unsafe{
    //     cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_INFO,CString::new("rust json rpc").unwrap().as_ptr(),gbk!("进入handle_client..."));
    // };
    let mut request = String::new();
    let result = stream.read_to_string(&mut request);
    match result {
        Ok(_) => {
            unsafe{
                cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_DEBUG,CString::new("rust json rpc").unwrap().as_ptr(),gbk!(request.as_str()));
            };
            let json_value: Value = serde_json::from_str(request.as_str()).unwrap();
            let notification = json_value.as_object().unwrap();
            let method = notification.get("method").unwrap().as_str().unwrap();
            // unsafe{
            //     cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_DEBUG,CString::new("rust json rpc").unwrap().as_ptr(),gbk!(method))
            // };
            let params = notification.get("params").unwrap().as_object().unwrap();
            match method{
                "SendPrivateMessage" => {
                    let message = params.get("message").unwrap().as_str().unwrap();
                    let qqnum = params.get("qqnum").unwrap().as_i64().unwrap();
                    unsafe{
                        cqpapi::CQ_sendPrivateMsg(PUPURIUM.auth_code, qqnum, gbk!(message));
                    };
                }
                "SendGroupMessage" => {
                    let message = params.get("message").unwrap().as_str().unwrap();
                    let groupnum = params.get("groupnum").unwrap().as_i64().unwrap();
                    unsafe{
                        cqpapi::CQ_sendGroupMsg(PUPURIUM.auth_code, groupnum, gbk!(message));
                    };
                }
                "SendDiscussionMessage" => {
                   let message = params.get("message").unwrap().as_str().unwrap();
                   let discussion_num = params.get("discussionnum").unwrap().as_i64().unwrap();
                   unsafe{
                       cqpapi::CQ_sendDiscussMsg(PUPURIUM.auth_code, discussion_num, gbk!(message));
                   };
                }
                "GetToken" => {
                    let csrf_token = unsafe{
                        cqpapi::CQ_getCsrfToken(PUPURIUM.auth_code)
                    };
                    let login_qq = unsafe{
                        cqpapi::CQ_getLoginQQ(PUPURIUM.auth_code)
                    };
                    let cookies = unsafe{
                        let bytes = CStr::from_ptr(cqpapi::CQ_getCookies(PUPURIUM.auth_code)).to_bytes();
                        str::from_utf8(bytes).unwrap()
                    };
                    let cookies = json_trans(cookies.to_owned());
                    let notification = format!(r#"{{"method":"Token","params":{{"token":{},"cookies":"{}","loginqq":{}}}}}"#, csrf_token, cookies, login_qq);
                    send_notification(notification);
                }
                "FriendAdd" => {
                    let response_flag = params.get("responseFlag").unwrap().as_str().unwrap();
                    let accept = params.get("accept").unwrap().as_i64().unwrap() as i32;
                    let memo = params.get("memo").unwrap().as_str().unwrap();
                    unsafe{
                        cqpapi::CQ_setFriendAddRequest(PUPURIUM.auth_code, gbk!(response_flag), accept, gbk!(memo));
                    };
                }
                "GroupAdd" => {
                    let response_flag = params.get("responseFlag").unwrap().as_str().unwrap();
                    let accept = params.get("accept").unwrap().as_i64().unwrap() as i32;
                    let sub_type = params.get("subType").unwrap().as_i64().unwrap() as i32;
                    let reason = params.get("reason").unwrap().as_str().unwrap();
                    unsafe{
                        cqpapi::CQ_setGroupAddRequestV2(PUPURIUM.auth_code, gbk!(response_flag), sub_type, accept, gbk!(reason));
                    };
                }
                "GroupLeave" => {
                    let groupnum = params.get("groupnum").unwrap().as_i64().unwrap();
                    let qqnum = params.get("qqnum").unwrap().as_i64().unwrap();
                    unsafe{
                        cqpapi::CQ_setGroupLeave(PUPURIUM.auth_code, groupnum, qqnum, 0);
                    };
                }
                "GroupBan" => {
                    let groupnum = params.get("groupnum").unwrap().as_i64().unwrap();
                    let qqnum = params.get("qqnum").unwrap().as_i64().unwrap();
                    let seconds = params.get("seconds").unwrap().as_i64().unwrap();
                    unsafe{
                        cqpapi::CQ_setGroupBan(PUPURIUM.auth_code, groupnum, qqnum, seconds);
                    };
                }
                // "GetGroupMemberInfo"=>{//Auth=130 //getGroupMemberInfoV2
                //     let groupnum = params.get("groupnum").unwrap().as_i64().unwrap();
                //     let qqnum = params.get("qqnum").unwrap().as_i64().unwrap();
                //     unsafe{
                //         cqpapi::CQ_getGroupMemberInfoV2(PUPURIUM.auth_code, groupnum, qqnum, 0);
                //     };
                //     let notification = format!(r#"{{"method":"GroupMemberInfo","params":{{"token":{},"cookies":"{}","loginqq":{}}}}}"#, csrf_token, cookies, login_qq);
                //     send_notification(notification);
                // }
                _ =>{
                    unsafe{
                        cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_ERROR,CString::new("rust json rpc default").unwrap().as_ptr(),gbk!(request.as_str()));
                    };
                }
            }
        }
        Err(e)=>{
            let error_msg = format!("{:?}", e);
            unsafe{
                cqpapi::CQ_addLog(PUPURIUM.auth_code,cqpapi::CQLOG_ERROR,CString::new("rust json rpc error").unwrap().as_ptr(),gbk!(error_msg.as_str()))
            };
        }
    }
    let _ = stream.shutdown(Shutdown::Both);
}