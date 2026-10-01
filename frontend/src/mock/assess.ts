// Dữ liệu chấm bài dùng chung giữa /grading (US-PROTO-03) và /assignments/bt03 (US-PROTO-01).
import type { Bt03State } from "./state";

export const BT03_CRITERIA = ["Xác định tác nhân và bề mặt tấn công", "Phân tích tấn công", "Đánh giá tác động", "Biện pháp phòng thủ"] as const;
export const BT03_MAX_PER_CRITERION = 2.5;

/** Bản chấm nháp của AI (lượt 1). Lượt 2 chấm tiêu chí 2 = 2,5 → lệch 1,5 → "Cần xem kỹ". */
export const BT03_SEED: Bt03State = {
  status: "draft",
  scores: [2.0, 1.0, 2.0, 2.0],
  comments: [
    "Bài nêu rõ nhóm tác nhân và các điểm vào của hệ thống bị tấn công.",
    "Mô tả chuỗi tấn công còn thiếu bước leo thang đặc quyền.",
    "Đánh giá được thiệt hại dữ liệu nhưng chưa định lượng thời gian gián đoạn.",
    "Có đề xuất phân đoạn mạng và xác thực đa yếu tố, chưa nêu cách giám sát.",
  ],
};
/** Lượt chấm thứ hai của AI cho tiêu chí 2 (để hiện thông báo lệch). */
export const BT03_SECOND_PASS_CRITERION_2 = 2.5;

// ---- bài làm của SV B (BT03) ---------------------------------------------------------------

export const BT03 = {
  id: "bt03",
  title: "Phân tích một vụ tấn công thực tế",
  kind: "ESSAY" as const,
  due: new Date("2026-10-22T23:59:00+07:00"),
  latePerDay: 0.5,
  lateMaxDays: 2,
};

export const BT03_SUBMISSION_ID = "sub-bt03-sv-2";

/** Bài nộp của B: nộp 23/10 08:10 = muộn 1 ngày. */
export const BT03_SUBMITTED_AT = new Date("2026-10-23T08:10:00+07:00");

export type EssayPara = { id: string; page: 1 | 2; text: string };

