import { API_BASE } from "./endpoint";
import { EARLY_REFRESH_KEY } from "./authSession";
import { COURSE_STORAGE_KEY } from "../session/cookies";

/**
 * Mã nội tuyến sau EARLY_REFRESH_SCRIPT (DEF LCP-2, US-P10-08 AC6): `/assignments/<id>` có phần tử LCP là CHỮ DỮ LIỆU (đề bài),
 * đến sau chuỗi tải JS → hydrate → gọi API. Ngay khi lần làm mới sớm trả token, mã này gọi đúng một GET của màn (lớp lấy từ `?course=`, không có thì lớp đã nhớ) và
 * điền chữ thô vào ô `[data-part=early-body]` của PreShell: người dùng đọc được đề trước khi JS xong. React hydrate PreShell bỏ qua ô này (suppressHydrationWarning);
 * khung thật thay PreShell và tự gọi API như cũ (một GET thừa, không chia sẻ dữ liệu). Token chỉ nằm trong bộ nhớ của hàm này; bản `clone()` để không ăn thân của lời gọi React dùng.
 */
export const EARLY_PAINT_SCRIPT = `try{var P=window[${JSON.stringify(EARLY_REFRESH_KEY)}],m=/^\\/(assignments)\\/([^/]+)$/.exec(location.pathname);if(P&&m&&m[2]!=="history"){var c=new URLSearchParams(location.search).get("course")||localStorage.getItem(${JSON.stringify(COURSE_STORAGE_KEY)}),a=m[1]==="assignments";if(c&&/^[\\w-]{1,64}$/.test(c)&&/^[\\w-]{1,64}$/.test(m[2])){P.then(function(r){return r.ok?r.clone().json():null}).then(function(b){return b&&fetch(${JSON.stringify(API_BASE)}+"/courses/"+c+(a?"/assignments/"+m[2]+"/mine":"/practice/attempts/"+m[2]),{headers:{Accept:"application/json",Authorization:"Bearer "+b.access_token},credentials:"include",signal:AbortSignal.timeout(15000)})}).then(function(r){return r&&r.ok?r.json():null}).then(function(d){var t=d&&(a?d.assignment.description:d.items[d.current_index||0].question.stem);if(typeof t!=="string"||!t)return;var go=function(){var e=document.querySelector("[data-part=early-body]");if(e){e.textContent=t.replace(/[*\\x60]/g,"");e.parentElement.parentElement.hidden=false}};document.readyState==="loading"?document.addEventListener("DOMContentLoaded",go):go()}).catch(function(){})}}}catch(e){}`;
