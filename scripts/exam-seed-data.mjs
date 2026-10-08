// Dữ liệu mẫu của US-PE-09 (SRS FEAT-weekly-exam 4.11): ngân hàng câu hỏi, hai bài code, mẫu đáp án CỐ ĐỊNH của sinh viên và mô hình chấm ĐỘC LẬP (số hữu tỉ bằng BigInt, không dùng mã của gateway)
// để `scripts/gen-expected-exam-scores.mjs` tính `seed/expected_exam_scores.csv`. `seed.mjs` và `check-exam-seed.mjs` dùng chung tệp này; không có số ngẫu nhiên.

export const TOPICS = ["Mật mã đối xứng", "Mật mã công khai", "Hàm băm", "Xác thực", "Mạng"];

/** Dựng một câu: `o` = các đáp án (chuỗi); `ok` = chỉ số đáp án đúng. */
const q = (topic, difficulty, type, stem, o, ok, why) => ({ topic, difficulty, type, stem, options: o, correct: ok, explanation: why });
const tf = (topic, difficulty, stem, value, why) => ({ topic, difficulty, type: "TRUE_FALSE", stem, value, explanation: why });
const S = "MCQ_SINGLE";
const M = "MCQ_MULTI";

// 20 câu: 12 MCQ_SINGLE, 4 MCQ_MULTI, 4 TRUE_FALSE; 5 chủ đề; đủ ba mức khó; mỗi câu có giải thích. `title` là khoá tự nhiên (idempotent) trong lớp.
const RAW = [
  q("Mật mã đối xứng", "EASY", S, "Thuật toán nào sau đây là mật mã khối đối xứng?", ["AES", "RSA", "SHA-256", "Diffie-Hellman"], [0], "AES dùng cùng một khoá để mã hoá và giải mã."),
  q("Mật mã đối xứng", "MEDIUM", S, "Độ dài khối của AES là bao nhiêu bit?", ["64", "128", "192", "512"], [1], "AES luôn xử lý khối 128 bit; chỉ độ dài khoá thay đổi."),
  q("Mật mã đối xứng", "HARD", M, "Chế độ vận hành nào sau đây cần vector khởi tạo (IV) hoặc nonce?", ["ECB", "CBC", "CTR", "GCM"], [1, 2, 3], "ECB mã hoá từng khối độc lập nên không dùng IV (và lộ khuôn mẫu); CBC, CTR, GCM cần IV hoặc nonce."),
  tf("Mật mã đối xứng", "EASY", "Hai bên dùng cùng một khoá bí mật trong mật mã đối xứng.", true, "Đó chính là định nghĩa của mật mã đối xứng."),
  q("Mật mã công khai", "EASY", S, "Trong RSA, khoá nào được dùng để KÝ một thông điệp?", ["Khoá công khai của người nhận", "Khoá bí mật của người ký", "Khoá công khai của người ký", "Khoá phiên"], [1], "Chữ ký tạo bằng khoá bí mật của người ký; ai cũng kiểm được bằng khoá công khai."),
  q("Mật mã công khai", "MEDIUM", S, "Bài toán toán học nào là cơ sở an toàn của RSA?", ["Tính logarit rời rạc", "Phân tích số nguyên lớn ra thừa số nguyên tố", "Tìm va chạm hàm băm", "Giải hệ phương trình tuyến tính"], [1], "An toàn của RSA dựa trên độ khó của phân tích thừa số nguyên tố."),
  q("Mật mã công khai", "HARD", M, "Những thuật toán nào sau đây là mật mã khoá công khai?", ["RSA", "ECC", "AES", "ElGamal"], [0, 1, 3], "RSA, ECC và ElGamal là khoá công khai; AES là đối xứng."),
  tf("Mật mã công khai", "MEDIUM", "Khoá bí mật của RSA có thể được suy ra dễ dàng từ khoá công khai.", false, "Nếu suy ra được dễ dàng thì RSA không an toàn."),
  q("Hàm băm", "EASY", S, "Hàm băm nào cho đầu ra 256 bit?", ["MD5", "SHA-1", "SHA-256", "CRC32"], [2], "SHA-256 cho đầu ra 256 bit."),
  q("Hàm băm", "MEDIUM", S, "Tính chất nào mô tả việc khó tìm hai đầu vào khác nhau cho cùng một giá trị băm?", ["Kháng tiền ảnh", "Kháng va chạm", "Hiệu ứng thác đổ", "Tính khả nghịch"], [1], "Đó là tính kháng va chạm."),
  q("Hàm băm", "HARD", M, "Những hàm nào hiện không còn được khuyến nghị cho chữ ký số vì đã có va chạm thực tế?", ["MD5", "SHA-1", "SHA-256", "SHA-3"], [0, 1], "MD5 và SHA-1 đã có va chạm thực tế."),
  tf("Hàm băm", "EASY", "Từ giá trị băm có thể khôi phục lại dữ liệu gốc.", false, "Hàm băm là một chiều."),
  q("Xác thực", "EASY", S, "Yếu tố xác thực nào là \"thứ bạn biết\"?", ["Vân tay", "Mật khẩu", "Thẻ từ", "Giọng nói"], [1], "Mật khẩu là yếu tố kiến thức."),
  q("Xác thực", "MEDIUM", S, "Vì sao nên lưu mật khẩu bằng hàm băm có muối (salt)?", ["Để giảm dung lượng lưu trữ", "Để cùng mật khẩu cho ra băm khác nhau và chống bảng cầu vồng", "Để mã hoá được hai chiều", "Để tăng tốc độ đăng nhập"], [1], "Muối làm mỗi băm duy nhất và vô hiệu bảng tính sẵn."),
  q("Xác thực", "HARD", M, "Những biện pháp nào giúp chống tấn công dò mật khẩu trực tuyến?", ["Giới hạn số lần thử", "Khoá tạm thời tài khoản", "Lưu mật khẩu dạng rõ", "Xác thực hai yếu tố"], [0, 1, 3], "Giới hạn thử, khoá tạm và hai yếu tố đều làm chậm hoặc chặn việc dò mật khẩu."),
  tf("Xác thực", "MEDIUM", "Xác thực hai yếu tố chỉ cần hai mật khẩu khác nhau.", false, "Hai yếu tố phải thuộc hai loại khác nhau (biết / có / là)."),
  q("Mạng", "EASY", S, "Giao thức nào cung cấp mã hoá cho trang web?", ["HTTP", "FTP", "TLS", "ICMP"], [2], "TLS là lớp bảo mật của HTTPS."),
  q("Mạng", "MEDIUM", S, "Tường lửa lọc gói tin hoạt động chủ yếu dựa trên thông tin nào?", ["Nội dung tệp đính kèm", "Địa chỉ và cổng trong tiêu đề gói", "Mật khẩu người dùng", "Chữ ký số"], [1], "Bộ lọc gói xét địa chỉ IP, cổng và giao thức."),
  q("Mạng", "HARD", S, "Tấn công ARP spoofing nhằm mục đích gì?", ["Làm đầy bảng định tuyến", "Gắn địa chỉ MAC của kẻ tấn công với IP của nạn nhân", "Phá khoá TLS", "Làm quá tải DNS"], [1], "Kẻ tấn công giả mạo ánh xạ IP–MAC để chen vào giữa."),
  q("Mạng", "MEDIUM", S, "Cổng mặc định của HTTPS là bao nhiêu?", ["21", "80", "443", "8080"], [2], "HTTPS dùng cổng 443."),
];
export const QUESTIONS = RAW.map((x, i) => ({ ...x, title: `Câu ${String(i + 1).padStart(2, "0")} — ${x.topic}` }));