/** Bài làm ~2 trang của Trần Thu Uyên: chiến dịch chuỗi cung ứng SolarWinds Orion (2020). */
export const BT03_ESSAY: EssayPara[] = [
  {
    id: "p1",
    page: 1,
    text: "Bài viết phân tích chiến dịch tấn công chuỗi cung ứng nhằm vào phần mềm giám sát hạ tầng SolarWinds Orion, bị phát hiện tháng 12 năm 2020. Đây là ví dụ điển hình cho việc kẻ tấn công không đánh thẳng vào mục tiêu cuối, mà chọn một nhà cung cấp phần mềm được hàng nghìn tổ chức tin tưởng và cài đặt sẵn trong mạng nội bộ.",
  },
  {
    id: "p2",
    page: 1,
    text: "Tác nhân được các hãng an ninh quy cho nhóm APT29 (còn gọi là Cozy Bear), một nhóm tấn công có tài trợ nhà nước, hoạt động kiên trì và ưu tiên gián điệp hơn phá hoại. Bề mặt tấn công bị khai thác là hệ thống dựng phần mềm của SolarWinds — nơi mã nguồn được biên dịch và ký số trước khi phát hành — chứ không phải máy chủ của khách hàng.",
  },
  {
    id: "p3",
    page: 1,
    text: "Kẻ tấn công cấy một thư viện độc hại tên SUNBURST vào tệp SolarWinds.Orion.Core.BusinessLayer.dll. Vì bản cập nhật vẫn được ký bằng chứng thư số hợp lệ của SolarWinds nên mọi cơ chế kiểm tra chữ ký ở phía khách hàng đều cho qua. Khoảng 18.000 tổ chức tải bản cập nhật có mã độc, nhưng kẻ tấn công chỉ chọn khoảng 100 mục tiêu để khai thác tiếp.",
  },
  {
    id: "p4",
    page: 1,
    text: "Chuỗi tấn công gồm bốn bước: xâm nhập môi trường dựng phần mềm, cấy mã vào bản phát hành chính thức, nằm im 12–14 ngày để vượt qua các phép kiểm tra tự động, rồi gọi về máy chủ điều khiển qua tên miền avsvmcloud.com với lưu lượng giả dạng giao thức cập nhật hợp lệ của chính Orion.",
  },
  {
    id: "p5",
    page: 1,
    text: "Ở giai đoạn sau, nhóm tấn công nạp TEARDROP và Cobalt Strike để di chuyển ngang trong mạng, rồi đánh cắp khoá ký SAML của máy chủ liên kết danh tính. Với khoá này họ tự phát hành thẻ truy cập cho chính mình (kỹ thuật Golden SAML) và leo thang lên quyền quản trị dịch vụ thư điện tử đám mây mà không cần biết mật khẩu hay vượt qua xác thực đa yếu tố.",
  },
  {
    id: "p6",
    page: 2,
    text: "Thiệt hại không nằm ở dữ liệu bị xoá mà ở niềm tin: nhiều cơ quan liên bang Mỹ phải coi toàn bộ hệ thống là đã bị xâm nhập, dựng lại máy chủ và xoay vòng toàn bộ khoá cùng chứng thư. Riêng SolarWinds công bố chi phí ứng phó năm 2021 khoảng 40 triệu đô la, chưa kể thiệt hại thương hiệu và các vụ kiện sau đó.",
  },
  {
    id: "p7",
    page: 2,
    text: "Em chưa tìm được số liệu đáng tin về thời gian gián đoạn dịch vụ của từng tổ chức bị ảnh hưởng, nên phần đánh giá tác động vận hành ở đây mới dừng ở mức định tính.",
  },
  {
    id: "p8",
    page: 2,
    text: "Về phòng thủ, em đề xuất ba nhóm biện pháp: (1) phân đoạn mạng để máy chủ giám sát không có đường đi thẳng ra Internet, (2) xác thực đa yếu tố và giới hạn quyền cho tài khoản dịch vụ, kèm xoay vòng khoá ký định kỳ, (3) kiểm tra tính toàn vẹn của bản dựng bằng quy trình build tái lập được và bản kê thành phần phần mềm (SBOM).",
  },
  {
    id: "p9",
    page: 2,
    text: "Ngoài ra cần giám sát hành vi bất thường của tiến trình hợp lệ, ví dụ Orion tự mở kết nối DNS tới một tên miền lạ ngoài danh sách đã biết, vì chữ ký số hợp lệ không còn là bằng chứng đủ để tin một bản cập nhật.",
  },
  {
    id: "p10",
    page: 2,
    text: "Bài học rút ra: trong mô hình không tin mặc định, phần mềm của bên thứ ba cũng là một bề mặt tấn công. Kiểm soát phải đặt cả ở nơi nhận bản cập nhật chứ không chỉ ở nơi phát hành.",
  },
];

/** Đoạn trích AI dùng làm bằng chứng cho từng tiêu chí (trỏ tới đoạn trong bài làm). */
export const BT03_EVIDENCE: Array<{ para: string; quote: string }> = [
  { para: "p2", quote: "Bề mặt tấn công bị khai thác là hệ thống dựng phần mềm của SolarWinds — nơi mã nguồn được biên dịch và ký số trước khi phát hành" },
  { para: "p5", quote: "đánh cắp khoá ký SAML của máy chủ liên kết danh tính… leo thang lên quyền quản trị dịch vụ thư điện tử đám mây" },
  { para: "p6", quote: "nhiều cơ quan liên bang Mỹ phải coi toàn bộ hệ thống là đã bị xâm nhập, dựng lại máy chủ và xoay vòng toàn bộ khoá cùng chứng thư" },
  { para: "p8", quote: "phân đoạn mạng để máy chủ giám sát không có đường đi thẳng ra Internet… kiểm tra tính toàn vẹn của bản dựng bằng quy trình build tái lập được và bản kê thành phần phần mềm (SBOM)" },
];

