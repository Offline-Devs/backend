package handler

// ErrorResponse پاسخ خطا
type ErrorResponse struct {
	Error string `json:"error" description:"پیام خطا"`
}

// SuccessResponse پاسخ موفق عمومی
type SuccessResponse struct {
	Message string      `json:"message" description:"پیام موفقیت"`
	Data    interface{} `json:"data,omitempty" description:"داده‌های پاسخ"`
}

// ListResponse پاسخ لیستی
type ListResponse struct {
	Data  interface{} `json:"data" description:"لیست داده‌ها"`
	Total int64       `json:"total" description:"تعداد کل نتایج"`
	Page  int         `json:"page,omitempty" description:"شماره صفحه"`
	Limit int         `json:"limit,omitempty" description:"تعداد نتایج در هر صفحه"`
}

// PaginationParams پارامترهای صفحه‌بندی
type PaginationParams struct {
	Page  int `json:"page" form:"page" example:"1" description:"شماره صفحه"`
	Limit int `json:"limit" form:"limit" example:"20" description:"تعداد نتایج در هر صفحه"`
}
