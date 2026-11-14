import { ArrowRight, Pause, Play, ShieldCheck } from "lucide-react";
import Link from "next/link";
import type { CSSProperties } from "react";
import s from "./Landing.module.css";
import { LandingMotion } from "./LandingMotion";

// Trang giới thiệu v2 theo mẫu `docs/sprints/10.7/mau/landing.html` (P-107-12, DESIGN.md §11.1): HTML tĩnh từ máy chủ, không gọi API. Số liệu khớp từng chữ với báo cáo sprint 6, 8, 9
// (docs/sprints/6/report.md dòng 34–35; 8/report.md dòng 24; 9/report.md dòng 25). Chuyển động bằng CSS + một client component (`LandingMotion`: con trỏ đoạn chạy, tab vai, khung ghim, hiện theo cuộn, hộp video).
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

// tên lớp theo mô-đun; `is-*` là lớp trạng thái toàn cục (CSS viết bằng :global, script bật / tắt) — giữ nguyên để hình mặc định HIỆN (mẫu gắn sẵn `is-on`)
const cx = (v: string) => v.split(" ").map((n) => s[n] ?? (n.startsWith("is-") ? n : "")).filter(Boolean).join(" ");

/** Ảnh chụp màn hình từ bản demo đã seed (scripts/landing-video): `srcset` 640 / 800 / 1280 (không nhúng 1600 px cho mọi bề rộng); có width / height để không dịch bố cục. */
function Shot({ name, alt, className, single }: { name: string; alt: string; className?: string; single?: boolean }) {
  if (name === "sinh-vien-375-hom-nay-360") {
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={`/landing/${name}.webp`} width={360} height={780} alt={alt} loading="lazy" decoding="async" className={className} />;
  }
  const mid = name.startsWith("story-") ? "800w, " : "";
  if (single) {
    // một nguồn 1280 px: ảnh bị cắt vuông ở 390 px nên hiện gần 600 px rộng — bản 640 sẽ bị phóng
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={`/landing/${name}-1280.webp`} width={1280} height={800} alt={alt} loading="lazy" decoding="async" className={className} />;
  }
  const set = `/landing/${name}-640.webp 640w, ${mid ? `/landing/${name}-800.webp ${mid}` : ""}/landing/${name}-1280.webp 1280w`;
  // eslint-disable-next-line @next/next/no-img-element
  return <img src={`/landing/${name}-1280.webp`} srcSet={set} sizes="(min-width: 1024px) 1280px, 100vw" width={1280} height={800} alt={alt} loading="lazy" decoding="async" className={className} />;
}