// ---- hàng chờ chấm BT03 (28 bài) -----------------------------------------------------------

export type Submission = {
  id: string;
  studentId: string;
  /** điểm nháp của AI, thang 10, trước khi trừ nộp muộn */
  ai: number;
  /** lý do cờ "Cần xem kỹ"; không có = bình thường */
  flag?: string;
  /** đã duyệt sẵn từ trước (B không nằm trong số này — GV duyệt trong kịch bản) */
  approved: boolean;
  submittedAt: Date;
  source: "Tải lên trên EduPilot" | "Nộp qua email lớp";
  lateDays: number;
};

const FLAGS: Record<string, string> = {
  "sv-2": "Hai lượt chấm lệch hơn 1 điểm",
  "sv-9": "Bài ngắn bất thường so với yêu cầu (dưới 400 chữ)",
  "sv-14": "Trùng đoạn với bài của một sinh viên khác",
  "sv-21": "AI không đủ chắc chắn ở tiêu chí Đánh giá tác động",
  "sv-6": "Hai lượt chấm lệch hơn 1 điểm",
  "sv-18": "Bài nộp qua email, chưa khớp định dạng rubric",
};
/** Hai bài có cờ nhưng GV đã duyệt từ hôm qua → lọc mặc định vẫn còn đúng 4 bài. */
const PRE_APPROVED = ["sv-6", "sv-18", "sv-11", "sv-15", "sv-23", "sv-27", "sv-30", "sv-7", "sv-19"];
/** Hai sinh viên chưa nộp BT03 (30 − 2 = 28 bài). */
const NOT_SUBMITTED = ["sv-3", "sv-26"];

/** 28 bài nộp BT03 của lớp 1, tất định. Bài của B luôn đứng đầu khi lọc "Cần xem kỹ". */
export function bt03Submissions(students: Array<{ id: string }>): Submission[] {
  return students
    .filter((s) => !NOT_SUBMITTED.includes(s.id))
    .map((s) => {
      const n = Number(s.id.slice(3));
      const lateDays = s.id === "sv-2" ? 1 : 0;
      const day = 20 + (n % 3);
      const hour = 9 + (n % 11);
      return {
        id: `sub-bt03-${s.id}`,
        studentId: s.id,
        ai: s.id === "sv-2" ? 7.0 : Math.min(10, Math.round((5.75 + ((n * 13) % 35) / 10) * 4) / 4),
        flag: FLAGS[s.id],
        approved: PRE_APPROVED.includes(s.id),
        submittedAt:
          s.id === "sv-2"
            ? BT03_SUBMITTED_AT
            : new Date(`2026-10-${day}T${String(hour).padStart(2, "0")}:${String((n * 7) % 60).padStart(2, "0")}:00+07:00`),
        source: s.id === "sv-18" ? "Nộp qua email lớp" : "Tải lên trên EduPilot",
        lateDays,
      } satisfies Submission;
    });
}

// ---- bài làm của các sinh viên khác (vụ tấn công khác, ngắn hơn) -----------------------------

type Case = { title: string; paras: string[]; evidence: [string, string, string, string] };

