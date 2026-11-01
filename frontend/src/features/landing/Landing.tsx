import { getImageProps } from "next/image";
import { preload } from "react-dom";
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

/** Ảnh chụp màn hình: tệp WebP đã nén sẵn (không qua bộ tối ưu của Next), điện thoại (≤ 719 px) tải bản 640 px; có width / height để không dịch bố cục. */
function Shot({ name, alt, lazy, className, sizes }: { name: string; alt: string; lazy?: boolean; className?: string; sizes: string }) {
  const { props } = getImageProps({ src: `/landing/${name}.webp`, width: 1440, height: 900, alt, unoptimized: true, loading: lazy ? "lazy" : undefined, className, sizes });
  const small = getImageProps({ src: `/landing/${name}-640.webp`, width: 640, height: 400, alt, unoptimized: true });
  return (
    <picture>
      <source media="(max-width: 719px)" srcSet={small.props.src} width={640} height={400} />
      <img {...props} alt={alt} />
    </picture>
  );
}

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
  // Ảnh đầu trang chỉ có ở ≥ 720 px (điện thoại ẩn nó để chữ là ứng viên LCP, ảnh 117 KB làm LCP trượt ngưỡng 2,5 s trên 4G chậm): tải ngay từ <head>.
  preload("/landing/giang-vien-duyet-bai.webp", { as: "image", media: "(min-width: 720px)", fetchPriority: "high" });
  return (
    <div className={s.page} data-part="landing">
      <header className={s.top}>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src="/brand/logo-edupilot.svg" alt="EduPilot" height={36} className={s.logo} />
        <ButtonLink href="/login" variant="ghost" size="sm">Đăng nhập</ButtonLink>
      </header>

      {/* role="main" thay vì <main>: HTML của `/` còn mang cả khung PreShell (có <main id="main">) cho người quay lại; chỉ một khung hiện, khung kia display:none nên không vào cây truy cập */}
      <div role="main" id="landing-main">
        <section className={s.hero} aria-labelledby="hero-title">
          <h1 id="hero-title" className={s.display}>Vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân</h1>
          <p className={s.lead}>Hỏi đáp, chấm bài, sổ điểm và theo dõi lớp trong một nơi. AI soạn nháp; giảng viên là người quyết.</p>
          <div className={s.actions}>
            <ButtonLink href="/login" variant="primary">Đăng nhập</ButtonLink>
            <ButtonLink href="/register" variant="secondary">Tạo tài khoản</ButtonLink>
          </div>
          <div className={s.heroShot}>
            <Shot className={s.shot} name="giang-vien-duyet-bai" lazy sizes="(min-width: 1200px) 1120px, 100vw"
              alt="Màn duyệt bài của giảng viên: bài làm bên trái, điểm từng tiêu chí, đoạn trích và nhận xét nháp bên phải" />
          </div>
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
              <Shot className={s.shot} name="giang-vien-hom-nay" sizes="(min-width: 1200px) 520px, 100vw" alt="Màn Hôm nay của giảng viên: ba việc cần xử lý và buổi học sắp tới" />
              <Shot className={s.shot} name="giang-vien-so-diem" sizes="(min-width: 1200px) 520px, 100vw" alt="Sổ điểm của lớp: điểm giữa kỳ, cuối kỳ, điểm học phần và điểm chữ từng sinh viên" />
            </div>
          </div>
        </section>

        <section className={s.section} aria-labelledby="student-title">
          <h2 id="student-title" className={s.h2}>Cho sinh viên</h2>
          <div className={s.split}>
            <Items rows={STUDENT} />
            <div className={s.shotsStudent}>
              <Shot className={s.shot} name="sinh-vien-chat-rieng" sizes="(min-width: 1200px) 520px, 100vw" alt="Chat riêng của sinh viên: câu trả lời có nguồn tham khảo trong tài liệu lớp và dòng báo đã ẩn thông tin cá nhân" />
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img className={s.phone} src="/landing/sinh-vien-375-hom-nay-360.webp" width={360} height={780} loading="lazy" alt="Màn Hôm nay của sinh viên trên điện thoại" />
            </div>
          </div>
        </section>
      </div>

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