// 10 câu của bài trắc nghiệm mẫu (chỉ số vào QUESTIONS): 6 chọn một, 2 nhiều đáp án, 2 đúng/sai; mỗi câu 1 điểm, thang 10, PARTIAL.
export const EXAM_MCQ = [0, 1, 4, 5, 8, 16, 2, 6, 3, 7];

// ---- mẫu đáp án cố định -----------------------------------------------------------------------------------------------
/** `ok` đúng hết; `bad` chọn đáp án sai; `half` (nhiều đáp án) chỉ chọn MỘT đáp án đúng; `none` bỏ trống. */
export function kindFor(student, qpos, isMulti) {
  if (student === "A") return "ok";
  if (student === "B") return qpos === 1 || qpos === 8 ? "bad" : "ok"; // B 8/10: sai câu 2 và câu 9
  if (qpos === 4) return (student * 3) % 10 < 7 ? "bad" : "ok"; // câu 5 khó: ≤ 40 % đúng (AC4: Câu sai nhiều)
  const n = (student * 7 + qpos * 3) % 11; // student ≥ 4 (sv04…sv25)
  if (n <= 5 || n === 10) return "ok";
  if (n === 6 || n === 9) return "bad";
  if (n === 7) return isMulti ? "half" : "ok";
  return "none";
}

