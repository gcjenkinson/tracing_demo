# Tracing demonstration

Augmenting Dapper-like spans with low-level tracing captured by DTrace.
The Span contex is passed into DTrace at the start of the Span using
a USDT probe (opentracing*:jaeger:span:start) and cleared at the
end of the span (opentracing*:jaeger:span:finish). 

Within the duration of the Span DTrace scripts can correlate
tracing with the Span through a predicate that checks for the
presence of the Span Context stored in the thread local variables.

```syscall::read:entry
/this->sc != 0/
{
``` 
