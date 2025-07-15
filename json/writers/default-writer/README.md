# Default JSON writer

## Performance

* Allocations
The exposed writers generally amortize all internal allocations.

There is an exception: when writing numerical values using the `Number()` method with types
from the `math/big` standard library, serialization occurs using their `AppendText` method.

Since `math/big` is not optimized for zero-allocation, there are a few buffered allocated internally by this library.


* CPU profiling
  * appendFloatG vs appendFloat

  |   | appendFloatG | appendFloat |
  |---|--------------|-------------|
  |w/ | 3668
  |wo/


 go test -v -bench . -run Bench -benchmem -benchtime 30s -cpuprofile=cpu.pprof 
goos: linux
goarch: amd64
pkg: github.com/fredbi/core/json/writers/default-writer
cpu: AMD Ryzen 7 5800X 8-Core Processor             
BenchmarkProfile
BenchmarkProfile/with_unbuffered
BenchmarkProfile/with_unbuffered/writer_profile_with_math/big_values
BenchmarkProfile/with_unbuffered/writer_profile_with_math/big_values-16         	 9823088	      3662 ns/op	     498 B/op	      10 allocs/op
BenchmarkProfile/with_unbuffered/writer_profile_without_math/big_values
BenchmarkProfile/with_unbuffered/writer_profile_without_math/big_values-16      	11984888	      3008 ns/op	       0 B/op	       0 allocs/op
BenchmarkProfile/with_buffered
BenchmarkProfile/with_buffered/writer_profile_with_math/big_values
BenchmarkProfile/with_buffered/writer_profile_with_math/big_values-16           	10152925	      3491 ns/op	     498 B/op	      10 allocs/op
BenchmarkProfile/with_buffered/writer_profile_without_math/big_values
BenchmarkProfile/with_buffered/writer_profile_without_math/big_values-16        	12769054	      2810 ns/op	       1 B/op	       0 allocs/op
PASS
ok  	github.com/fredbi/core/json/writers/default-writer	143.520s
fred@fred-dev2:~/src/github.com/fredbi/core/json/writers/default-writer$ ppro


goos: linux
goarch: amd64
pkg: github.com/fredbi/core/json/writers/default-writer
cpu: AMD Ryzen 7 5800X 8-Core Processor             
BenchmarkProfile
BenchmarkProfile/with_unbuffered
BenchmarkProfile/with_unbuffered/writer_profile_with_math/big_values
BenchmarkProfile/with_unbuffered/writer_profile_with_math/big_values-16         	 9644301	      3668 ns/op	     510 B/op	      10 allocs/op
BenchmarkProfile/with_unbuffered/writer_profile_without_math/big_values
BenchmarkProfile/with_unbuffered/writer_profile_without_math/big_values-16      	12169294	      2981 ns/op	       0 B/op	       0 allocs/op
BenchmarkProfile/with_buffered
BenchmarkProfile/with_buffered/writer_profile_with_math/big_values
BenchmarkProfile/with_buffered/writer_profile_with_math/big_values-16           	10176492	      3479 ns/op	     485 B/op	      10 allocs/op
BenchmarkProfile/with_buffered/writer_profile_without_math/big_values
BenchmarkProfile/with_buffered/writer_profile_without_math/big_values-16        	12731281	      2811 ns/op	       1 B/op	       0 allocs/op
PASS
ok  	github.com/fredbi/core/json/writers/default-writer	143.028s

generics.xs
goos: linux
goarch: amd64
pkg: github.com/fredbi/core/json/writers/default-writer
cpu: AMD Ryzen 7 5800X 8-Core Processor             
BenchmarkProfile
BenchmarkProfile/with_unbuffered
BenchmarkProfile/with_unbuffered/writer_profile_with_math/big_values
BenchmarkProfile/with_unbuffered/writer_profile_with_math/big_values-16         	 9555480	      3703 ns/op	     530 B/op	      10 allocs/op
BenchmarkProfile/with_unbuffered/writer_profile_without_math/big_values
BenchmarkProfile/with_unbuffered/writer_profile_without_math/big_values-16      	12313472	      2949 ns/op	       0 B/op	       0 allocs/op
BenchmarkProfile/with_buffered
BenchmarkProfile/with_buffered/writer_profile_with_math/big_values
BenchmarkProfile/with_buffered/writer_profile_with_math/big_values-16           	10174920	      3497 ns/op	     493 B/op	      10 allocs/op
BenchmarkProfile/with_buffered/writer_profile_without_math/big_values
BenchmarkProfile/with_buffered/writer_profile_without_math/big_values-16        	12660932	      2805 ns/op	       1 B/op	       0 allocs/op
PASS
ok  	github.com/fredbi/core/json/writers/default-writer	143.020s

