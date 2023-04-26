#!/bin/bash

code_folder='src/'
curr_folder=$(pwd)'/'
servers=(A B C D E)
ports=(10001 10002 10003 10004 10005)

perm="1234"
len=4
num=24

# shutdown servers
shutdown() {
	for port in ${ports[@]}; do
		kill -15 $(lsof -ti:$port)
	done
}
# trap shutdown EXIT

# initialize servers
cp config.txt ${code_folder}/config.txt
cd $code_folder
for server in ${servers[@]}; do
	./server $server config.txt > ../server_${server}.log 2>&1 &
done

sleep 5

out_folder=$(pwd)'/testcase/concurrent/'
mkdir $out_folder
for ((i = 0; i < ${#perm}; i++)); do
    test_no="${perm:$i:1}"
    echo ${test_no}
    # cmd="gtimeout -s SIGTERM 5s ./client ${test_no} config.txt < ${curr_folder}input${test_no}.txt > ${out_folder}output${test_no}.log 2>&1 &"
    # echo $cmd
    gtimeout -s SIGTERM 5s ./client ${test_no} config.txt < ${curr_folder}input${test_no}.txt > ${out_folder}output${test_no}.log 2>&1 &
    # $cmd
done

sleep 5

dir1=$out_folder
found=0
for ((i = 0; i < num; i++)); do
    dir2=$(pwd)'/testcase/'$i'/'
  # Compare the contents of the two directories using the 'diff' command
    diff_output=$(diff -rq "$dir1" "$dir2")

    # Check if the diff output is empty (i.e. the two directories are the same)
    if [ -z "$diff_output" ]; then
      found=1
      # echo "The two directories are the same"
  fi
done

if [ "$found" -eq 1 ]; then
  echo "Same output found"
else
  timestamp=$(date +%s)
  failed_logs_folder=$(pwd)'/failed/'$timestamp'/'
  echo "Creating folder $failed_logs_folder"
  mkdir -p $failed_logs_folder
  for server in ${servers[@]}; do
	  cp ../server_${server}.log $failed_logs_folder
  done 
  cp ${out_folder}output*.log $failed_logs_folder
  for ((i = 0; i < ${#perm}; i++)); do
    test_no="${perm:$i:1}"
    cp ${curr_folder}input${test_no}.txt $failed_logs_folder
  done
  echo "Did not match any output"
fi

shutdown
sleep 2

# cd $curr_folder
# echo "Difference between your output and expected output:"
# diff output1.log expected1.txt
# diff output2.log expected2.txt
