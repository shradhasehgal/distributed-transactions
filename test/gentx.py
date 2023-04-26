import random
from collections import defaultdict
import sys
from string import ascii_lowercase
from time import sleep

# There will be 26**ACCOUNT_LEN accounts
NO_SERVERS = 5

# probability that a transaction is a deposit
# initially more transaction will be deposits since transfers from non-existent accounts will not be placed
DEP_PROB = 0.6
WITHDRAW_PROB = 0.8
BALANCE_PROB = 1.0


def random_account(): 
    accounts = ["koo", "boo", "goo", "too", "soo", "woo", "zoo", "moo", "noo", "boo"]
    servers = ["A", "B", "C", "D", "E"]
    servers = servers[:2]
    accounts = accounts[:5]
    account = accounts[random.randint(0, len(accounts)-1)]
    server = servers[random.randint(0, len(servers)-1)]
    return f'{server}.{account}'

print(f"BEGIN")
while True:

    txn_execute = random.random() 
    if txn_execute < 0.05:
        commit_or_abort = random.random() 
        if commit_or_abort < 0.8:
            print(f"COMMIT")
        else:
            print("ABORT")
        break        
    else:
        no = random.random() 
        account = random_account()
        if no < DEP_PROB:
            amount = random.randrange(1,101)
            print(f"DEPOSIT {account} {amount}")
        elif no < WITHDRAW_PROB:
            amount = random.randrange(1,101)
            print(f"WITHDRAW {account} {amount}")
        else:
            print(f"BALANCE {account}")

