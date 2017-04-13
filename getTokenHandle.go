package main

var (
	// LoginQQ : qq number
	LoginQQ int64
	// CsrfToken : CsrfToken
	CsrfToken int64
	// Cookies : cookies
	Cookies string

	groupsLoaded = false
)

func getTokenHandle(p Params) {
	CsrfToken, _ = p.getInt64("token")
	LoginQQ, _ = p.getInt64("loginqq")
	Cookies, _ = p.getString("cookies")
	logger.Printf("LoginQQ:%d, Csrf_token:%d, Cookies:%s\n", LoginQQ, CsrfToken, Cookies)
	if !groupsLoaded {
		logger.Println("refresh groups info")
		groups = getGroups(LoginQQ, Cookies, CsrfToken)
		groupsLoaded = true
		logger.Println("groups info refreshed")
	}
}
