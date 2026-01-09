# Implementation (Million Operations Per Second)
Implementation	SPSC	MPSC	MPMC	SPMC
Buffered Channel	15-20M	5-8M	2-4M	8-12M
Lock-based Ring Buffer	8-12M	3-5M	1-2M	4-6M
Lock-free (CAS)	25-35M	10-15M	5-8M	12-18M
Lock-free (Sequence)	30-40M	12-18M	6-10M	15-22M

- CAS: should use the pointer of struct for the not plain type
- how to deal with the buffer is full, wait or return immediately or overwrite
- the best way is use Sequence way to implement all things like SPSC, MPSC, MPMC, SPMC 