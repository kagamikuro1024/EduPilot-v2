#!/bin/bash
# res.sh TC KQ "ghi chú"  -> thêm vào /tmp/qc2-r95-results.tsv (dòng sau thắng) và bản sao evidence
printf '%s\t%s\t%s\n' "$1" "$2" "$3" >> /tmp/qc2-r95-results.tsv
