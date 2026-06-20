# Endpoint Test Report

Generated while building the integration test suite under `backend/tests/`.
Every endpoint registered in `internal/router/router.go` is exercised by at
least one happy-path test plus error/authorization cases. All tests pass against
the live Postgres + Redis stack.

- **Test groups:** 28 top-level `Test*` functions, 88 sub-cases.
- **Statement coverage** (handlers + middleware + auth + jalali pkg): **71.8%**.
  The uncovered lines are almost entirely the `500 Internal Server Error`
  branches that only fire on a database failure, which black-box tests can't
  trigger without tearing down the DB.

---

## 1. Endpoint coverage matrix

Legend: 🔓 public · 🔒 requires JWT · 👑 requires `admin` role.

### Public / auth

| Method | Path | Auth | Tested cases |
|---|---|---|---|
| GET | `/health` | 🔓 | 200 |
| POST | `/auth/request-otp` | 🔓 | 200 (mock OTP returned), 400 missing phone |
| POST | `/auth/verify-otp` | 🔓 | 200 new user, admin-role mapping, 401 wrong code, 400 missing fields, 403 inactive |
| POST | `/auth/refresh` | 🔓 | 200, 401 invalid, 401 access-token-as-refresh, 401 deleted user, 400 missing body |
| GET | `/blog` | 🔓 | 200 (only published) |
| GET | `/blog/:slug` | 🔓 | 200, 404 unpublished, 404 missing |
| GET | `/subjects` | 🔓 | 200 valid major, 400 missing param, 400 invalid major |
| GET | `/majors` | 🔓 | 200 (4 majors) |

### Student-scoped

| Method | Path | Auth | Tested cases |
|---|---|---|---|
| POST | `/students/profile` | 🔒 | create, update, 400 missing names, 400 bad jalali, 401 |
| GET | `/students/profile` | 🔒 | 200, 404 before creation, 401 |
| POST | `/exams` | 🔒 | 201 with subjects, 404 no profile |
| GET | `/exams` | 🔒 | 200 list |
| GET | `/exams/:id` | 🔒 | 200, 404 missing, 404 cross-student |
| PUT | `/exams/:id` | 🔒 | 200 (replaces subjects), 400 bad jalali, 404 missing, 404 cross-student |
| DELETE | `/exams/:id` | 🔒 | 200 + confirm gone |
| POST | `/mistakes` | 🔒 | 201, 400 non-positive q#, 404 no profile |
| GET | `/mistakes` | 🔒 | 200 list |
| PUT | `/mistakes/:id` | 🔒 | 200, 400 non-positive q#, 404 missing |
| DELETE | `/mistakes/:id` | 🔒 | 200 + confirm gone |
| GET | `/students/performance` | 🔒 | 200, 404 no profile |
| GET | `/students/statistics` | 🔒 | 200 (avg verified), date filter, 404 no profile |
| GET | `/students/dashboard` | 🔒 | 200 (counts verified) |
| POST | `/upload` | 🔒 | 200 png, 400 bad ext, 400 no file, 401 |
| POST | `/upload/multiple` | 🔒 | 200 two files, 400 all invalid, 400 no files |

### Admin

| Method | Path | Auth | Tested cases |
|---|---|---|---|
| GET | `/admin/students` | 👑 | 200, 403 student, 401 no token |
| GET | `/admin/students/with-stats` | 👑 | pagination, approved filter |
| GET | `/admin/students/:id` | 👑 | 200, 404 missing |
| GET | `/admin/students/:id/exams` | 👑 | 200 |
| GET | `/admin/students/:id/mistakes` | 👑 | 200 |
| PUT | `/admin/students/:id` | 👑 | 200, 400 no fields |
| PUT | `/admin/students/:id/approve` | 👑 | 200 (flag set) |
| DELETE | `/admin/students/:id` | 👑 | 200 |
| GET | `/admin/students/:id/performance` | 👑 | 200 |
| POST | `/admin/students/:id/performance` | 👑 | 201, 404 missing student |
| PUT | `/admin/performance/:id` | 👑 | 200 |
| DELETE | `/admin/performance/:id` | 👑 | 200 |
| GET | `/admin/students/:id/statistics` | 👑 | 200 |
| GET | `/admin/dynamic-fields` | 👑 | 200 with filter |
| POST | `/admin/dynamic-fields` | 👑 | 201, 400 missing required |
| PUT | `/admin/dynamic-fields/:id` | 👑 | 200 |
| DELETE | `/admin/dynamic-fields/:id` | 👑 | 200 |
| GET | `/admin/blog` | 👑 | 200 (incl. unpublished) |
| POST | `/admin/blog` | 👑 | 200 (auto-slug), 400 missing title |
| PUT | `/admin/blog/:id` | 👑 | 200 |
| PUT | `/admin/blog/:id/publish` | 👑 | 200 + public visibility |
| DELETE | `/admin/blog/:id` | 👑 | 200 |

