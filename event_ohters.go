package main

var (
	LoginQQ, Csrf_token int64
	Cookies             string

	getGroups bool = false
)

func eventToken(p Params) {
	Csrf_token, _ = p.GetInt64("token")
	LoginQQ, _ = p.GetInt64("loginqq")
	Cookies, _ = p.GetString("cookies")
	logger.Printf("LoginQQ:%d, Csrf_token:%d, Cookies:%s\n", LoginQQ, Csrf_token, Cookies)
	if !getGroups{
		groups = GetGroups(LoginQQ, Cookies, Csrf_token)
		getGroups = true
	}
	logger.Println("refresh groups info")
}
