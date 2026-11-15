#!/usr/bin/env bash
# QC GATE-PG TC-GATE-30 — gom ỨNG VIÊN để đọc diff internal/auth và internal/httpapi (PG.md "Bạn tự kiểm" mục 6).
# Chỉ chạy ở phase 2 (QC được đọc mã dev khi chấm). Script KHÔNG kết luận: mỗi ứng viên QC mở ra đọc rồi ghi BUG-n hoặc "chấp nhận" + lý do.
#   bash docs/sprints/2/qc/scripts/diff-review.sh        (BASE=<sha> để đổi mốc; mặc định merge-base với origin/main)
source "$(dirname "$0")/lib.sh"
BASE=${BASE:-$(git merge-base HEAD origin/main)}
P="backend-go/internal/auth backend-go/internal/httpapi"
echo "# diff $BASE..HEAD — $(git rev-parse --short HEAD)"
git diff --stat "$BASE"..HEAD -- $P | tail -40
files=$(git diff --name-only "$BASE"..HEAD -- $P | grep '\.go$' | grep -v '_test\.go$')
echo; echo "# file sản xuất (không test): $(echo "$files" | grep -c .)"
scan() {  # scan <tên> <regex> — chỉ in dòng THÊM VÀO (diff +) ở file sản xuất
  local hits; hits=$(git diff -U0 "$BASE"..HEAD -- $files 2>/dev/null | grep -E '^\+[^+]' | grep -E -- "$2")
  printf '%-44s %s\n' "$1" "$(echo "$hits" | grep -c .)"; [ -n "$hits" ] && echo "$hits" | head -8 | sed 's/^/    /'; }
echo; echo "# ứng viên (0 = sạch; >0 → đọc từng dòng)"
scan "float64 (cấm cho điểm; auth/httpapi không cần)"    'float64'
scan "OFFSET trong SQL/Go (cấm)"                         '\bOFFSET\b|\.Offset\('
scan "fmt.Print*/log.Print* (phải slog)"                 'fmt\.Print|log\.Print|println\('
scan "os.Getenv ngoài config"                            'os\.Getenv|os\.LookupEnv'
scan "math/rand (jti/token phải crypto/rand)"            '"math/rand'
scan "InsecureSkipVerify"                                'InsecureSkipVerify'
scan "so sánh mật khẩu/hash bằng =="                     'password.*==|hash.*==|== *password|== *hash'
scan "context.Background()/TODO() (mất deadline/trace)"  'context\.(Background|TODO)\(\)'
scan "time.Sleep trong mã sản xuất"                      'time\.Sleep'
scan "ghép chuỗi vào SQL"                                'Sprintf\("[^"]*(SELECT|INSERT|UPDATE|DELETE)|"\s*\+\s*[a-zA-Z_.]+\s*\+\s*"[^"]*(WHERE|FROM)'
scan "biến toàn cục có thể đổi (var tên thường ở cấp gói)" '^\+var [a-z][A-Za-z0-9_]* '
scan "panic( (chỉ chấp nhận ở khởi tạo)"                  'panic\('
scan "ghi file/đĩa cục bộ"                               'os\.(Create|WriteFile|OpenFile|Mkdir)|ioutil\.'
scan "log có token/password/hash/Authorization"          'slog\.[A-Za-z]+\(.*(token|password|hash|Authorization|secret)'
scan "header Authorization ghi vào log/response"          'Header\(\)\.Set\("Authorization|Header\.Get\("Authorization"\).*slog'
scan "alg không cố định (None/any method)"               'SigningMethodNone|WithoutClaimsValidation|ParseUnverified'
scan "tin X-Forwarded-For không qua CIDR"                'X-Forwarded-For'
scan "định danh lấy từ body/query (student_code/user_id)" 'FormValue\("(user_id|student_code)|Query\(\)\.Get\("(user_id|student_code)'
echo; echo "# kiểm tay theo checklist (tc-GATE-PG.md TC-GATE-30): lỗi bọc %w; handler mỏng; CourseAccessGuard mặc định từ chối; 5xx/429 không lưu idempotency;"
echo "#   cursor so sánh hàng (created_at,id); ETag băm; recover giữ http.ErrAbortHandler; mọi response lỗi qua một hàm viết lỗi; rate limit fail-open có log hạn chế."
