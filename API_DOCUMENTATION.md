# API Documentation - نوشیروانی آکادمی

مستندات کامل API برای سیستم مدیریت آکادمی نوشیروانی به صورت Swagger/OpenAPI است.

## 🚀 دسترسی به مستندات

پس از اجرای سرور، می‌توانید مستندات Swagger را در آدرس‌های زیر مشاهده کنید:

### توسعه محلی
```
http://localhost:8080/swagger/index.html
```

### Docker
```
http://localhost:8080/swagger/index.html
```

نکته: خود سرویس Go مسیرها را بدون prefix ارائه می‌کند، مثل `/auth/request-otp`. در Docker/Nginx همان endpointها از بیرون با prefix `/api/v1/` منتشر می‌شوند، مثل `/api/v1/auth/request-otp`.

## 📖 بخش‌های API

### 1. احراز هویت (Authentication)
- **درخواست OTP**: `POST /api/v1/auth/request-otp`
- **تأیید OTP و ورود**: `POST /api/v1/auth/verify-otp`
- **تازه‌سازی توکن**: `POST /api/v1/auth/refresh`

### 2. مدیریت پروفایل دانشجو (Student Profile)
- **تکمیل/بروزرسانی پروفایل**: `POST /api/v1/students/profile`
- **دریافت پروفایل**: `GET /api/v1/students/profile`

### 3. مدیریت آزمون‌ها (Exams)
- **ایجاد آزمون**: `POST /api/v1/exams`
- **دریافت لیست آزمون‌ها**: `GET /api/v1/exams`
- **دریافت جزئیات آزمون**: `GET /api/v1/exams/{id}`
- **حذف آزمون**: `DELETE /api/v1/exams/{id}`

### 4. مدیریت اشتباهات (Mistakes)
- **ثبت اشتباه جدید**: `POST /api/v1/mistakes`
- **دریافت لیست اشتباهات**: `GET /api/v1/mistakes`
- **حذف اشتباه**: `DELETE /api/v1/mistakes/{id}`

### 5. مدیریت بلاگ (Blog Management)
- **دریافت مقالات منتشر‌شده**: `GET /api/v1/blog` (بدون احراز هویت)
- **دریافت مقاله با slug**: `GET /api/v1/blog/{slug}` (بدون احراز هویت)
- **ایجاد مقاله جدید**: `POST /api/v1/blog` (فقط مدیران)
- **بروزرسانی مقاله**: `PUT /api/v1/blog/{id}` (فقط مدیران)
- **انتشار مقاله**: `PUT /api/v1/blog/{id}/publish` (فقط مدیران)
- **حذف مقاله**: `DELETE /api/v1/blog/{id}` (فقط مدیران)
- **دریافت تمام مقالات**: `GET /api/v1/blog` (فقط مدیران)

### 6. مدیریت دانشجویان (Admin)
- **دریافت لیست دانشجویان**: `GET /api/v1/admin/students`
- **دریافت جزئیات دانشجو**: `GET /api/v1/admin/students/{id}`
- **بروزرسانی دانشجو**: `PUT /api/v1/admin/students/{id}`
- **تایید دانشجو**: `PUT /api/v1/admin/students/{id}/approve`
- **حذف دانشجو**: `DELETE /api/v1/admin/students/{id}`

### 7. مدیریت فیلدهای سفارشی (Dynamic Fields)
- **دریافت فیلدها**: `GET /api/v1/admin/dynamic-fields`
- **ایجاد فیلد جدید**: `POST /api/v1/admin/dynamic-fields`
- **بروزرسانی فیلد**: `PUT /api/v1/admin/dynamic-fields/{id}`
- **حذف فیلد**: `DELETE /api/v1/admin/dynamic-fields/{id}`

## 🔐 احراز هویت

تمام endpoint‌های محافظت‌شده به توکن JWT احتیاج دارند.

### روش استفاده:
1. ابتدا `POST /api/v1/auth/request-otp` را با شماره تلفن فراخوانی کنید
2. کد OTP را دریافت کنید
3. `POST /api/v1/auth/verify-otp` را با شماره و کد فراخوانی کنید
4. توکن‌های `access_token` و `refresh_token` را دریافت کنید
5. توکن را در Header به صورت زیر ارسال کنید:
```
Authorization: Bearer {access_token}
```

### تازه‌سازی توکن:
وقتی توکن منقضی شود، می‌توانید از `refresh_token` برای دریافت توکن جدید استفاده کنید:
```
POST /api/v1/auth/refresh
{
  "refresh_token": "your_refresh_token"
}
```