go test -v -run Bench -bench . -benchtime 10s
goos: linux
goarch: amd64
pkg: github.com/fredbi/core/json/writers/default-writer
cpu: AMD Ryzen 7 5800X 8-Core Processor             
BenchmarkProfile
BenchmarkProfile/with_unbuffered
BenchmarkProfile/with_unbuffered/writer_profile_with_math/big_values
BenchmarkProfile/with_unbuffered/writer_profile_with_math/big_values-16         	 3155394	      3808 ns/op	     490 B/op	      10 allocs/op
BenchmarkProfile/with_unbuffered/writer_profile_without_math/big_values
BenchmarkProfile/with_unbuffered/writer_profile_without_math/big_values-16      	 3886904	      3073 ns/op	       4 B/op	       0 allocs/op
BenchmarkProfile/with_buffered
BenchmarkProfile/with_buffered/writer_profile_with_math/big_values
BenchmarkProfile/with_buffered/writer_profile_with_math/big_values-16           	 3206721	      3695 ns/op	     473 B/op	      10 allocs/op
BenchmarkProfile/with_buffered/writer_profile_without_math/big_values
BenchmarkProfile/with_buffered/writer_profile_without_math/big_values-16        	 4148911	      2875 ns/op	       3 B/op	       0 allocs/op
BenchmarkProfile/with_buffered2
BenchmarkProfile/with_buffered2/writer_profile_with_math/big_values
BenchmarkProfile/with_buffered2/writer_profile_with_math/big_values-16          	 3202348	      3722 ns/op	     517 B/op	      10 allocs/op
BenchmarkProfile/with_buffered2/writer_profile_without_math/big_values
BenchmarkProfile/with_buffered2/writer_profile_without_math/big_values-16       	 4006818	      2993 ns/op	       3 B/op	       0 allocs/op
BenchmarkProfile/with_indented
BenchmarkProfile/with_indented/writer_profile_with_math/big_values
BenchmarkProfile/with_indented/writer_profile_with_math/big_values-16           	 2620304	      4523 ns/op	     529 B/op	      10 allocs/op
BenchmarkProfile/with_indented/writer_profile_without_math/big_values
BenchmarkProfile/with_indented/writer_profile_without_math/big_values-16        	 3112048	      3865 ns/op	      26 B/op	       0 allocs/op
PASS
ok  	github.com/fredbi/core/json/writers/default-writer	95.552s

with new writeText

    

## Background and credits

This JSON writer has been largely inspired by the work from https://github.com/mailru/easyjson.

We've kept the concept of a writer to proces JSON tokens and escape strings, 
very much like in https://github.com/mailru/easyjson/blob/master/jwriter/writer.go.

However, this implementation introduces a few significant differences:

  * several implementations of the writers interfaces may be proposed, possibly optimized for different use-cases
  * unlike the easyjson version, we don't want to support complex types such as objects or arrays, only scalar values
  * this implementation supports the types defined to support JSON et JSON tokens by the other modules exposed
    in [github.com/fredbi/core](https://github.com/fredbi/core)
  * this makes the writer suitable for:
    * writing directly tokens produced by a [github.com/fredbi/core/json/lexers/Lexer](https://github.com/fredbi/core/blob/master/json/lexers/lexers.go)
    * writing values stored in a [github.com/fredbi/core/json/stores/Store](https://github.com/fredbi/core/blob/master/json/stores/stores.go)
    * writing JSON types defined in [github.com/fredbi/core/json/types](https://github.com/fredbi/core/blob/master/json/types/types.go)
  * the idea of a "chunked buffer" has been revisited and reimplementented. It may or may not be a good option, depending on the use-case.
    So we propose an unbuffered alternative.
  * this implementation leverages memory pools more systematically
