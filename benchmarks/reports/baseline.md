# Số đo nền (MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB)

| Container | CPU nghỉ % | RAM MiB |
| --- | ---: | ---: |
| ep-t1-worker-1 | 4.12 | 14.6 |
| ep-t1-gateway-1 | 3.3 | 17.1 |
| ep-t1-mailpit-1 | 0 | 6.7 |
| ep-t1-redis-1 | 0.92 | 20.8 |
| ep-t1-minio-1 | 0 | 81.7 |
| ep-t1-postgres-1 | 0.59 | 329.6 |
| ep-t1-caddy-1 | 0.03 | 20.7 |
| ep-t1-pgbouncer-1 | 2.26 | 7.8 |
| ep-t1-mock-graph-1 | 0 | 3.9 |
| ep-t1-mock-imap-1 | 0 | 4.7 |
| ep-t1-judge-1 | 0.06 | 10.1 |
| ep-t1-frontend-1 | 1.84 | 52.5 |
| **Tổng** | | **570.2** |

| Image | MB |
| --- | ---: |
| gateway | 17.7 |
| worker | 16.7 |
| frontend | 96.2 |

Khởi động lạnh (up → readyz 200): 18.1 / 18.1 / 18.9 s; trung vị 18.1 s.

| Endpoint | p50 ms | p95 ms | p99 ms | Đạt |
| --- | ---: | ---: | ---: | --- |
| sv_me_attendance | 2.5 | 4.4 | 5.4 | PASS |
| gv_course_today | 7.6 | 9.8 | 11.3 | PASS |
| w_thread | 316.8 | 327.5 | 330.9 | PASS |
| w_heartbeat | 1.1 | 1.6 | 1.8 | PASS |
| sv_threads | 3.5 | 5.2 | 7.4 | PASS |
| w_chat_session | 3.0 | 4.5 | 6.9 | PASS |
| gv_students | 7.7 | 9.4 | 10.4 | PASS |
| gv_gradebook | 8.6 | 10.7 | 14.1 | PASS |
| sv_me_today | 1.7 | 2.8 | 4.1 | PASS |
| sv_me_grades | 5.8 | 7.8 | 9.2 | PASS |
| gv_profile_360 | 4.4 | 5.7 | 9.4 | PASS |
| gv_me_today | 8.4 | 11.0 | 13.8 | PASS |
| sv_library | 2.1 | 3.4 | 4.8 | PASS |
