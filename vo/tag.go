package vo

type EditTagVo struct {
	Name      string `form:"name,omitempty"`
	Desc      string `form:"desc,omitempty"`
	ID        uint   `form:"id,omitempty"`
	ParentID  *uint  `form:"parentID,omitempty"`
	ShowInAll string `form:"showInAll,omitempty"`
	ShowInHot string `form:"showInHot,omitempty"`
	OpenShow  string `form:"openShow,omitempty"`
	CssClass  string `form:"cssClass,omitempty"`
}
