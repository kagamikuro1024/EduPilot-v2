import { getImageProps } from "next/image";
import { preload } from "react-dom";
import Link from "next/link";
import { ButtonLink } from "@/shared/ui/Button";
import s from "./Landing.module.css";

// Trang giới thiệu (Persuade): HTML tĩnh từ máy chủ, không gọi API. Số liệu lấy từ báo cáo sprint 6, 8, 9 (docs/sprints/*/report.md); ảnh chụp từ bản demo đã seed.
const REPO = "https://github.com/kagamikuro1024/EduPilot-v2";

const PROOFS = [
  { stat: [146], title: "yêu cầu gửi AI, 0 lần lộ họ tên hay MSSV", line: "Quét toàn bộ yêu cầu gửi nhà cung cấp AI: danh sách lớp không còn họ tên hay MSSV nào." },
  { stat: [30, "/", 30], title: "sinh viên mẫu có điểm khớp tính tay", line: "Điểm học phần do mã tính từ quy chế, khớp đáp án tính tay. AI chỉ trích công thức, giảng viên xác nhận." },
  { stat: [22], title: "bài chấm cùng lúc, chat không chậm thêm", line: "Chấm 22 bài trong lúc 20 sinh viên đang chat: độ trễ chat tăng 0,05 %." },
] as const;

// Ví dụ minh hoạ cho dòng chảy bài nộp: số mẫu, không phải số đo.
const FLOW = [
  { title: "AI soạn nháp", line: "Chấm theo rubric, mỗi tiêu chí kèm đoạn trích từ bài làm.", tag: "Nháp" },
  { title: "Giảng viên duyệt", line: "Sửa từng tiêu chí, duyệt, rồi mới công bố.", tag: "Đã duyệt" },
  { title: "Mã tính điểm", line: "Công thức trích từ quy chế và do giảng viên xác nhận. Không có AI trong phép tính.", tag: "Mã tính" },
] as const;

// Lời thay cho video không tiếng (WCAG 1.2.1); phải khớp phụ đề in trong video (scripts/landing-video/record.mjs, mảng CAPTIONS).
const VIDEO_CAPTIONS = [
  "Sinh viên hỏi riêng về điểm và chuyên cần của chính mình.",
  "Họ tên và MSSV được che trước khi gửi cho AI; câu trả lời kèm nguồn trong tài liệu của lớp.",
  "Câu hỏi AI chưa chắc được chuyển tới hộp thư của giảng viên, kèm lý do.",
  "Giảng viên thấy hàng chờ chấm, AI chấm nháp theo rubric kèm đoạn trích từ bài làm.",
  "Giảng viên sửa điểm, duyệt rồi mới công bố. AI không tự công bố.",
  "Sổ điểm: điểm học phần do mã tính từ quy chế, mỗi điểm có giải trình từng dòng.",
  "Hôm nay: việc cần giảng viên quyết, xếp theo mức khẩn.",
] as const;

const TEACHER = [
  ["Hôm nay", "Việc cần xử lý xếp theo mức khẩn: yêu cầu vào lớp, câu hỏi AI chưa chắc, bài chờ duyệt."],
  ["Chấm bài", "AI chấm nháp theo rubric kèm đoạn trích từ bài. Bạn sửa từng tiêu chí, duyệt, rồi mới công bố."],
  ["Sổ điểm", "Công thức trích từ quy chế và do bạn xác nhận. Mỗi điểm có giải trình từng dòng; chốt thành bản bất biến."],
  ["Theo dõi lớp", "Điểm danh, phát biểu, thời gian học và cảnh báo sinh viên có nguy cơ."],
] as const;

const STUDENT = [
  ["Chat riêng", "Hỏi về điểm, chuyên cần, lịch hoặc nội dung bài học. Câu trả lời kèm nguồn trong tài liệu của lớp."],
  ["Nộp bài tập", "Nộp tệp ngay trên trang. Điểm và nhận xét theo từng tiêu chí chỉ hiện sau khi giảng viên công bố."],
  ["Kết quả của tôi", "Điểm học phần có giải trình từng dòng; yêu cầu xem lại điểm khi cần."],
  ["Threads", "Hỏi cả lớp và đọc câu trả lời đã được giảng viên xác nhận."],
] as const;

