#!/bin/bash

code_folder='src/'
curr_folder=$(pwd)'/'
servers=(A B C D E)
ports=(10001 10002 10003 10004 10005)

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

run_servers(){
  for server in ${servers[@]}; do
    ./server $server config.txt > ../server_${server}.log 2>&1 &
  done
}




#!/bin/bash

# Define the array
my_array=("1" "2" "3" "4")
# my_array=("1" "2")


# Define a function to generate permutations
# This function uses recursion to generate all permutations
generate_permutations() {
  local items=("$@")
  local i

  # Base case: if the array has only one element, return it as a permutation
  if [ ${#items[@]} -eq 1 ]; then
    echo "${items[0]}"
    return
  fi

  # Generate permutations by swapping each element with the first element and
  # recursively generating permutations for the remaining elements
  for ((i = 0; i < ${#items[@]}; i++)); do
    local first="${items[$i]}"
    local rest=("${items[@]:0:i}" "${items[@]:i+1}")
    for perm in $(generate_permutations "${rest[@]}"); do
      echo "$first$perm"
    done
  done
}

# Generate the permutations and store them in an array
permutations=($(generate_permutations "${my_array[@]}"))

# Print the permutations
echo "Permutations:"
count=0
to_remove=$(pwd)'/testcase'
echo $to_remove
rm -rf $to_remove

mkdir testcase
for perm in "${permutations[@]}"; do
    mkdir testcase/$count
    out_folder=$(pwd)'/testcase/'${count}'/'
    run_servers
    for ((i = 0; i < ${#perm}; i++)); do
        test_no="${perm:$i:1}"
        gtimeout -s SIGTERM 5s ./client ${permutations[$i]} config.txt < ${curr_folder}input${test_no}.txt > ${out_folder}output${test_no}.log 2>&1
    done
    shutdown
    ((count++))

    echo "done"
done

# cd $curr_folder
# echo "Difference between your output and expected output:"
# diff output1.log expected1.txt
# diff output2.log expected2.txt
