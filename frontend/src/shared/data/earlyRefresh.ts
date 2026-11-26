import { EARLY_REFRESH_KEY } from "./authSession";
import { API_BASE } from "./endpoint";
import { RETURNING_KEY } from "../session/returning";

/**
 * Mã nội tuyến cho `<head>` của root layout (U1, US-P10-08): bắt đầu `POST /auth/refresh` ngay khi phân tích HTML, song song với tải JS.
 * Cùng điều kiện với AuthProvider (`/` chưa từng đăng nhập thì không gọi) và cùng khoá `ep-refresh` giữa các tab; `authSession.doRefresh` lấy lại đúng MỘT lần rồi xoá.
 */
export const EARLY_REFRESH_SCRIPT = `try{if(location.pathname!=="/"||localStorage.getItem(${JSON.stringify(RETURNING_KEY)})==="1"){var f=function(){return fetch(${JSON.stringify(`${API_BASE}/auth/refresh`)},{method:"POST",credentials:"include",headers:{Accept:"application/json","Content-Type":"application/json"},signal:(function(){try{return AbortSignal.timeout(15000)}catch(e){var c=new AbortController();setTimeout(function(){c.abort()},15000);return c.signal}})()})};window[${JSON.stringify(EARLY_REFRESH_KEY)}]=navigator.locks?navigator.locks.request("ep-refresh",f):f()}}catch(e){}`;
