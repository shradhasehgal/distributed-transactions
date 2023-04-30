## Design Document

### Group Members

* Shradha Sehgal, ssehgal4@illinois.edu
* Chaitanya Bhandari, cbb1996@illinois.edu

### GitLab details

### Virtual Machine details

Cluster number: 28

Exact VM details:

* sp23-cs425-2801.cs.illinois.edu
* sp23-cs425-2802.cs.illinois.edu
* sp23-cs425-2803.cs.illinois.edu
* sp23-cs425-2804.cs.illinois.edu
* sp23-cs425-2805.cs.illinois.edu
* sp23-cs425-2806.cs.illinois.edu
* sp23-cs425-2807.cs.illinois.edu
* sp23-cs425-2808.cs.illinois.edu
* sp23-cs425-2809.cs.illinois.edu
* sp23-cs425-2810.cs.illinois.edu

### Instructions for running our code

#### Download Golang

Execute the following commands:
* `wget https://go.dev/dl/go1.19.5.linux-amd64.tar.gz`
* `rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.5.linux-amd64.tar.gz`
* `export PATH=$PATH:/usr/local/go/bin`

#### Clone repository

* `git clone git@gitlab.engr.illinois.edu:cbb1996/cs425.git`

#### Browse to MP3 code

```cd cs425/mp3```

#### Build and run

In the head, run

```
go mod tidy
```

Running the server

``` 
cd server 
go build
./server <server_name> config.txt
```

Running the client
``` 
cd client 
go build
./client <client_name> config.txt
```

where `config.txt` is the configuration file containing the branch, hostname, and the port no. of a server.

### Concurrent control approach

#### Timestamp ordering

We use timestamped ordering as our concurrency control approach for distributed transactions. This is an optimistic concurrency control policy where we assign each transaction an id, which determines its position in serialization order.
Next, we impose conditions on the read and write transactions on any object O by checking that the transactions that accessed it had lower IDs. 

#### Deadlock prevention 

Since we use an optimistic concurrency control policy of timestamped ordering, we do not run into deadlocks. This happens because we maintain an order of transactions by assigning them IDs. 

Following this serialization order, we ensure that T’s write to objects are allowed only if transactions that have read or written O have lower ids than T and T's read to object O is allowed only if O was last written by a transaction with a lower id than T.

If any rule is violated, we abort the transaction, thereby avoiding deadlock altogether as older transactions never wait on newer ones.

#### Data structures

We implement the above timestamped ordering by maintaining per-object states. For each object, we store its committed value, the timestamp that wrote that committed ID,  list of read (RTS), and tentative write timestamps (TW). We use locks around the object state, so they are updated one at a time and transactions affecting different objects can still execute in concurrent manner.

#### Commits, Aborts, and Rollback

If any of the aforementioned rules are violated, we abort that entire transaction. We do this by performing a rollback of the transaction - we remove that transaction from the read timestamps and tentative writes list. 

Since we roll back the transaction (i.e. remove the intermediary steps that it did from the read timestamps and tentative writes list), partial changes do not take place. If the transaction commits, all its changes take place and if it is aborted, none of its changes take place.

We also update the transaction state to aborted so that any transactions that were waiting on the result of this transaction can then proceed (note: other transactions wait on another one by polling on it at regular intervals). 

During commit, a similar procedure is followed where we update the transaction state to committed. We do not remove the tentative writes or read timestamps related to this transaction, and instead update the committed value and timestamp for all the objects involved in that transaction. 

#### Further Reading

The exact working of the timestamped ordering algorithm is shown in the [slides](https://courses.grainger.illinois.edu/ece428/sp2023//assets/slides/lect19-after.pdf).
