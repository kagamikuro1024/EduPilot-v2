#!/bin/bash
# s4n-rec.sh TC P|F "ghi chú"  → ghi vào results-smk34/results-s4n.tsv
printf '%s\t%s\t%s\n' "$1" "$2" "$(echo "$3" | tr '\t\n' '  ' | cut -c1-900)" >> ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/results-smk34/results-s4n.tsv
