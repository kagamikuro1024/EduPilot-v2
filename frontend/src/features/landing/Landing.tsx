import { Play } from "lucide-react";
import { preload } from "react-dom";
import Link from "next/link";
import { Button, ButtonLink } from "@/shared/ui/Button";
import s from "./Landing.module.css";
import { LandingMedia } from "./LandingMedia";

// Trang giới thiệu v2 (Persuade, DESIGN.md §11.1): HTML tĩnh từ máy chủ, không gọi API. Số liệu lấy từ báo cáo sprint 6, 8, 9 (docs/sprints/*/report.md); ảnh chụp từ bản demo đã seed
// (scripts/landing-video). Chuyển động hoàn toàn bằng CSS (`animation-timeline`); JS thêm duy nhất là LandingMedia (clip màn đầu + hộp thoại video).
const REPO = "https://github.com/kagamikuro1024/EduPilot-v2";

// Mỗi số một câu người ngoài hiểu ngay, khớp từng chữ với nguồn (US-UX91-08 AC9). Chỉ số nguyên (146) đếm lên; "30/30", "0,05 %" hiện tĩnh.
const PROOFS = [
  { stat: [146], title: "yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp", line: "Bộ phát hiện đạt recall 0,960 trên 200 mẫu tự soạn." }, // docs/sprints/6/report.md dòng 34–35
  { stat: [30, "/", 30], title: "sinh viên mẫu có điểm học phần khớp tính tay", line: "Điểm do mã tính từ quy chế; AI chỉ trích công thức, giảng viên xác nhận." }, // docs/sprints/8/report.md dòng 24
  { stat: ["+", 0.05, " %"], title: "Chấm hàng loạt: thời gian đến chữ đầu tiên của chat chỉ tăng 0,05 %", line: "Đo bằng k6 với nhà cung cấp AI mô phỏng." }, // docs/sprints/9/report.md dòng 25 (TTFT +0,05 %)
] as const;

// Ví dụ minh hoạ cho dòng chảy bài nộp: số mẫu, không phải số đo.
const FLOW = [
  { title: "AI soạn nháp", tag: "Nháp" },
  { title: "Giảng viên duyệt", tag: "Đã duyệt" },
  { title: "Mã tính điểm", tag: "Mã tính" },
] as const;

// Kể chuyện theo cuộn: khung ghim đổi Chat → Duyệt bài → Sổ điểm (AC6). Ảnh chụp bằng scripts/landing-video/run.sh từ bản demo.
const STORY = [
  { name: "story-chat-v1", title: "Hỏi riêng, có nguồn", line: "Họ tên và MSSV được che trước khi gửi cho AI; câu trả lời kèm nguồn trong tài liệu của lớp.", alt: "Chat riêng của sinh viên: câu trả lời có nguồn tham khảo trong tài liệu lớp và dòng báo đã ẩn thông tin cá nhân" },
  { name: "story-review-v1", title: "AI chấm nháp, giảng viên duyệt", line: "Mỗi tiêu chí có điểm nháp và đoạn trích từ bài làm; giảng viên sửa rồi mới duyệt và công bố.", alt: "Màn duyệt bài của giảng viên: bài làm bên trái, điểm từng tiêu chí, đoạn trích và nhận xét nháp bên phải" },
  { name: "story-gradebook-v1", title: "Điểm có giải trình", line: "Điểm học phần do mã tính từ quy chế; mỗi điểm mở ra giải trình từng dòng.", alt: "Sổ điểm của lớp với bảng giải trình từng dòng cách tính điểm học phần của một sinh viên" },
] as const;

const POSTER_ALT = "Màn duyệt bài của giảng viên: AI chấm nháp theo rubric, kèm đoạn trích từ bài làm.";

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
  ["Sổ điểm", "Công thức trích từ quy chế và do bạn xác nhận. Mỗi điểm có giải trình từng dòng."],
  ["Theo dõi lớp", "Điểm danh, phát biểu, thời gian học và cảnh báo sinh viên có nguy cơ."],
] as const;

const STUDENT = [
  ["Chat riêng", "Hỏi về điểm, chuyên cần, lịch hoặc nội dung bài học. Câu trả lời kèm nguồn trong tài liệu của lớp."],
  ["Nộp bài tập", "Nộp tệp ngay trên trang. Điểm và nhận xét theo từng tiêu chí chỉ hiện sau khi giảng viên công bố."],
  ["Kết quả của tôi", "Điểm học phần có giải trình từng dòng; yêu cầu xem lại điểm khi cần."],
  ["Hỏi đáp công khai", "Hỏi cả lớp và đọc câu trả lời đã được giảng viên xác nhận."],
] as const;

/** Ảnh chụp màn hình (đều dưới màn hình đầu, nạp lười): WebP 1600 px (gấp đôi cho khung ~940 px) và 800 px cho điện thoại; có width / height để không dịch bố cục. */
function Shot({ name, alt, className }: { name: string; alt: string; className?: string }) {
  // eslint-disable-next-line @next/next/no-img-element
  return <img src={`/landing/${name}.webp`} srcSet={`/landing/${name}-800.webp 800w, /landing/${name}.webp 1600w`} sizes="(min-width: 1100px) 940px, 100vw" width={1600} height={1000} alt={alt} loading="lazy" decoding="async" className={className} />;
}

