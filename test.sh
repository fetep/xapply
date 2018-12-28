#!/usr/bin/env bash

set -euo pipefail

basedir=$(dirname $0)
xapply=$(readlink -f "$basedir/xapply")
fail=0
pass=0
five=". . . . ."

tmpd=$(mktemp -d)
cd $tmpd
trap "rm -rf $tmpd" EXIT

### xapply -v
$xapply -v 'echo %1' $five >cmds
lines=$(wc -l <cmds)
if [[ $lines == "10" ]]; then
  pass=$((pass+1))
else
  echo "FAIL: xapply -v output (expected 10 lines, got $lines)"
  fail=$((fail+1))
fi

### xapply -x
$xapply -x 'echo %1' $five >/dev/null 2>cmds
lines=$(wc -l <cmds)
if [[ $lines == "5" ]]; then
  pass=$((pass+1))
else
  echo "FAIL: xapply -x output (expected 5 lines, got $lines)"
  fail=$((fail+1))
fi

### xapply -n
$xapply -n 'echo %1' $five >out
lines=$(wc -l <out)
if [[ $lines == "5" ]]; then
  pass=$((pass+1))
else
  echo "FAIL: xapply -n noop mode (expected 5 lines, got $lines)"
  fail=$((fail+1))
fi

### xapply -S should run "shell -c command"
out=$($xapply -S/bin/echo '%1' .)
if [[ $out == "-c ." ]]; then
  pass=$((pass+1))
else
  echo "FAIL: xapply -S shell mode (expected '-c .' , got '$out')"
  fail=$((fail+1))
fi

### xapply -P basic testing
start=$(date +%s)
$xapply -P5 'sleep 1 #' $five $five
duration=$(($(date +%s)-start))
if [[ $duration == "2" ]]; then
  pass=$((pass+1))
else
  echo "FAIL: xapply -P not executing in parallel (expected 2s sleep, got ${duration}s sleep)"
fi

echo "Test Summary: Pass=$pass Fail=$fail"
if [[ $fail > 0 ]]; then
  exit 1
fi
exit 0