## 📝 نمونه درخواست‌ها

### 1. درخواست OTP
```bash
curl -X POST http://localhost:8080/api/v1/auth/request-otp \
  -H "Content-Type: application/json" \
  -d '{"phone": "+989123456789"}'
```

در محیط توسعه با `OTP_PROVIDER=mock` و `EXPOSE_MOCK_OTP=true`، کد OTP داخل پاسخ برمی گردد. این رفتار فقط برای توسعه/تست است. در `ENVIRONMENT=production` استفاده از `mock` یا فعال بودن `EXPOSE_MOCK_OTP` مجاز نیست.

### 2. تأیید OTP
```bash
curl -X POST http://localhost:8080/api/v1/auth/verify-otp \
  -H "Content-Type: application/json" \
  -d '{"phone": "+989123456789", "code": "123456"}'
```

### 3. ایجاد آزمون
```bash
curl -X POST http://localhost:8080/api/v1/exams \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {access_token}" \
  -d '{
    "title": "ریاضی نوبت اول",
    "jalali_date": "1400/01/15",
    "major": "ریاضی",
    "negative_mark": 0.25,
    "total_subjects": 3,
    "subjects": [
      {
        "subject_name": "جبر",
        "total_questions": 20,
        "answered": 18,
        "correct": 16
      }
    ]
  }'
```

`exam_date` و `jalali_date` را همزمان ارسال نکنید؛ فقط یکی از آن‌ها باید در ورودی باشد.

### 4. دریافت پروفایل
```bash
curl -X GET http://localhost:8080/api/v1/students/profile \
  -H "Authorization: Bearer {access_token}"
```

## 🐳 اجرا با Docker

سرور API خودکار با Docker Compose اجرا می‌شود:

```bash
docker-compose up backend
```

مستندات در `http://localhost:8080/swagger/index.html` قابل دسترس خواهد بود.

## 📚 ساختار Response

تمام پاسخ‌های API با ساختار استاندارد ارائه می‌شوند:

### پاسخ موفق:
```json
{
  "id": "uuid",
  "field1": "value1",
  "created_at": "2026-05-08T12:00:00Z"
}
```

### پاسخ خطا:
```json
{
  "error": "Error message"
}
```

## 🔄 کدهای وضعیت HTTP

- `200`: درخواست موفق
- `201`: منبع با موفقیت ایجاد شد
- `400`: درخواست نامعتبر
- `401`: عدم اجازه دسترسی / توکن منقضی
- `403`: عدم صلاحیت (مثلاً دانشجو سعی می‌کند به بخش مدیران دسترسی پیدا کند)
- `404`: منبع یافت نشد
- `500`: خطای سرور

## 🛠️ فیلدهای سفارشی

سیستم امکان ایجاد فیلدهای سفارشی برای دانشجویان و آزمون‌ها را فراهم می‌کند.

### انواع فیلد:
- `text`: متن ساده
- `number`: عدد
- `select`: انتخاب از لیست
- `checkbox`: چک‌باکس
- `date`: تاریخ

## 📋 نکات مهم

1. **تاریخ جلالی**: برای تاریخ‌ها می‌توانید از فرمت جلالی `YYYY/MM/DD` استفاده کنید. اگر فیلد جلالی می‌فرستید، مقدار ذخیره‌شده و خروجی به‌صورت canonical و صفر-پد شده برمی‌گردد.
2. **تاریخ‌های دوگانه**: در endpointهایی که هم فیلد میلادی و هم جلالی دارند، این دو فیلد mutually exclusive هستند و باید فقط یکی ارسال شود.
3. **نمره منفی**: فیلد `negative_mark` مقدار نمره کسر شده برای هر پاسخ غلط است و باید عددی بین `0` تا `1` باشد (پیش‌فرض: `0`).
4. **شماره تلفن**: شماره تلفن باید با `+98` شروع شود
5. **Slug**: برای مقالات بلاگ، slug به‌صورت خودکار از عنوان ایجاد می‌شود
6. **محدودیت میزان درخواست**: سرور از Rate Limiting استفاده می‌کند و headerهای `X-RateLimit-*` و `Retry-After` برمی‌گرداند

## 📞 پشتیبانی

برای سؤالات و مشکلات:
- Email: support@noshirvaniacademy.com
- GitHub: https://github.com/Offline-Devs

---

**نسخه API**: 1.0
**آخرین بروزرسانی**: 2026/05/08
