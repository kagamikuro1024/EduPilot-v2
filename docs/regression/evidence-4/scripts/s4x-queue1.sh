#!/bin/bash
# S4X: hàng đợi lượt Playwright sprint 4 N còn lại (mỗi lượt qua khoá /tmp/qc2-pw.lock trong s4n-pw.sh)
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4
bash scripts/s4n-pw.sh s4x1 desktop smkt3-nav.spec.ts smkt3-journey.spec.ts smkt3-bell.spec.ts
bash scripts/s4n-pw.sh s4x3 desktop s4n-caddy.spec.ts
bash scripts/s4n-pw.sh s4x2 desktop smkt3-p20231.spec.ts
