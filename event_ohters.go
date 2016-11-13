package main

import (
	"fmt"
	"log"
)

var (
	LoginQQ, Csrf_token int64
	Cookies             string
)

func eventToken(p Params) {
	Csrf_token, _ = p.GetInt64("token")
	LoginQQ, _ = p.GetInt64("loginqq")
	Cookies, _ = p.GetString("cookies")
	fmt.Printf("LoginQQ:%d, Csrf_token:%d, Cookies:%s\n", LoginQQ, Csrf_token, Cookies)
	groups = GetGroups(LoginQQ, Cookies, Csrf_token)
	log.Println("refresh groups info")
}
