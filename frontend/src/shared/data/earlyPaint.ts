import { API_BASE } from "./endpoint";
import { EARLY_GET_KEY, EARLY_REFRESH_KEY } from "./authSession";
import { earlyText } from "./earlyText";

/**
 * Mã nội tuyến sau EARLY_REFRESH_SCRIPT (LCP-2, US-P10-08 AC6): `/assignments/<id>` và `/practice/<attemptId>` có phần tử LCP là CHỮ DỮ LIỆU (đề bài / câu hỏi),
 * đến sau chuỗi tải JS → hydrate → gọi API. Ngay khi lần làm mới sớm trả token, mã này gọi đúng một GET của màn (lớp CHỈ lấy từ `?course=`: đoán lớp đã nhớ sẽ là
 * một request thừa khi sai, trái luật «không dò lớp») và điền chữ thô vào ô `[data-part=early-body]` của PreShell: người dùng đọc được đề trước khi JS xong.
 * React hydrate PreShell bỏ qua ô này (suppressHydrationWarning). Token chỉ nằm trong bộ nhớ của hàm này; `clone()` để không ăn thân của lời gọi React dùng.
 * `/assignments`: GET chỉ đọc thuần → lời hứa kết quả đặt ở `window[EARLY_GET_KEY]`, `apiClient` dùng lại đúng một lần (trang chỉ còn MỘT GET).
 * `/practice`: KHÔNG chia sẻ — GET của khung thật mang `X-Practice-Tab` (tiếp quản tab, chặn hai tab thi) và hiệu chỉnh đồng hồ thi bằng thời điểm gửi / nhận của chính nó.
 */
export const EARLY_PAINT_SCRIPT = `try{var P=window[${JSON.stringify(EARLY_REFRESH_KEY)}],m=/^\\/(assignments|practice)\\/([^/]+)$/.exec(location.pathname);if(P&&m&&m[2]!=="history"){var c=new URLSearchParams(location.search).get("course"),a=m[1]==="assignments";if(c&&/^[\\w-]{1,64}$/.test(c)&&/^[\\w-]{1,64}$/.test(m[2])){var T=${earlyText.toString()},u=${JSON.stringify(API_BASE)}+"/courses/"+c+(a?"/assignments/"+m[2]+"/mine":"/practice/attempts/"+m[2]),j=P.then(function(r){return r.ok?r.clone().json():null}).then(function(b){return b&&fetch(u,{headers:{Accept:"application/json",Authorization:"Bearer "+b.access_token},credentials:"include",signal:AbortSignal.timeout(15000)})}).then(function(r){return r&&r.ok?r.json():null});if(a)window[${JSON.stringify(EARLY_GET_KEY)}]={u:u,p:j.catch(function(){return null})};j.then(function(d){var t=d&&(a?d.assignment.description:d.items[d.current_index||0].question.stem);if(typeof t!=="string"||!t)return;var go=function(){var e=document.querySelector("[data-part=early-body]");if(e){e.textContent=T(t);e.parentElement.parentElement.hidden=false}};document.readyState==="loading"?document.addEventListener("DOMContentLoaded",go):go()}).catch(function(){})}}}catch(e){}`;