/** Ảnh chụp màn hình (đều dưới màn hình đầu, nạp lười để không tranh băng thông với poster video là ảnh LCP): tệp WebP đã nén sẵn (không qua bộ tối ưu của Next), điện thoại (≤ 719 px) tải bản 640 px; có width / height để không dịch bố cục. */
function Shot({ name, alt, className, sizes }: { name: string; alt: string; className?: string; sizes: string }) {
  const { props } = getImageProps({ src: `/landing/${name}.webp`, width: 1440, height: 900, alt, unoptimized: true, loading: "lazy", className, sizes });
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
  // Poster video là ảnh lớn nhất lần vẽ đầu ở ≥ 720 px: tải ngay từ <head>. Ở điện thoại khối chữ đầu trang chiếm trọn màn hình đầu, video nằm dưới nếp gấp nên chữ là LCP
  // (poster 52 KB tranh băng thông 4G chậm với phông + JS làm LCP 3,1 s khi nó nằm trong màn hình đầu).
  preload("/landing/gioi-thieu-v1.webp", { as: "image", media: "(min-width: 720px)", fetchPriority: "high" });
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
          <div className={s.heroText}>
            <h1 id="hero-title" className={s.display}>Vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân</h1>
            <span className={s.thread} aria-hidden />
            <p className={s.lead}>AI soạn nháp. Giảng viên quyết. Điểm do mã tính.</p>
            <div className={s.actions}>
              <ButtonLink href="/login" variant="primary">Đăng nhập</ButtonLink>
              <ButtonLink href="/register" variant="secondary">Tạo tài khoản</ButtonLink>
            </div>
          </div>
          <div className={s.heroShot}>
            <video className={s.video} controls preload="none" playsInline poster="/landing/gioi-thieu-v1.webp" width={1280} height={800} aria-label="Video giới thiệu EduPilot" aria-describedby="video-loi">
              <source src="/landing/gioi-thieu-v1.mp4" type="video/mp4" />
              <source src="/landing/gioi-thieu-v1.webm" type="video/webm" />
            </video>
            <p id="video-loi" className="ep-sr-only">{VIDEO_CAPTIONS.join(" ")}</p>
          </div>
        </section>

        <section className={s.section} aria-labelledby="flow-title">
          <h2 id="flow-title" className={s.h2}>AI soạn nháp, giảng viên quyết, mã tính điểm</h2>
          <ol className={s.flow}>
            {FLOW.map((f, i) => (
              <li key={f.title} className={s.step} style={{ "--i": i } as React.CSSProperties}>
                <span className={s.mark} aria-hidden>{i + 1}</span>
                <h3 className={s.h3}>{f.title}</h3>
                <p className={s.body}>{f.line}</p>
                <div className={s.specimen} aria-hidden>
                  {i < 2 ? (
                    <>
                      <p><span>Tiêu chí 1</span><b>8,0</b></p>
                      <p><span>Tiêu chí 2</span>{i === 0 ? <b>7,5</b> : <b className={s.edit}><s>7,5</s> <em>8,5</em></b>}</p>
                    </>
                  ) : (
                    <>
                      <p><span>0,1 × 9,0 + 0,3 × 8,0 + 0,6 × 8,3</span></p>
                      <p><span>Điểm học phần</span><b>8,3</b></p>
                    </>
                  )}
                  <i className={i === 0 ? s.tagDraft : s.tagDone}>{f.tag}</i>
                </div>
              </li>
            ))}
          </ol>
          <p className={s.note}>Ví dụ minh hoạ, số liệu mẫu.</p>
        </section>

        <section className={s.section} aria-labelledby="proof-title">
          <h2 id="proof-title" className={s.h2}>Khác biệt, có số đo</h2>
          <ol className={s.proofs}>
            {PROOFS.map((p, i) => (
              <li key={p.title} className={s.proof} style={{ "--i": i } as React.CSSProperties}>
                <p className={s.stat}>
                  {p.stat.map((x, i) => (typeof x === "number" ? <span key={i} className={x >= 1 ? s.num : undefined} style={x >= 1 ? ({ "--n": x } as React.CSSProperties) : undefined}>{String(x).replace(".", ",")}</span> : x))}
                </p>
                <h3 className={s.h3}>{p.title}</h3>
                <p className={s.body}>{p.line}</p>
              </li>
            ))}
          </ol>
          <p className={s.note}>Đo trên dữ liệu mô phỏng; nhà cung cấp AI trong phép đo chat là bản mô phỏng.</p>
        </section>

        <section className={s.section} aria-labelledby="roles-title">
          <h2 id="roles-title" className={s.h2}>Cho giảng viên và sinh viên</h2>
          <div className={`${s.roles} ${s.reveal}`}>
            {/* Hai nút radio thay cho tab: đổi vai bằng CSS (:has), không JS; bàn phím dùng mũi tên */}
            <fieldset className={s.tabs}>
              <legend className="ep-sr-only">Xem theo vai</legend>
              <input type="radio" name="role" value="gv" id="role-gv" defaultChecked className={s.radio} />
              <label htmlFor="role-gv">Giảng viên</label>
              <input type="radio" name="role" value="sv" id="role-sv" className={s.radio} />
              <label htmlFor="role-sv">Sinh viên</label>
            </fieldset>
            <div className={`${s.split} ${s.panelGv}`}>
              <Items rows={TEACHER} />
              <div className={s.shots}>
                <Shot className={s.shot} name="giang-vien-hom-nay" sizes="(min-width: 1200px) 520px, 100vw" alt="Màn Hôm nay của giảng viên: ba việc cần xử lý và buổi học sắp tới" />
                <Shot className={s.shot} name="giang-vien-so-diem" sizes="(min-width: 1200px) 520px, 100vw" alt="Sổ điểm của lớp: điểm giữa kỳ, cuối kỳ, điểm học phần và điểm chữ từng sinh viên" />
              </div>
            </div>
            <div className={`${s.split} ${s.panelSv}`}>
              <Items rows={STUDENT} />
              <div className={s.shotsStudent}>
                <Shot className={s.shot} name="sinh-vien-chat-rieng" sizes="(min-width: 1200px) 520px, 100vw" alt="Chat riêng của sinh viên: câu trả lời có nguồn tham khảo trong tài liệu lớp và dòng báo đã ẩn thông tin cá nhân" />
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img className={s.phone} src="/landing/sinh-vien-375-hom-nay-360.webp" width={360} height={780} loading="lazy" alt="Màn Hôm nay của sinh viên trên điện thoại" />
              </div>
            </div>
          </div>
        </section>

        <section className={s.section} aria-labelledby="cta-title">
          <h2 id="cta-title" className={s.h2}>Bắt đầu với lớp của bạn</h2>
          <div className={s.actions}>
            <ButtonLink href="/login" variant="primary">Đăng nhập</ButtonLink>
            <ButtonLink href="/register" variant="secondary">Tạo tài khoản</ButtonLink>
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
