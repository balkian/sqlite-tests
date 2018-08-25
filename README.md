This is a quick comparison of three different approaches to load a big CSV/TSV file into a sqlite database.
As input, I used a sample of ~125MB from the [Twitter Social Graph 2009 dataset](https://old.datahub.io/dataset/twitter-social-graph-www2010).

TL;DR sqlite has an `.import` function that works wonderfully.

# Results

## Python

```
 ❯ time python2 convert.py sample-twitter_rv.net        
Done: 9999999 lines. 131128444 / 131128444 bytes (100.0 %)
python2 convert.py sample-twitter_rv.net  63.96s user 27.51s system 63% cpu 2:23.53 total
```

This is the output of cProfile:
```
python2 -m cProfile convert.py sample-twitter_rv.net 
Done: 9999999 lines. 131128444 / 131128444 bytes (100.0 %)
         50006080 function calls (50006073 primitive calls) in 241.581 seconds

   Ordered by: cumulative time

   ncalls  tottime  percall  cumtime  percall filename:lineno(function)
        1    0.000    0.000  241.581  241.581 convert.py:1(<module>)
        1   12.686   12.686  241.576  241.576 convert.py:18(main)
 10000000    5.114    0.000  157.256    0.000 convert.py:7(addusers)
 10000004  152.164    0.000  152.164    0.000 {method 'execute' of 'sqlite3.Connection' objects}
     1001   66.972    0.067   66.972    0.067 {method 'commit' of 'sqlite3.Connection' objects}
 10000002    2.471    0.000    2.471    0.000 {method 'split' of 'str' objects}
 10000000    1.103    0.000    1.103    0.000 {method 'strip' of 'str' objects}
 10000000    1.015    0.000    1.015    0.000 {len}
     1001    0.032    0.000    0.046    0.000 convert.py:12(update_progress)
     1001    0.011    0.000    0.011    0.000 {method 'format' of 'str' objects}
     1000    0.005    0.000    0.005    0.000 {method 'tell' of 'file' objects}
        1    0.000    0.000    0.005    0.005 __init__.py:24(<module>)
        1    0.002    0.002    0.004    0.004 dbapi2.py:24(<module>)
     2002    0.003    0.000    0.003    0.000 {method 'write' of 'file' objects}
        1    0.001    0.001    0.002    0.002 collections.py:11(<module>)
```

## Golang


### With Transactions

```
 ❯ time ./twitter-sqlite sample-twitter_rv.net  
 131128444/131128444  Bytes (100.000%) 10000000 lines - 0.00 Bps (avg. 1656955.51 Bps)
./sqlite-twitter sample-twitter_rv.net  71.16s user 24.15s system 120% cpu 1:19.15 total
```

Using the pprof module, I could also extract some profiling information:

```
 ❯ go tool pprof sqlite-twitter tx
File: sqlite-twitter
Build ID: 708e90eba7948cb0851dfbf3bb6170ccaa418eff
Type: cpu
Time: Aug 24, 2018 at 4:26pm (CEST)
Duration: 1.26mins, Total samples = 1.47mins (117.03%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) cum
(pprof) top20
Showing nodes accounting for 65.26s, 73.82% of 88.40s total
Dropped 222 nodes (cum <= 0.44s)
Showing top 20 nodes out of 70
      flat  flat%   sum%        cum   cum%
     0.09s   0.1%   0.1%     58.96s 66.70%  main.main
     0.44s   0.5%   0.6%     53.39s 60.40%  database/sql.(*Tx).StmtContext
     0.11s  0.12%  0.72%     48.59s 54.97%  database/sql.asString
     0.06s 0.068%  0.79%     47.55s 53.79%  github.com/mattn/go-sqlite3.(*SQLiteConn).Prepare
    43.32s 49.00% 49.80%     44.27s 50.08%  runtime.c128hash
     0.02s 0.023% 49.82%     40.16s 45.43%  net/url.Values.Encode
     0.02s 0.023% 49.84%     39.79s 45.01%  github.com/mattn/go-sqlite3.(*SQLiteConn).lastError.func3
     0.24s  0.27% 50.11%     23.36s 26.43%  runtime.findrunnable
     0.03s 0.034% 50.15%     23.24s 26.29%  runtime.casgstatus.func3
     0.44s   0.5% 50.64%     16.84s 19.05%  runtime.forEachP
     0.17s  0.19% 50.84%     15.17s 17.16%  internal/poll.runtime_pollSetDeadline
    14.98s 16.95% 67.78%     14.98s 16.95%  runtime.duffcopy
     0.41s  0.46% 68.25%     10.67s 12.07%  runtime.startm
     0.17s  0.19% 68.44%      8.62s  9.75%  runtime.panicdottypeI
     0.11s  0.12% 68.56%      7.25s  8.20%  runtime.needm
     0.31s  0.35% 68.91%      6.97s  7.88%  runtime.panicdottypeE
     0.33s  0.37% 69.29%      6.90s  7.81%  github.com/mattn/go-sqlite3.(*SQLiteDriver).Open
     1.81s  2.05% 71.33%      6.38s  7.22%  runtime.newm1
     0.09s   0.1% 71.44%      4.75s  5.37%  runtime.startlockedm
     2.11s  2.39% 73.82%      4.66s  5.27%  runtime.schedtrace

```


### Raw statements and fmt.Sprintf

Plain awful.
I didn't even complete the test.
I suspect `Exec()` starts a new transaction, even though some sources claim it does not if you use a single string (e.g. with `fmt.Sprint`).

### Just one transaction

```
 ❯ time ./twitter-sqlite sample-twitter_rv.net  
./sqlite-twitter sample-twitter_rv.net  67.94s user 20.34s system 129% cpu 1:08.10 total
```

```
 ❯ go tool pprof sqlite-twitter tx
File: sqlite-twitter
Build ID: 7ec752e835de12b94418fffb45515e1b0f89e89f
Type: cpu
Time: Aug 24, 2018 at 4:57pm (CEST)
Duration: 1.25mins, Total samples = 1.46mins (117.52%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) cum
(pprof) top20
Showing nodes accounting for 64.61s, 73.59% of 87.80s total
Dropped 207 nodes (cum <= 0.44s)
Showing top 20 nodes out of 70
      flat  flat%   sum%        cum   cum%
     0.08s 0.091% 0.091%     55.82s 63.58%  main.updateStatus
     0.49s  0.56%  0.65%     51.68s 58.86%  database/sql.(*Tx).StmtContext
     0.12s  0.14%  0.79%     46.56s 53.03%  database/sql.asString
     0.11s  0.13%  0.91%     45.52s 51.85%  github.com/mattn/go-sqlite3.(*SQLiteConn).Prepare
    41.12s 46.83% 47.74%     41.97s 47.80%  runtime.c128hash
     0.01s 0.011% 47.76%     38.64s 44.01%  github.com/mattn/go-sqlite3.(*SQLiteConn).lastError.func3
     0.05s 0.057% 47.81%     38.64s 44.01%  net/url.Values.Encode
     0.38s  0.43% 48.25%     25.40s 28.93%  runtime.findrunnable
     0.04s 0.046% 48.29%     25.25s 28.76%  runtime.casgstatus.func3
     0.51s  0.58% 48.87%     17.72s 20.18%  runtime.forEachP
     0.30s  0.34% 49.21%     15.93s 18.14%  internal/poll.runtime_pollSetDeadline
    15.61s 17.78% 66.99%     15.61s 17.78%  runtime.duffcopy
     0.46s  0.52% 67.52%     11.50s 13.10%  runtime.startm
     0.15s  0.17% 67.69%      9.12s 10.39%  runtime.panicdottypeI
     1.71s  1.95% 69.64%      7.39s  8.42%  runtime.newm1
     0.06s 0.068% 69.70%      7.34s  8.36%  runtime.needm
     0.36s  0.41% 70.11%      7.28s  8.29%  runtime.panicdottypeE
     0.45s  0.51% 70.63%      5.93s  6.75%  github.com/mattn/go-sqlite3.(*SQLiteDriver).Open
     2.48s  2.82% 73.45%      5.72s  6.51%  runtime.schedtrace
     0.12s  0.14% 73.59%      4.77s  5.43%  runtime.startlockedm
```
## CLI

### Indexing first

```
❯ time sh sqlite.sh sample-twitter_rv.net
sh sqlite.sh sample-twitter_rv.net  25.18s user 6.67s system 91% cpu 34.900 total
```

### Indexing afterwards

```
❯ time sh sqlite.sh sample-twitter_rv.net
sh sqlite.sh sample-twitter_rv.net  14.91s user 1.30s system 84% cpu 19.279 total
```

# Comments

There are way too many knobs to fiddle with, and I know very little about sqlite or SQL in general.
This is a very specific use-case, and I've tried to tune the settings accordingly.

Python was the easiest one to try.
It is the language I'm more familiar with, and sqlite3 is included in the standard library, so only this file is needed.

In Go, I tried compiling with `go build` in my machine and copying the binary to a remote host.
I couldn't run it, apparently due to a mismatched glibc version or LDPATH.
Instead, I had to use: `go build -ldflags "-linkmode external -extldflags -static"  . `.
It raises a warning, but I had no issue in my tests.

In the end, the sqlite command line was the fastest of the three, and very easy to set up.

If the file you are working with is sorted and without duplicates, the best option is to create the indexes after all the data has been loaded.
You will also have to start over if the import fails or is interrupted.
It is harder to remove duplicates afterwards in such a big dataset, and the cleanest solution is to simply copy all the unique entries to a new table and delete the old one.
