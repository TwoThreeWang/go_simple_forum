package vo

type LoginRequest struct {
	Username      string `form:"username,omitempty"`
	Password      string `form:"password,omitempty"`
	CaptchaAnswer int    `form:"captcha_answer,omitempty"`
	CaptchaID     string `form:"captcha_id,omitempty"`
}
type RegisterRequest struct {
	Username       string `form:"username,omitempty"`
	Password       string `form:"password,omitempty"`
	RepeatPassword string `form:"repeatPassword,omitempty"`
	Email          string `form:"email,omitempty"`
	Bio            string `form:"bio,omitempty"`
	CaptchaAnswer  int    `form:"captcha_answer,omitempty"`
	CaptchaID      string `form:"captcha_id,omitempty"`
}

type EditUserRequest struct {
	Uid      uint   `form:"uid,omitempty"`
	Username string `form:"username,omitempty"`
	Password string `form:"password,omitempty"`
	Email    string `form:"email,omitempty"`
	Bio      string `form:"bio,omitempty"`
	Avatar   string `form:"avatar,omitempty"`
}

type Userinfo struct {
	Username string
	Role     string
	ID       uint
	Email    string
	Avatar   string
}

type ResetPwd struct {
	Email    string `form:"email,omitempty"`
	Password string `form:"password,omitempty"`
	Key      string `form:"key,omitempty"`
}