const CASES: Case[] = [
  {
    title: "Mã độc tống tiền WannaCry (2017)",
    paras: [
      "Bài viết phân tích đợt lây lan của mã độc tống tiền WannaCry tháng 5 năm 2017, ảnh hưởng tới hơn 200.000 máy tính ở 150 quốc gia, trong đó nặng nhất là hệ thống bệnh viện NHS của Anh.",
      "Tác nhân được quy cho nhóm Lazarus. Bề mặt tấn công là lỗ hổng EternalBlue trong giao thức chia sẻ tệp SMBv1 của Windows, đã có bản vá MS17-010 từ hai tháng trước nhưng nhiều tổ chức chưa cập nhật.",
      "Mã độc tự lây lan trong mạng nội bộ mà không cần người dùng mở tệp: quét cổng 445, khai thác lỗ hổng, mã hoá tệp rồi đòi tiền chuộc bằng Bitcoin. Việc không phân đoạn mạng khiến một máy nhiễm kéo theo cả khoa phòng.",
      "Tác động lớn nhất là gián đoạn dịch vụ: nhiều ca mổ bị hoãn, bệnh án không truy cập được. Phòng thủ cần: vá lỗi theo chu kỳ bắt buộc, tắt SMBv1, phân đoạn mạng và sao lưu ngoại tuyến có kiểm thử phục hồi.",
    ],
    evidence: [
      "Bề mặt tấn công là lỗ hổng EternalBlue trong giao thức chia sẻ tệp SMBv1 của Windows",
      "quét cổng 445, khai thác lỗ hổng, mã hoá tệp rồi đòi tiền chuộc bằng Bitcoin",
      "nhiều ca mổ bị hoãn, bệnh án không truy cập được",
      "vá lỗi theo chu kỳ bắt buộc, tắt SMBv1, phân đoạn mạng và sao lưu ngoại tuyến có kiểm thử phục hồi",
    ],
  },
  {
    title: "Lỗ hổng Log4Shell (2021)",
    paras: [
      "Bài viết phân tích lỗ hổng Log4Shell (CVE-2021-44228) trong thư viện ghi nhật ký Log4j của Java, công bố tháng 12 năm 2021 và bị khai thác hàng loạt chỉ sau vài giờ.",
      "Tác nhân rất đa dạng: từ nhóm đào tiền ảo tới nhóm tấn công có tài trợ nhà nước. Bề mặt tấn công là mọi trường dữ liệu người dùng nhập được ghi vào nhật ký, kể cả tiêu đề User-Agent hay tên thiết bị.",
      "Chuỗi tấn công rất ngắn: gửi chuỗi tra cứu JNDI vào một trường bất kỳ, máy chủ tự tải lớp Java từ máy chủ của kẻ tấn công và thực thi mã từ xa, không cần xác thực.",
      "Tác động là rủi ro thực thi mã từ xa trên hàng triệu hệ thống. Phòng thủ: nâng cấp Log4j, chặn kết nối ra ngoài từ máy chủ ứng dụng, dùng bản kê thành phần phần mềm để biết hệ thống nào bị ảnh hưởng.",
    ],
    evidence: [
      "Bề mặt tấn công là mọi trường dữ liệu người dùng nhập được ghi vào nhật ký",
      "gửi chuỗi tra cứu JNDI vào một trường bất kỳ, máy chủ tự tải lớp Java từ máy chủ của kẻ tấn công",
      "rủi ro thực thi mã từ xa trên hàng triệu hệ thống",
      "nâng cấp Log4j, chặn kết nối ra ngoài từ máy chủ ứng dụng, dùng bản kê thành phần phần mềm",
    ],
  },
  {
    title: "Rò rỉ dữ liệu thẻ của Target (2013)",
    paras: [
      "Bài viết phân tích vụ rò rỉ dữ liệu thẻ thanh toán của chuỗi bán lẻ Target cuối năm 2013, ảnh hưởng khoảng 40 triệu thẻ và 70 triệu hồ sơ khách hàng.",
      "Kẻ tấn công không đánh thẳng vào Target mà lừa đảo lấy tài khoản của một nhà thầu bảo trì điều hoà. Bề mặt tấn công là cổng dành cho nhà cung cấp, nối thẳng được tới mạng nội bộ do thiếu phân đoạn.",
      "Từ tài khoản nhà thầu, kẻ tấn công di chuyển ngang tới hệ thống thanh toán và cài mã độc đọc dữ liệu thẻ ngay trong bộ nhớ máy bán hàng, rồi gom dữ liệu về một máy chủ nội bộ trước khi gửi ra ngoài.",
      "Tác động gồm chi phí bồi thường, thay thẻ và mất niềm tin khách hàng. Phòng thủ: tách mạng nhà cung cấp, xác thực đa yếu tố cho truy cập bên thứ ba, giám sát luồng dữ liệu ra bất thường.",
    ],
    evidence: [
      "Bề mặt tấn công là cổng dành cho nhà cung cấp, nối thẳng được tới mạng nội bộ do thiếu phân đoạn",
      "di chuyển ngang tới hệ thống thanh toán và cài mã độc đọc dữ liệu thẻ ngay trong bộ nhớ máy bán hàng",
      "chi phí bồi thường, thay thẻ và mất niềm tin khách hàng",
      "tách mạng nhà cung cấp, xác thực đa yếu tố cho truy cập bên thứ ba, giám sát luồng dữ liệu ra bất thường",
    ],
  },
];

