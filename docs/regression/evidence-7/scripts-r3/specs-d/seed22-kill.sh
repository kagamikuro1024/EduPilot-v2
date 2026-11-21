#!/bin/bash
# SEED-22: chạy seed trên DB trống, kill -9 tiến trình node ngay khi dòng log đầu tiên của bước 14 (PUT điểm danh buổi 1) hiện ra
cd ~/Documents/TA_Agent_qc3/docs/regression/evidence-7
L=r5/seed22-run1.log; : > $L
./scripts-r3/specs-d/seed-run.sh --verbose > $L 2>&1 &
PID=$!; echo "pid=$PID" > r5/seed22-kill.log
tail -n0 -f $L | awk '/^\[14\/20\]/{f=1} f && /\/attendance → /{print; fflush(); system("pkill -9 -f \"node scripts/seed.mjs\""); exit}' > /dev/null
echo "$(date +%T) kill -9 ngay khi dòng PUT điểm danh đầu tiên của bước 14 hiện ra; bước cuối trong log: $(grep '^\[' $L | tail -1)" >> r5/seed22-kill.log
sleep 1; echo "final attendance_records=$(docker exec ep-r-postgres-1 psql -U edupilot -d qc3seed -At -c 'select count(*) from attendance_records'); sessions_done=$(docker exec ep-r-postgres-1 psql -U edupilot -d qc3seed -At -c 'select count(*) from class_sessions where attendance_done_at is not null')" >> r5/seed22-kill.log
