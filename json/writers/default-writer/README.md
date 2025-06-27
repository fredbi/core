# Default JSON writer


## Background and credits

This JSON writer has been largely inspired by the work from https://github.com/mailru/easyjson.

We've maintained the concept of a writer to proces JSON tokens and escape strings, 
very much like in https://github.com/mailru/easyjson/blob/master/jwriter/writer.go.

However, this implementation introduces a few significant differences:

  * many implementations of [writers.Writer] may be proposed, possibly optimized for different use-cases
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
