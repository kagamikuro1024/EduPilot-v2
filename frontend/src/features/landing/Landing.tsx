import Image from "next/image";
import Link from "next/link";
import { ButtonLink } from "@/shared/ui/Button";
import s from "./Landing.module.css";

// Trang giới thiệu (Persuade): HTML tĩnh từ máy chủ, không gọi API. Số liệu lấy từ báo cáo sprint 6, 8, 9 (docs/sprints/*/report.md); ảnh chụp từ bản demo đã seed.
const REPO = "https://github.com/kagamikuro1024/EduPilot-v2";

const PROOFS = [
  { stat: "0 / 146", title: "Tên và MSSV được che trước mọi lời gọi AI", line: "Quét 146 yêu cầu gửi nhà cung cấp: không còn MSSV hay họ tên trong danh sách lớp." },
  { stat: "30 / 30", title: "AI soạn nháp, giảng viên quyết, điểm do mã tính", line: "30 sinh viên mẫu có điểm học phần khớp đáp án tính tay. AI chỉ trích công thức từ quy chế, giảng viên xác nhận." },
  { stat: "+0,05 %", title: "Chấm hàng loạt không làm chậm trò chuyện", line: "Chấm 22 bài cùng lúc, độ trễ chat của 20 sinh viên tăng 0,05 % (k6, nhà cung cấp AI mô phỏng)." },
] as const;

const TEACHER = [
  ["Hôm nay", "Việc cần xử lý xếp theo mức khẩn: yêu cầu vào lớp, câu hỏi AI chưa chắc, bài chờ duyệt."],
  ["Chấm bài", "AI chấm nháp theo rubric kèm đoạn trích từ bài. Bạn sửa từng tiêu chí, duyệt, rồi mới công bố."],
  ["Nhận bài", "Từ ứng dụng, tệp ZIP hoặc thư email. Bài chưa rõ của ai nằm ở hàng riêng chờ gán."],
  ["Sổ điểm", "Công thức trích từ quy chế và do bạn xác nhận. Mỗi điểm có giải trình từng dòng; chốt thành bản bất biến."],
  ["Threads", "AI soạn trả lời nháp; giảng viên xác nhận trước khi sinh viên thấy."],
  ["Theo dõi lớp", "Điểm danh, phát biểu, thời gian học và cảnh báo sinh viên có nguy cơ."],
] as const;

const STUDENT = [
  ["Chat riêng", "Hỏi về điểm, chuyên cần, lịch hoặc nội dung bài học. Câu trả lời kèm nguồn trong tài liệu của lớp."],
  ["Nộp bài tập", "Nộp tệp ngay trên trang. Điểm và nhận xét theo từng tiêu chí chỉ hiện sau khi giảng viên công bố."],
  ["Kết quả của tôi", "Điểm học phần có giải trình từng dòng; yêu cầu xem lại điểm khi cần."],
  ["Luyện đề và bài thi", "Luyện đề trắc nghiệm; bài thi có phần lập trình C++ chấm tự động."],
  ["Threads", "Hỏi cả lớp và đọc câu trả lời đã được giảng viên xác nhận."],
  ["Lịch và thư viện", "Buổi học, bài thi và tài liệu ở một chỗ, dùng được trên điện thoại."],
] as const;

function Items({ rows }: { rows: ReadonlyArray<readonly [string, string]> }) {
  return (
    <dl className={s.items}>
      {rows.map(([t, d]) => (
        <div key={t} className={s.item}>
          <dt>{t}</dt>
          <dd>{d}</dd>
        </div>
      ))}
    </dl>
  );
}

export function Landing() {
  return (
    <div className={s.page} data-part="landing">
      <header className={s.top}>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src="/brand/logo-edupilot.svg" alt="EduPilot" height={36} className={s.logo} />
        <ButtonLink href="/login" variant="ghost" size="sm">Đăng nhập</ButtonLink>
      </header>

      <main id="main">
        <section className={s.hero} aria-labelledby="hero-title">
          <h1 id="hero-title" className={s.display}>Vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân</h1>
          <p className={s.lead}>Hỏi đáp, chấm bài, sổ điểm và theo dõi lớp trong một nơi. AI soạn nháp; giảng viên là người quyết.</p>
          <div className={s.actions}>
            <ButtonLink href="/login" variant="primary">Đăng nhập</ButtonLink>
            <ButtonLink href="/register" variant="secondary">Tạo tài khoản</ButtonLink>
          </div>
          <Image className={s.shot} src="/landing/giang-vien-duyet-bai.webp" width={1440} height={900} priority unoptimized sizes="(min-width: 1200px) 1120px, 100vw"
            alt="Màn duyệt bài của giảng viên: bài làm bên trái, điểm từng tiêu chí, đoạn trích và nhận xét nháp bên phải" />
        </section>

        <section className={s.section} aria-labelledby="proof-title">
          <h2 id="proof-title" className={s.h2}>Khác biệt, có số đo</h2>
          <ol className={s.proofs}>
            {PROOFS.map((p) => (
              <li key={p.stat} className={s.proof}>
                <p className={s.stat}>{p.stat}</p>
                <h3 className={s.h3}>{p.title}</h3>
                <p className={s.body}>{p.line}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className={s.section} aria-labelledby="teacher-title">
          <h2 id="teacher-title" className={s.h2}>Cho giảng viên</h2>
          <div className={s.split}>
            <Items rows={TEACHER} />
            <div className={s.shots}>
              <Image className={s.shot} src="/landing/giang-vien-hom-nay.webp" width={1440} height={900} unoptimized sizes="(min-width: 1200px) 520px, 100vw" alt="Màn Hôm nay của giảng viên: ba việc cần xử lý và buổi học sắp tới" />
              <Image className={s.shot} src="/landing/giang-vien-so-diem.webp" width={1440} height={900} unoptimized sizes="(min-width: 1200px) 520px, 100vw" alt="Sổ điểm của lớp: điểm giữa kỳ, cuối kỳ, điểm học phần và điểm chữ từng sinh viên" />
            </div>
          </div>
        </section>

        <section className={s.section} aria-labelledby="student-title">
          <h2 id="student-title" className={s.h2}>Cho sinh viên</h2>
          <div className={s.split}>
            <Items rows={STUDENT} />
            <div className={s.shotsStudent}>
              <Image className={s.shot} src="/landing/sinh-vien-chat-rieng.webp" width={1440} height={900} unoptimized sizes="(min-width: 1200px) 520px, 100vw" alt="Chat riêng của sinh viên: câu trả lời có nguồn tham khảo trong tài liệu lớp và dòng báo đã ẩn thông tin cá nhân" />
              <Image className={s.phone} src="/landing/sinh-vien-375-hom-nay.webp" width={750} height={1624} unoptimized sizes="240px" alt="Màn Hôm nay của sinh viên trên điện thoại" />
            </div>
          </div>
        </section>
      </main>

      <footer className={s.foot}>
        <p>Đồ án tốt nghiệp · Lê Văn Quang Trung</p>
        <p>
          <Link href={REPO} className={s.link} rel="noopener noreferrer" target="_blank">Mã nguồn trên GitHub</Link>
        </p>
        <p className={s.note}>Dữ liệu trên bản demo là dữ liệu mô phỏng.</p>
      </footer>
    </div>
  );
}