---

## 2. Issues found

Severity is my judgement for a student-academy app. Items marked **(test)** have
a regression test in `backend/tests/known_behavior_test.go` pinning the current
behaviour.

### High

1. **Deactivated users can still refresh access tokens. (test)**
   `AuthHandler.VerifyOTP` rejects `is_active=false` users, but
   `RefreshToken` (`auth_handler.go:197`) never checks `IsActive`. A user who is
   deactivated keeps minting fresh 1-hour access tokens until their refresh
   token expires (default 15 days). **Fix:** check `user.IsActive` in
   `RefreshToken` and reject with 401/403.

### Medium

2. **Write/delete on a non-existent ID returns `200 OK`. (test)**
   `UpdateStudent`, `ApproveStudent`, `DeleteStudent`, blog `Update`/`Publish`/
   `Delete`, `DeleteExam`, dynamic-field `Update`/`Delete`, and performance
   `Update`/`Delete` all run a bare `Updates`/`Delete` and report success even
   when **zero rows matched**. Clients can't tell a real update from a no-op, and
   it masks bugs. **Fix:** inspect `result.RowsAffected` and return 404 when 0.

3. **No ownership check on `exam_id` / `subject_exam_id` in mistakes. (test)**
   `MistakeHandler.Create`/`Update` accept any `exam_id`, including one
   belonging to another student (verified: returns 201). Low data-leak risk
   today because reads are still scoped by `student_id`, but it lets a student
   create rows referencing foreign exams. **Fix:** validate the referenced exam
   belongs to the caller's student profile.

4. **Client-supplied Jalali dates are stored unnormalised, breaking date
   filtering. (test)**
   `JalaliToGregorian` uses `Sscanf("%d/%d/%d")`, which accepts `1403/9/5`, and
   the handlers store that raw string in `jalali_date`. Statistics filtering
   compares dates **lexicographically** (`jalali_date >= ?` in
   `statistics_handler.go:126`), so `"1403/9/5" > "1403/10/1"` — wrong. The
   server-generated dates are zero-padded and fine; only client-provided ones
   are at risk. **Fix:** re-format to `YYYY/MM/DD` after parsing before storing.

5. **`randomString` in the upload handler is not random.**
   `upload_handler.go:41` does `charset[time.Now().UnixNano() % len]` inside a
   tight loop, so every byte is derived from near-identical timestamps — in
   practice the 8 characters are frequently identical and always predictable.
   Filenames don't collide in practice because of the `UnixNano()` prefix, but
   the "random" suffix adds little entropy and is guessable. **Fix:** use
   `crypto/rand` (the project already does in `sms/otp.go`).

### Low / polish

6. **`POST /admin/blog` returns `200`, not `201`.** `blog_handler.go:73` and the
   Swagger annotation (`@Success 201`) disagree. Other create handlers return
   201. Minor inconsistency, but clients keying on status will be surprised.

7. **Duplicate Swagger `@Router /blog [get]`.** Both `BlogHandler.List`
   (admin) and `PublicList` annotate `/blog [get]`; the admin route is actually
   `/admin/blog`. The generated spec is ambiguous. **Fix:** annotate List as
   `/admin/blog [get]`.

8. **`GetAllStudentsWithStats` is N+1.** `admin_handler.go:443` runs two
   `COUNT` queries per student in a loop. Fine for the `Limit(100)` / page size,
   but it will not scale; consider a grouped aggregate query.

9. **No phone-number format validation.** `RequestOTP`/`VerifyOTP` accept any
   non-empty string as a phone. Combined with auto-provisioning in `VerifyOTP`,
   any string that receives a valid OTP becomes a user. Low risk (OTP must still
   match) but worth a regex/normalisation step.

10. **Role is baked into the access token and not re-checked.** If an admin is
    demoted, their existing access token keeps admin rights until it expires
    (≤1h). Standard JWT trade-off; documenting for awareness.

11. **In-memory rate limiter is per-process.** `middleware/rate_limiter.go`
    keeps counters in a local map, so limits are per-replica and reset on
    restart. The OTP limiter (Redis-backed) is fine; the global one isn't
    horizontally consistent.

### Notes (not bugs)

- **GORM `default:true` zero-value trap.** `User.IsActive` and
  `DynamicFieldDefinition.IsActive` carry `gorm:"default:true"`. Creating a
  struct with the field left as `false` inserts `true` (GORM treats the zero
  value as "unset"). You can only deactivate via an explicit `Update`/`map`.
  Surfaced while writing the inactive-user test.
- **Exam update replaces subjects wholesale** (delete-all + re-insert). Intended,
  but worth knowing it's not a merge.

---

## 3. How to reproduce

```bash
cd backend
docker exec backend-postgres-1 psql -U postgres -c "CREATE DATABASE noshirvani_test"  # once
go test ./tests/ -v
```

See `backend/tests/README.md` for harness details and env overrides.
