// P-11-46: cpuSlowdownMultiplier của lượt Lighthouse simulate theo tốc độ MÁY CHẠY. Cố định 4 làm TBT tăng 2–3× khi runner chậm (benchmarkIndex ≈ 2500 so với ≈ 4200: CI 43743335571).
// multiplier = 4 × 4200 / benchmarkIndex, kẹp [2, 8]; benchmarkIndex đo bằng chính hàm của Lighthouse trong cùng Chromium sẽ chạy lhci. In ra stdout: một số.

// Bản sao hàm computeBenchmarkIndex của Lighthouse (core/lib/page-functions.js; chạy trong trang): trung bình của hai phép đo, mỗi phép 0,5 s — GC (nối chuỗi) và sao chép mảng.
function computeBenchmarkIndex() {
  function benchmarkIndexGC() {
    const start = Date.now();
    let iterations = 0;
    while (Date.now() - start < 500) {
      let s = "";
      for (let j = 0; j < 10000; j++) s += "a";
      if (s.length === 1) throw new Error("never");
      iterations++;
    }
    return Math.round(iterations / 10 / ((Date.now() - start) / 1000));
  }
  function benchmarkIndexNoGC() {
    const arrA = [];
    const arrB = [];
    for (let i = 0; i < 100000; i++) arrA[i] = arrB[i] = i;
    const start = Date.now();
    let iterations = 0;
    while (iterations % 10 !== 0 || Date.now() - start < 500) {
      const src = iterations % 2 === 0 ? arrA : arrB;
      const tgt = iterations % 2 === 0 ? arrB : arrA;
      for (let j = 0; j < src.length; j++) tgt[j] = src[j];
      iterations++;
    }
    return Math.round(iterations / 10 / ((Date.now() - start) / 1000));
  }
  return (benchmarkIndexGC() + benchmarkIndexNoGC()) / 2;
}

const REF = 4200; // benchmarkIndex của runner «bình thường» (các lượt xanh 4150–4450)
(async () => {
  const { chromium } = await import("@playwright/test");
  const b = await chromium.launch({ args: ["--no-sandbox"] });
  const p = await b.newPage();
  const samples = [];
  for (let i = 0; i < 3; i++) samples.push(await p.evaluate(`(${computeBenchmarkIndex.toString()})()`)); // 3 × ~1 s; lấy trung vị cho đỡ nhiễu
  await b.close();
  samples.sort((x, y) => x - y);
  const bi = samples[1];
  const m = Math.min(8, Math.max(2, Math.round(((4 * REF) / bi) * 100) / 100));
  console.error(`benchmarkIndex ${JSON.stringify(samples.map(Math.round))} → trung vị ${Math.round(bi)}; cpuSlowdownMultiplier = ${m} (4 × ${REF} / ${Math.round(bi)}, kẹp [2, 8])`);
  console.log(m);
})();