/** Chỉ số đáp án sinh viên chọn (hoặc `null` = bỏ trống / giá trị đúng-sai) theo `kind`. */
export function choose(question, kind) {
  if (kind === "none") return null;
  if (question.type === "TRUE_FALSE") return { value: kind === "ok" ? question.value : !question.value };
  const all = question.options.map((_, i) => i);
  const wrong = all.filter((i) => !question.correct.includes(i));
  if (kind === "ok") return { idx: question.correct };
  if (kind === "half") return { idx: [question.correct[0]] };
  return { idx: [wrong[0]] }; // bad: một đáp án sai
}

/** Điểm một câu theo PARTIAL, hữu tỉ [tử, mẫu] (1 điểm / câu) — mô hình độc lập với Quiz Engine. */
export function earnedMcq(question, ans) {
  if (ans == null) return [0n, 1n];
  if (question.type === "TRUE_FALSE") return [ans.value === question.value ? 1n : 0n, 1n];
  const tp = ans.idx.filter((i) => question.correct.includes(i)).length;
  const fp = ans.idx.length - tp;
  const net = tp - fp;
  if (net <= 0) return [0n, 1n];
  return [BigInt(net), BigInt(question.correct.length)];
}

/** Sinh viên làm bài trắc nghiệm: A, B và sv04…sv25 (22 người) = 24 lượt. C (sv.nguyco) và sv26…sv30 không làm. */
export const MCQ_STUDENTS = [{ who: "sv.gioi", id: "A" }, { who: "sv.kha", id: "B" }, ...Array.from({ length: 22 }, (_, i) => ({ who: `sv${String(i + 4).padStart(2, "0")}`, id: i + 4 }))];

export function mcqAnswers(id) {
  return EXAM_MCQ.map((qi, pos) => {
    const q = QUESTIONS[qi];
    const kind = kindFor(id, pos, q.type === "MCQ_MULTI");
    return { qi, pos, kind, ans: choose(q, kind) };
  });
}

// ---- bài code ----------------------------------------------------------------------------------------------------------
export const CODE = [
  {
    slug: "gcd-lon-nhat",
    title: "Ước chung lớn nhất",
    topic: "Cơ bản",
    languages: ["c11", "cpp17"],
    stem: "Cho hai số nguyên không âm a và b (không đồng thời bằng 0). Đọc a b từ dòng đầu vào và in ra ước chung lớn nhất của chúng.\n\nVí dụ: đầu vào `12 18` → đầu ra `6`.",
    weights: [1, 1, 2, 2, 2],
    tests: [
      { name: "sample1", input: "12 18\n", expected: "6\n", sample: true },
      { name: "sample2", input: "7 13\n", expected: "1\n", sample: true },
      { name: "hidden1", input: "0 5\n", expected: "5\n", sample: false },
      { name: "hidden2", input: "1000000007 998244353\n", expected: "1\n", sample: false },
      { name: "hidden3", input: "123456 7890\n", expected: "6\n", sample: false },
    ],
    reference: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  long long a, b;\n  cin >> a >> b;\n  while (b) { long long t = a % b; a = b; b = t; }\n  cout << a << \"\\n\";\n  return 0;\n}\n",
  },
  {
    slug: "dem-tu",
    title: "Đếm từ",
    topic: "Xử lý chuỗi",
    languages: ["cpp17"],
    stem: "Đọc một dòng văn bản và in ra số từ. Các từ cách nhau bởi một hoặc nhiều dấu cách; đầu và cuối dòng có thể có dấu cách thừa.\n\nVí dụ: đầu vào `xin chao the gioi` → đầu ra `4`.",
    weights: [1, 1, 1, 1, 1],
    tests: [
      { name: "sample1", input: "xin chao the gioi\n", expected: "4\n", sample: true },
      { name: "sample2", input: "  a  b \n", expected: "2\n", sample: true },
      { name: "hidden1", input: "   \n", expected: "0\n", sample: false },
      { name: "hidden2", input: "mot\n", expected: "1\n", sample: false },
      { name: "hidden3", input: "a b c d e f g h i j\n", expected: "10\n", sample: false },
    ],
    reference: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  string line, w;\n  getline(cin, line);\n  istringstream in(line);\n  int n = 0;\n  while (in >> w) n++;\n  cout << n << \"\\n\";\n  return 0;\n}\n",
  },
];

