#!/bin/bash
cd ~/Documents/TA_Agent_qc2 || exit 1
git fetch -q origin qc2/regression
git merge-base --is-ancestor FETCH_HEAD HEAD || git merge --no-edit FETCH_HEAD 2>&1 | tail -2
git push -q origin HEAD:refs/heads/qc2/regression 2>&1 | tail -2
git rev-parse --short HEAD
