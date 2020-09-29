# Go-string

[![Go Report Card](https://goreportcard.com/badge/github.com/boyter/go-string)](https://goreportcard.com/report/github.com/boyter/go-string)
[![Str Count Badge](https://sloc.xyz/github/boyter/go-string/)](https://github.com/boyter/go-string/)

Useful string utility functions for Go projects. Either because they are faster than the common Go version or do not exist in the standard library.

You can find all details here https://pkg.go.dev/github.com/boyter/go-string

Probably the most useful methods are IndexAll and IndexAllIgnoreCase which for string literal searches should be drop in replacements for regexp.FindAllIndex while totally avoiding the regular expression engine and as such being much faster.

Some quick benchmarks using a simple program which opens a 550MB file and searches over it in memory. 
Each search is done three times, the first using regexp.FindAllIndex and the second using IndexAllIgnoreCase.

For this specific example the wall clock time to run is at least 10x less.

```
$ ./csperf ſecret 550MB
File length 576683100

FindAllIndex (regex ignore case)
Scan took 25.403231773s 16680
Scan took 25.39742299s 16680
Scan took 25.227218738s 16680

IndexAllIgnoreCase (custom)
Scan took 2.04013314s 16680
Scan took 2.019360935s 16680
Scan took 1.996732171s 16680
```

Note that it performs best with real documents and wost when searching over random data. Depending on what you are searching you may have a similar speed up or a marginal one.

FindAllIndex has a similar speed up,

```
// BenchmarkFindAllIndex-8                         2458844	       480.0 ns/op
// BenchmarkIndexAll-8                            14819680	        79.6 ns/op
```

See the benchmarks for full proof where they test various edge cases.

The other most useful method is HighlightString. HighlightString takes in some content and locations and then inserts in/out
strings which can be used for highlighting around matching terms. For example you could pass in `"test"` and have it return `"<strong>te</strong>st"`.
The argument locations accepts output from regexp.FindAllIndex or the included `IndexAllIgnoreCase` or `IndexAll`.

All code is dual-licenced as either MIT or Unlicence. Your choice when you use it.

Note that as an Australian I cannot put this into the public domain, hence the choice most liberal licences I can find. 