// Lời giải của sinh viên (cpp17). `gcdOk2` là `gcdOk` đổi tên biến + dàn trang: cặp được gieo cho so độ giống.
export const SOLUTIONS = {
  gcdOk: CODE[0].reference,
  gcdOk2: "#include <bits/stdc++.h>\nusing namespace std;\nint main(){\n    long long x, y;\n    cin >> x >> y;\n    while (y) {\n        long long r = x % y;\n        x = y;\n        y = r;\n    }\n    cout << x << \"\\n\";\n    return 0;\n}\n",
  gcdZero: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  long long a, b;\n  scanf(\"%lld %lld\", &a, &b);\n  if (!a) { puts(\"0\"); return 0; }\n  while (b) { a %= b; swap(a, b); }\n  printf(\"%lld\\n\", a);\n}\n",
  gcdRec: "#include <bits/stdc++.h>\nusing namespace std;\nlong long g(long long a, long long b) { return b == 0 ? a : g(b, a % b); }\nint main() {\n  long long p, q;\n  cin >> p >> q;\n  cout << g(p, q) << endl;\n}\n",
  gcdCe: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  long long a, b\n  cin >> a >> b;\n  cout << a << \"\\n\";\n}\n",
  wordsOk: CODE[1].reference,
  wordsSpaces: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  string line;\n  getline(cin, line);\n  int spaces = 0;\n  for (char c : line) if (c == ' ') spaces++;\n  cout << spaces + 1 << \"\\n\";\n  return 0;\n}\n",
  wordsSpaces2: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  string s;\n  getline(cin, s);\n  printf(\"%d\\n\", (int)count(s.begin(), s.end(), ' ') + 1);\n}\n",
  wordsLoop: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  string s;\n  getline(cin, s);\n  int n = 0;\n  bool inside = false;\n  for (char c : s) {\n    if (c != ' ') {\n      if (!inside) n++;\n      inside = true;\n    } else {\n      inside = false;\n    }\n  }\n  cout << n << \"\\n\";\n}\n",
  wordsStream: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  string w;\n  int total = 0;\n  while (cin >> w) total++;\n  cout << total << endl;\n  return 0;\n}\n",
  wordsFind: "#include <bits/stdc++.h>\nusing namespace std;\nint main() {\n  string s;\n  getline(cin, s);\n  size_t i = 0;\n  int k = 0;\n  while ((i = s.find_first_not_of(' ', i)) != string::npos) {\n    k++;\n    i = s.find(' ', i);\n    if (i == string::npos) break;\n  }\n  cout << k << \"\\n\";\n}\n",
};

/** Sáu lượt code: sinh viên (khoá tài khoản) → { gcd, words } = tên lời giải (hoặc null = không nộp). */
export const CODE_ATTEMPTS = [
  { who: "sv.gioi", label: "A", gcd: "gcdOk", words: "wordsOk" },
  { who: "sv.kha", label: "B", gcd: "gcdZero", words: "wordsLoop" },
  { who: "sv04", label: "S04", gcd: "gcdCe", words: "wordsSpaces" },
  { who: "sv05", label: "S05", gcd: "gcdOk2", words: "wordsStream" }, // cặp (A, S05) giống nhau
  { who: "sv06", label: "S06", gcd: "gcdRec", words: "wordsSpaces2" },
  { who: "sv07", label: "S07", gcd: null, words: "wordsFind" },
];

/** Kết quả mong đợi của một lời giải trên một bài: { passedWeight, verdict } theo mô hình độc lập (chạy "bằng tay" các test). */
export function expectedCode(problemIdx, solution) {
  const p = CODE[problemIdx];
  const total = p.weights.reduce((a, b) => a + b, 0);
  if (solution == null) return { passed: 0, total, verdict: "" };
  if (solution === "gcdCe") return { passed: 0, total, verdict: "CE" };
  // gcdZero sai ở test có a = 0 (hidden1); wordsSpaces sai ở các test có dấu cách thừa (sample2, hidden1); còn lại đúng hết
  const failing = solution === "gcdZero" ? [2] : solution.startsWith("wordsSpaces") ? [1, 2] : [];
  const passed = p.weights.reduce((a, w, i) => a + (failing.includes(i) ? 0 : w), 0);
  return { passed, total, verdict: failing.length ? "WA" : "AC" };
}

/** Mẫu số chung / làm tròn nửa lên về 0,01 từ hữu tỉ [n, d] (điểm trên thang 10). */
export function round2(n, d) {
  const scaled = (n * 100n * 2n + d) / (d * 2n); // nửa lên, n ≥ 0
  const s = scaled.toString().padStart(3, "0");
  return `${s.slice(0, -2)}.${s.slice(-2)}`;
}