function Items({ title, rows }: { title: string; rows: ReadonlyArray<readonly [string, string]> }) {
  return (
    <div className={s.role}>
      <h3 className={s.h3}>{title}</h3>
      <dl className={s.items}>
        {rows.map(([t, d]) => (
          <div key={t} className={s.item}>
            <dt>{t}</dt>
            <dd>{d}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

export function Landing() {
  // ≥ 720 px: poster của clip (1280 px) là LCP, preload ở <head>. Điện thoại: khối chữ cao gần hết màn hình, chữ là LCP; chỉ mép trên của poster 640 px (19 KB, ưu tiên thấp) lộ ở đáy màn đầu — nhỏ hơn diện tích h1 nên không giành LCP (ảnh về lúc 2,7 s trên 4G chậm vì tranh băng thông với JS và phông).
  preload("/landing/gioi-thieu-clip-v1.webp", { as: "image", media: "(min-width: 720px)", fetchPriority: "high" });
  return (
    <div className={s.page} data-part="landing">
      <LandingMedia />
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
            <div className={`${s.actions} ${s.heroActions}`}>
              <ButtonLink href="/login" variant="primary">Đăng nhập</ButtonLink>
              <ButtonLink href="/register" variant="secondary">Tạo tài khoản</ButtonLink>
              <Button variant="text" icon={<Play aria-hidden width={14} height={14} fill="currentColor" />} data-open-video>Xem video</Button>
            </div>
          </div>
          <div className={`${s.heroShot} ${s.frame}`}>
            {/* Poster là <img> (LCP, preload ở <head>) chứ không phải thuộc tính `poster` của video: điện thoại chỉ tải bản 640 px, máy tính chỉ tải bản 1280 px; video nằm đè lên và chỉ hiện khi đã phát */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img className={s.posterSm} src="/landing/gioi-thieu-clip-v1-640.webp" width={640} height={400} alt={POSTER_ALT} fetchPriority="low" />
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img className={s.posterLg} src="/landing/gioi-thieu-clip-v1.webp" width={1280} height={800} alt={POSTER_ALT} fetchPriority="high" />
            {/* clip lặp 6–8 s: LandingMedia nạp và phát sau sự kiện `load`; không tiếng, bỏ qua khi reduce / Save-Data / tab ẩn */}
            <video className={s.video} data-clip muted loop playsInline preload="none" width={1280} height={800} aria-hidden="true" tabIndex={-1} />
            <button type="button" className={s.clipToggle} data-clip-toggle aria-pressed="false" hidden>Tạm dừng</button>
          </div>
        </section>

        <section className={`${s.section} ${s.wide}`} aria-labelledby="story-title">
          <h2 id="story-title" className={s.h2}>Xem EduPilot làm việc</h2>
          <div className={s.story}>
            <div className={`${s.stage} ${s.frame}`} aria-hidden="true">
              {STORY.map((x) => (
                <Shot key={x.name} name={x.name} alt={x.alt} className={s.stageImg} />
              ))}
            </div>
            <ol className={s.steps}>
              {STORY.map((x) => (
                <li key={x.name} className={s.step}>
                  <h3 className={s.h3}>{x.title}</h3>
                  <p className={s.body}>{x.line}</p>
                  <figure className={`${s.stepFig} ${s.frame}`}>
                    <Shot name={x.name} alt={x.alt} />
                  </figure>
                </li>
              ))}
            </ol>
          </div>
        </section>

        <section className={`${s.section} ${s.compact}`} aria-labelledby="flow-title">
          <h2 id="flow-title" className={s.h2}>AI soạn nháp, giảng viên quyết, mã tính điểm</h2>
          <ol className={s.flow}>
            {FLOW.map((f, i) => (
              <li key={f.title} className={s.step2} style={{ "--i": i } as React.CSSProperties}>
                <span className={s.mark} aria-hidden>{i + 1}</span>
                <h3 className={s.h3}>{f.title}</h3>
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
                  {p.stat.map((x, k) => (typeof x === "number" ? <span key={k} className={Number.isInteger(x) && x >= 1 && p.stat.length === 1 ? s.num : undefined} style={Number.isInteger(x) && x >= 1 && p.stat.length === 1 ? ({ "--n": x } as React.CSSProperties) : undefined}>{String(x).replace(".", ",")}</span> : x))}
                </p>
                <h3 className={s.h3}>{p.title}</h3>
                <p className={s.body}>{p.line}</p>
              </li>
            ))}
          </ol>
          <p className={s.note}>Đo trên dữ liệu mô phỏng.</p>
        </section>

        <section className={s.section} aria-labelledby="roles-title">
          <h2 id="roles-title" className={s.h2}>Cho giảng viên và sinh viên</h2>
          <div className={s.roles}>
            <Items title="Giảng viên" rows={TEACHER} />
            <Items title="Sinh viên" rows={STUDENT} />
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

      <dialog className={s.dialog} data-video-dialog aria-label="Video giới thiệu EduPilot">
        <video className={s.dialogVideo} controls preload="none" playsInline data-poster="/landing/gioi-thieu-v1.webp" width={1280} height={800} aria-describedby="video-loi">
          <source src="/landing/gioi-thieu-v1.mp4" type="video/mp4" />
          <source src="/landing/gioi-thieu-v1.webm" type="video/webm" />
        </video>
        <p id="video-loi" className="ep-sr-only">{VIDEO_CAPTIONS.join(" ")}</p>
        <form method="dialog" className={s.dialogBar}>
          <a className={s.link} href="/landing/gioi-thieu-v1.mp4" target="_blank" rel="noopener noreferrer">Mở video trong tab mới</a>
          <Button type="submit" variant="secondary" size="sm">Đóng</Button>
        </form>
      </dialog>

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
