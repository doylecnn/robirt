# robirt
模仿telegram平台的miaowu的机器人，基于cqp

* 你需要购买一个酷q pro 的授权
* 你需要知道cargo 怎么编译rust
** qu rustup.rs 下载并执行 rustup-init.exe 根据提示安装rust
*** 必须32位，建议GNU
** cargo build --release 编译出demo.dll 并改名为me.robirt.rust.welcome.dll
* 然将rust 编译出来的dll 扔到酷q pro 的那个指定的目录下（具体酷Q插件细节请自己访问cqp.cc）
* 然后要知道怎么go build 和 go install
* 哦，忘记了，你还要自己装pgsql
* 顺便，因为酷Q pro 只支持windows，所以以上都是windows

一切都好了之后，先运行go 编译出来的exe，再启动酷Q pro 就好了
