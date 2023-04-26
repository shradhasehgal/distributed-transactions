import sys
import os
directory = sys.argv[1]
# print 'Number of arguments:', len(sys.argv), 'arguments.'
# print 'Argument List:', str(sys.argv)

# for 
# print("FAIL")
for i in range(1,5):
    pathname = os.path.join(directory, f"output{i}.log")
    with open(pathname) as f:
        lines= f.readlines()
        fail = 0
        # print(lines)
        if len(lines) == 0:
            fail = 1
        else:
            last_line = lines[-1]

            if "ABORT" not in last_line and "COMMIT" not in last_line:
                fail = 1
        
        if fail == 1:
            print("FAIL")
            break