export function Landing() {
  return (
    <div className={s.page} data-part="landing">
      <script dangerouslySetInnerHTML={{ __html: "if(!matchMedia('(prefers-reduced-motion: reduce)').matches)document.documentElement.classList.add('m-on')" }} />
      <LandingMotion />
      <header className={`${cx("nav")} nav`}>
  <div className={`${cx("wrap")} wrap`}>
    <a className={cx("skip")} href="#landing-main">Tới nội dung</a>
    <a href="#top" aria-label="EduPilot"><img src="/brand/logo-edupilot.svg" alt="" height={36} className={cx("logo")} /></a>
    <div className={cx("nav__end")}><Link className={cx("nav__link")} href="/register">Tạo tài khoản</Link><Link className={cx("btn btn--primary btn--sm")} href="/login">Đăng nhập</Link></div>
  </div>
</header>

<main id="landing-main">
  <section className={cx("sec hero")} id="top" aria-labelledby="hero-title" data-enter>
    <div className={`${cx("wrap")} wrap`}><div className={`${cx("hero-text")} hero-text`}>
      <div className={cx("hero__text")}>
        <h1 className={cx("h1")} id="hero-title"><span className={cx("ln")}>Vận hành lớp học</span> <span className={cx("ln")}>có trợ lý AI</span></h1>
        <span className={cx("stripe")} aria-hidden="true"><i></i><b></b></span>
        <p className={cx("lead")} data-up style={{ "--i": 0 } as CSSProperties}>AI soạn nháp câu trả lời và điểm, giảng viên duyệt rồi mới công bố. Họ tên và MSSV được che trước khi gửi cho AI; điểm học phần do mã tính từ quy chế.</p>
        <div className={cx("actions")} data-up style={{ "--i": 1 } as CSSProperties}><Link className={cx("btn btn--primary")} href="/login">Đăng nhập<ArrowRight className={cx("ic arr")} aria-hidden /></Link><button className={cx("btn btn--secondary")} type="button" data-open-video><Play className={cx("ic")} aria-hidden />Xem video</button></div>
        <p className={cx("proof")} data-up style={{ "--i": 2 } as CSSProperties}><ShieldCheck className={cx("ic")} aria-hidden /><span><b>146</b> yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp</span></p>
      </div>
      <div className={cx("hero__shot")}><div className={cx("frame demo")} data-demo data-ui>
        <div role="img" aria-label="Màn duyệt bài: AI chấm nháp tiêu chí «Giải thích cách khai thác» 7,5 điểm, giảng viên sửa thành 8,5 rồi bấm Duyệt">
          <div aria-hidden="true">
            <p className={cx("demo__bar")}><b>Bài tập 03</b><span>Bài làm 12 / 18 · 761987 An ninh mạng</span></p>
            <div className={cx("demo__body")}>
              <p className={cx("quote")}>«…kẻ tấn công chèn chuỗi vào ô đăng nhập để câu truy vấn luôn đúng, từ đó bỏ qua bước kiểm tra mật khẩu…»</p>
              <div className={cx("rubric")}>
                <p className={cx("rubric__row rubric__row--head")}><span>Tiêu chí</span><span>AI nháp</span><span>Điểm</span></p>
                <p className={cx("rubric__row")}><span>Xác định đúng lỗ hổng</span><span className={cx("rubric__ai")}>8,0</span><b>8,0</b></p>
                <p className={cx("rubric__row rubric__row--edit")}><span>Giải thích cách khai thác</span><span className={cx("rubric__ai")}>7,5</span><span className={cx("stack stack--end")} data-t1><b className={cx("a-out23")}>7,5</b><span className={cx("a-in23 fix")}><s>7,5</s><b>8,5</b></span></span></p>
                <p className={cx("rubric__row")}><span>Đề xuất cách vá</span><span className={cx("rubric__ai")}>9,0</span><b>9,0</b></p>
              </div>
              <div className={cx("demo__foot")}>
                <span className={cx("stack")}><i className={cx("tag tag--draft a-out53")}>Nháp</i><i className={cx("tag tag--done a-in53")}>Đã duyệt</i></span>
                <span className={cx("total")}>Điểm bài<span className={cx("stack")}><b className={cx("a-out23")}>8,2</b><b className={cx("a-in23")}>8,5</b></span></span>
                <span className={cx("stack stack--end")}><span className={cx("chip a-out53")} data-t2>Duyệt</span><span className={cx("done a-in53")}>Duyệt lúc 10:42</span></span>
              </div>
            </div>
            <span className={cx("ptr")}><i className={cx("ptr__ring")}></i><svg className={cx("ptr__i")} viewBox="0 0 24 24" width="24" height="24"><path d="M5 3l14.5 7.6-6.4 1.8-2.6 6.4z" /></svg></span>
          </div>
        </div>
        <button className={cx("pause")} data-pause type="button" aria-pressed="false" aria-label="Tạm dừng"><Pause className={cx("ic pause__p")} aria-hidden /><Play className={cx("ic pause__pl")} aria-hidden /></button>
      </div>
        <div className={cx("frame chatcard")} data-up style={{ "--i": 5 } as CSSProperties} role="img" aria-label="Chat riêng: sinh viên hỏi cách tính điểm học phần, câu trả lời kèm nguồn trong quy chế"><p className={cx("chatcard__q")} aria-hidden="true">Điểm học phần tính thế nào ạ?</p><p aria-hidden="true">Chuyên cần 10 %, giữa kỳ 30 %, cuối kỳ 60 %.</p><p className={cx("chatcard__src")} aria-hidden="true"><ShieldCheck className={cx("ic")} aria-hidden />Nguồn: Quy chế học phần, mục 4</p></div>
      </div>
    </div></div>
  </section>

  <section className={cx("sec flowsec")} id="bai-nop" aria-labelledby="flow-title" data-reveal>
    <div className={`${cx("wrap")} wrap`}>
      <div className={cx("head")}><h2 className={cx("h2")} id="flow-title">Bài nộp đi qua ba bước</h2></div>
      <ol className={cx("steps3")}>
        <li className={cx("step")} style={{ "--i": 0 } as CSSProperties}><span className={cx("mark")} aria-hidden="true">1</span>
          <div className={cx("card")}><h3 className={cx("h3")}>AI chấm nháp theo rubric</h3>
            <div className={cx("spec")} data-ui aria-hidden="true"><p className={cx("k")} style={{ "--k": 0 } as CSSProperties}><span>Xác định đúng lỗ hổng</span><b>8,0</b></p><p className={cx("k")} style={{ "--k": 1 } as CSSProperties}><span>Giải thích cách khai thác</span><b>7,5</b></p><p className={cx("k")} style={{ "--k": 2 } as CSSProperties}><span>Đề xuất cách vá</span><b>9,0</b></p><i className={cx("tag tag--draft k")} style={{ "--k": 3 } as CSSProperties}>Nháp</i></div>
            <p>Mỗi tiêu chí có điểm nháp và đoạn trích từ chính bài làm.</p></div></li>
        <li className={cx("step")} style={{ "--i": 1 } as CSSProperties}><span className={cx("mark")} aria-hidden="true">2</span>
          <div className={cx("card")}><h3 className={cx("h3")}>Giảng viên sửa và duyệt</h3>
            <div className={cx("spec")} data-ui aria-hidden="true"><p><span>Xác định đúng lỗ hổng</span><b>8,0</b></p><p><span>Giải thích cách khai thác</span><b><s>7,5</s><span className={cx("k")} style={{ "--k": 1, display: "inline-block" } as CSSProperties}>8,5</span></b></p><p><span>Đề xuất cách vá</span><b>9,0</b></p><i className={cx("tag tag--done k")} style={{ "--k": 2 } as CSSProperties}>Đã duyệt</i></div>
            <p>Giảng viên sửa từng tiêu chí, duyệt, rồi mới công bố cho lớp.</p></div></li>
        <li className={cx("step")} style={{ "--i": 2 } as CSSProperties}><span className={cx("mark")} aria-hidden="true">3</span>
          <div className={cx("card")}><h3 className={cx("h3")}>Mã tính điểm học phần</h3>
            <div className={cx("spec")} data-ui aria-hidden="true"><p className={cx("k")} style={{ "--k": 0 } as CSSProperties}><span>Chuyên cần · 10 %</span><b>9,0</b></p><p className={cx("k")} style={{ "--k": 1 } as CSSProperties}><span>Giữa kỳ · 30 %</span><b>8,0</b></p><p className={cx("k")} style={{ "--k": 2 } as CSSProperties}><span>Cuối kỳ · 60 %</span><b>8,3</b></p><p className={cx("spec__sum k")} style={{ "--k": 3 } as CSSProperties}><span>Điểm học phần</span><b>8,3</b></p><i className={cx("tag tag--done k")} style={{ "--k": 4 } as CSSProperties}>Mã tính</i></div>
            <p>Công thức trích từ quy chế, giảng viên xác nhận; mỗi điểm mở ra giải trình từng dòng.</p></div></li>
      </ol>
    </div>
  </section>

  <section className={cx("sec storysec")} id="ba-man" aria-labelledby="story-title">
    <div className={`${cx("wrap story")} wrap story`}>
      <div className={cx("head")}><h2 className={cx("h2")} id="story-title">EduPilot trong ba màn</h2></div>
        <ol className={cx("story__list")}>
          <li data-story-item className={cx("story__item")}><h3 className={cx("h3")}>Chat riêng có nguồn</h3><p>Câu trả lời kèm nguồn trong tài liệu của lớp.</p></li>
          <li data-story-item className={cx("story__item")} aria-current="true"><h3 className={cx("h3")}>Duyệt bài AI chấm nháp</h3><p>Bài làm bên trái, điểm từng tiêu chí và đoạn trích bên phải.</p></li>
          <li data-story-item className={cx("story__item")}><h3 className={cx("h3")}>Sổ điểm có giải trình</h3><p>Mỗi điểm học phần mở ra cách tính từng dòng.</p></li>
        </ol>
        <div className={cx("story__shot frame")} data-story-shots>
          <Shot name="story-chat-v1" alt="Chat riêng của sinh viên: câu trả lời kèm nguồn trong tài liệu của lớp" />
          <Shot name="story-review-v1" alt="Màn duyệt bài của giảng viên: bài làm bên trái, điểm từng tiêu chí, đoạn trích và nhận xét nháp bên phải" className={cx("is-on")} />
          <Shot name="story-gradebook-v1" alt="Sổ điểm: điểm học phần mở ra cách tính từng dòng" />
        </div>
    </div>
  </section>

  <section className={cx("sec sodosec")} id="so-do" aria-labelledby="proof-title" data-reveal>
    <div className={`${cx("wrap")} wrap`}>
      <div className={cx("head")}><h2 className={cx("h2")} id="proof-title">Ba số đo trên dữ liệu mô phỏng</h2></div>
      <div className={cx("stats")}>
      <ol className={cx("stats__list")}>
        <li className={cx("stat")} style={{ "--i": 0 } as CSSProperties}><div className={cx("stat__body")}><p className={cx("stat__num")} data-count="146">146</p><h3 className={cx("h3 stat__what")}>yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp</h3><p className={cx("stat__how")}>Bộ phát hiện đạt recall 0,960 trên 200 mẫu tự soạn.</p></div></li>
        <li className={cx("stat")} style={{ "--i": 1 } as CSSProperties}><div className={cx("stat__body")}><p className={cx("stat__num")}>30/30</p><h3 className={cx("h3 stat__what")}>sinh viên mẫu có điểm học phần khớp tính tay</h3><p className={cx("stat__how")}>Điểm do mã tính từ quy chế; AI chỉ trích công thức, giảng viên xác nhận.</p></div></li>
        <li className={cx("stat")} style={{ "--i": 2 } as CSSProperties}><div className={cx("stat__body")}><p className={cx("stat__num")}>+0,05 %</p><h3 className={cx("h3 stat__what")}>Chấm hàng loạt, thời gian đến chữ đầu tiên của chat chỉ tăng 0,05 %</h3><p className={cx("stat__how")}>Đo bằng k6 với nhà cung cấp AI mô phỏng.</p></div></li>
      </ol>
      <div className={cx("stats__shot frame")}><Shot single name="story-gradebook-v1" alt="Sổ điểm của lớp mẫu: điểm thành phần và điểm học phần của từng sinh viên, mỗi điểm có giải trình" /></div>
      </div>
    </div>
  </section>

  <section className={cx("sec vaisec")} id="vai" aria-labelledby="roles-title">
    <div className={`${cx("wrap")} wrap`}>
      <div className={cx("head")}><h2 className={cx("h2")} id="roles-title">Việc của giảng viên và sinh viên</h2>
        <div className={cx("seg")} role="tablist" aria-label="Vai"><button type="button" role="tab" id="t-gv" aria-controls="p-gv" aria-selected="true">Giảng viên</button><button type="button" role="tab" id="t-sv" aria-controls="p-sv" aria-selected="false">Sinh viên</button><span data-ind className={cx("seg__ind")} aria-hidden="true"></span></div></div>
      <div className={cx("grid")}>
        <div className={cx("vai__text")}>
          <div role="tabpanel" id="p-gv" aria-labelledby="t-gv"><h3 className={cx("h3 role__name")}>Giảng viên</h3>
            <dl className={cx("items")}>
              <div><dt>Hôm nay</dt><dd>Việc cần xử lý xếp theo mức khẩn: yêu cầu vào lớp, câu hỏi AI chưa chắc, bài chờ duyệt.</dd></div>
              <div><dt>Chấm bài</dt><dd>AI chấm nháp theo rubric kèm đoạn trích từ bài. Bạn sửa, duyệt, rồi mới công bố.</dd></div>
              <div><dt>Sổ điểm</dt><dd>Công thức trích từ quy chế và do bạn xác nhận. Mỗi điểm có giải trình từng dòng.</dd></div>
              <div><dt>Theo dõi lớp</dt><dd>Điểm danh, phát biểu, thời gian học và cảnh báo sinh viên có nguy cơ.</dd></div>
            </dl></div>
          <div role="tabpanel" id="p-sv" aria-labelledby="t-sv"><h3 className={cx("h3 role__name")}>Sinh viên</h3>
            <dl className={cx("items")}>
              <div><dt>Chat riêng</dt><dd>Hỏi về điểm, chuyên cần, lịch hoặc nội dung bài học. Câu trả lời kèm nguồn trong tài liệu của lớp.</dd></div>
              <div><dt>Nộp bài tập</dt><dd>Nộp tệp ngay trên trang. Điểm và nhận xét chỉ hiện sau khi giảng viên công bố.</dd></div>
              <div><dt>Kết quả của tôi</dt><dd>Điểm học phần có giải trình từng dòng; yêu cầu xem lại điểm khi cần.</dd></div>
              <div><dt>Hỏi đáp công khai</dt><dd>Hỏi cả lớp và đọc câu trả lời đã được giảng viên xác nhận.</dd></div>
            </dl></div>
        </div>
        <div className={cx("vai__shot frame")} data-role-shots>
          <Shot name="giang-vien-hom-nay" alt="Màn Hôm nay của giảng viên: việc cần xử lý xếp theo mức khẩn" className={cx("is-on")} />
          <Shot name="sinh-vien-375-hom-nay-360" alt="Màn Hôm nay của sinh viên trên điện thoại" className={cx("phone")} />
        </div>
      </div>
    
      <div className={cx("actions vai__cta")}><Link className={cx("btn btn--secondary")} href="/register">Tạo tài khoản</Link></div>
    </div>
  </section>
</main>

<footer className={`${cx("sec closing")} closing`} aria-labelledby="cta-title" data-reveal>
  <div className={`${cx("wrap")} wrap`}>
    <div className={cx("close")}>
      <div className={cx("close__text")}>
        <span className={cx("stripe")} aria-hidden="true"><i></i><b></b></span>
        <h2 className={cx("h2")} id="cta-title">Lớp của bạn trên EduPilot</h2>
        <div className={cx("actions")}><Link className={cx("btn btn--primary")} href="/login">Đăng nhập<ArrowRight className={cx("ic arr")} aria-hidden /></Link><Link className={cx("btn btn--secondary")} href="/register">Tạo tài khoản</Link></div>
      </div>
      <div className={cx("close__shot frame")}><Shot name="giang-vien-so-diem" alt="Sổ điểm của lớp: điểm thành phần và điểm học phần của từng sinh viên" /></div>
    </div>
    <div className={cx("legal")}><p>Đồ án tốt nghiệp · Lê Văn Quang Trung · Dữ liệu trên bản demo là dữ liệu mô phỏng.</p><a href="https://github.com/kagamikuro1024/EduPilot-v2" rel="noopener noreferrer">Mã nguồn trên GitHub</a></div>
  </div>
</footer>

<dialog className={cx("dlg")} data-video-dialog aria-label="Video giới thiệu EduPilot">
        <video controls preload="none" playsInline data-poster="/landing/gioi-thieu-v1.webp" width={1280} height={800} aria-describedby="video-loi">
          <source src="/landing/gioi-thieu-v1.mp4" type="video/mp4" />
          <source src="/landing/gioi-thieu-v1.webm" type="video/webm" />
        </video>
        <p id="video-loi" className="ep-sr-only">{VIDEO_CAPTIONS.join(" ")}</p>
        <form method="dialog">
          <a className={cx("dlg__link")} href="/landing/gioi-thieu-v1.mp4" target="_blank" rel="noopener noreferrer">Mở video trong tab mới</a>
          <button className={cx("btn btn--secondary btn--sm")}>Đóng</button>
        </form>
      </dialog>
    </div>
  );
}
