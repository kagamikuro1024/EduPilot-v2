"use client";

import { Markdown } from "@/shared/domain";
import { DataTable, StatusText } from "@/shared/ui";
import { VERDICT_VI, vnum, type AnswerShape, type ResultItem } from "./resultsApi";
import s from "./Results.module.css";

const chosen = (a: AnswerShape, id: string) => Boolean(a?.option_ids?.includes(id));

/** Các câu của một kết quả (sinh viên sau công bố / Staff): câu trắc nghiệm có lựa chọn + đúng / sai, câu code có bảng test mẫu, số test ẩn và mã gập. Dùng chung nên hai nơi không lệch nhau. */
export function ResultItems({ items, staff, collapsible }: { items: ResultItem[]; staff?: boolean; /** mặc định gập, mở từng câu (US-PE-08 AC7, màn của sinh viên) */ collapsible?: boolean }) {
  return (
    <ol className={s.items} data-part="result-items">
      {items.map((it) => (
        <li key={it.item_id} className={s.item} data-part="result-item" data-correct={it.correct === null ? undefined : String(it.correct)}>
          {collapsible ? (
            <details className={s.fold}>
              <summary><ItemHead it={it} staff={staff} /></summary>
              <div className={s.foldBody}><ItemBody it={it} /></div>
            </details>
          ) : (
            <>
              <ItemHead it={it} staff={staff} />
              <ItemBody it={it} />
            </>
          )}
        </li>
      ))}
    </ol>
  );
}

function ItemHead({ it, staff }: { it: ResultItem; staff?: boolean }) {
  return (
    <span className={s.itemHead}>
      <span className={s.itemNo}>Câu {it.position}</span>
      <span className={s.itemScore}>{vnum(it.earned)} / {vnum(it.max)} điểm</span>
      {it.type !== "CODE" && it.correct === true && <StatusText tone="green">Đúng</StatusText>}
      {it.type !== "CODE" && it.correct === false && <StatusText tone="red">Sai</StatusText>}
      {it.overridden && <StatusText tone="blue">{staff ? "Đã điều chỉnh" : "Câu này đã được điều chỉnh bởi giảng viên"}</StatusText>}
    </span>
  );
}

function ItemBody({ it }: { it: ResultItem }) {
  return (
    <>
      <Markdown source={it.stem} />
      {it.type === "CODE" ? <CodeResult it={it} /> : <Choice it={it} />}
    </>
  );
}

function Choice({ it }: { it: ResultItem }) {
  if (it.type === "TRUE_FALSE") {
    const mine = it.mine?.value;
    const key = it.answer?.value;
    return (
      <ul className={s.opts}>
        {[true, false].map((v) => (
          <li key={String(v)} className={s.opt} data-mine={mine === v || undefined} data-key={key === v || undefined}>
            <span>{v ? "Đúng" : "Sai"}</span>
            {mine === v && <span className={s.tag}>Bạn chọn</span>}
            {key === v && <span className={s.tag} data-tone="ok">Đáp án đúng</span>}
          </li>
        ))}
        {mine === undefined && <li className={s.muted}>Bạn chưa trả lời câu này.</li>}
      </ul>
    );
  }
  const answered = (it.mine?.option_ids?.length ?? 0) > 0;
  return (
    <>
      <ul className={s.opts}>
        {it.options.map((o) => (
          <li key={o.id} className={s.opt} data-mine={chosen(it.mine, o.id) || undefined} data-key={chosen(it.answer, o.id) || undefined}>
            <span>{o.body}</span>
            {chosen(it.mine, o.id) && <span className={s.tag}>Bạn chọn</span>}
            {chosen(it.answer, o.id) && <span className={s.tag} data-tone="ok">Đáp án đúng</span>}
          </li>
        ))}
      </ul>
      {!answered && <p className={s.muted}>Bạn chưa trả lời câu này.</p>}
      {it.explanation && (
        <div className={s.explain} data-part="explanation">
          <p className={s.label}>Giải thích</p>
          <Markdown source={it.explanation} />
        </div>
      )}
    </>
  );
}

function CodeResult({ it }: { it: ResultItem }) {
  const f = it.final_submission;
  return (
    <div className={s.code} data-part="code-result">
      {!f && <p className={s.muted}>Bạn không nộp lời giải cho câu này.</p>}
      {f && !f.compile_ok && (
        <div>
          <p className={s.label}>Chưa biên dịch được</p>
          {it.compile_log && <pre className={s.pre} tabIndex={0} aria-label="Thông báo biên dịch">{it.compile_log}</pre>}
        </div>
      )}
      {it.samples.length > 0 && (
        <DataTable
          caption="Kết quả test mẫu"
          dense
          mobile="scroll"
          rows={it.samples}
          rowKey={(t) => t.name}
          columns={[
            { key: "name", header: "Test", render: (t) => t.name },
            { key: "v", header: "Kết quả", render: (t) => <StatusText tone={t.verdict === "AC" ? "green" : "red"}>{VERDICT_VI[t.verdict] ?? t.verdict}</StatusText> },
            { key: "t", header: "Thời gian", align: "end", render: (t) => `${t.time_ms} ms` },
            { key: "m", header: "Bộ nhớ", align: "end", render: (t) => `${t.memory_kb} KB` },
          ]}
        />
      )}
      {it.hidden && <p data-part="hidden-count">Test ẩn: đạt {it.hidden.passed} trên {it.hidden.total}</p>}
      {f && (
        <details className={s.src}>
          <summary>Mã của bạn ({f.language})</summary>
          <pre className={s.pre} tabIndex={0} aria-label="Mã của bạn">{f.source}</pre>
        </details>
      )}
    </div>
  );
}

/** Dòng tóm tắt dưới điểm (US-PE-08 AC7): «Bạn đúng 8/10 câu trắc nghiệm · bài lập trình: 3/4 test ẩn đạt»; phần nào không có câu thì bỏ. */
export function resultSummary(items: ResultItem[]): string {
  const mcq = items.filter((i) => i.type !== "CODE");
  const hidden = items.flatMap((i) => (i.type === "CODE" && i.hidden ? [i.hidden] : []));
  const parts = [
    ...(mcq.length > 0 ? [`Bạn đúng ${mcq.filter((i) => i.correct === true).length}/${mcq.length} câu trắc nghiệm`] : []),
    ...(hidden.length > 0 ? [`bài lập trình: ${hidden.reduce((n, h) => n + h.passed, 0)}/${hidden.reduce((n, h) => n + h.total, 0)} test ẩn đạt`] : []),
  ];
  return parts.join(" · ");
}