const OTHER_COMMENTS: [string, string, string, string] = [
  "Nêu được tác nhân và điểm vào của hệ thống, còn thiếu dẫn nguồn cho phần quy kết.",
  "Chuỗi tấn công rõ ràng theo thứ tự thời gian, nên bổ sung bằng chứng kỹ thuật.",
  "Đánh giá tác động đúng hướng nhưng ít số liệu định lượng.",
  "Biện pháp phòng thủ bám sát bài giảng, chưa nói cách kiểm chứng hiệu quả.",
];

/** Nội dung chấm của một bài nộp bất kỳ; bài của SV B dùng dữ liệu riêng (BT03_ESSAY + trạng thái bt03). */
export function submissionDetail(sub: Submission) {
  const n = Number(sub.studentId.slice(3));
  const c = CASES[n % CASES.length];
  const total = Math.round(sub.ai * 4) / 4;
  // chia điểm cho 4 tiêu chí, bước 0,25, không vượt trần
  const each = Math.min(BT03_MAX_PER_CRITERION, Math.round((total / 4) * 4) / 4);
  const scores: [number, number, number, number] = [each, each, each, Math.min(BT03_MAX_PER_CRITERION, Math.round((total - each * 3) * 4) / 4)];
  return {
    title: c.title,
    paras: c.paras.map((text, i) => ({ id: `p${i + 1}`, page: 1 as const, text })),
    evidence: c.evidence.map((quote, i) => ({ para: `p${i + 1}`, quote })),
    scores,
    comments: OTHER_COMMENTS,
  };
}

// ---- bài tập của lớp 1 (tab Bài tập ở /grading) ---------------------------------------------

export type Assignment = {
  id: string;
  title: string;
  kind: "ESSAY" | "QUIZ";
  /** hạn nộp (QUIZ: giờ đóng) */
  due: Date;
  submitted: number;
  size: number;
  state: "published" | "grading" | "open" | "draft";
  note: string;
};

export const ASSIGNMENTS: Assignment[] = [
  { id: "bt01", title: "Mô hình đe doạ", kind: "ESSAY", due: new Date("2026-09-17T23:59:00+07:00"), submitted: 30, size: 30, state: "published", note: "Đã công bố điểm 20/09" },
  { id: "bt02", title: "Tấn công mạng phổ biến", kind: "ESSAY", due: new Date("2026-10-01T23:59:00+07:00"), submitted: 29, size: 30, state: "published", note: "Đã công bố điểm 05/10 · 1 sinh viên thiếu bài" },
  { id: "bt03", title: BT03.title, kind: "ESSAY", due: BT03.due, submitted: 28, size: 30, state: "grading", note: "Cho nộp muộn 2 ngày, trừ 0,5 điểm mỗi ngày" },
  { id: "quiz01", title: "Mật mã đối xứng", kind: "QUIZ", due: new Date("2026-10-30T03:20:00+07:00"), submitted: 21, size: 30, state: "open", note: "Trắc nghiệm tính điểm · 15 câu · đóng sau 18 giờ nữa" },
];
