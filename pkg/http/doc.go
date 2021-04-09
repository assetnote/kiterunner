/*
Package http provides a 0 allocation wrapper around the fasthttp library.

The wrapper facilitates our redirect handling, our custom "target" and request building format,
and handling the various options we can attach to building a request.

Most structs in this package have a corresponding Acquire* and Release* function for using sync.Pool 'd objects.
This is to minimise allocations in your request hotloop. We recommend using Acquire* and Release* wherever possible
to ensure that unecessary allocations are avoided.

The Target type provides our wrapper around the full context needed to perform a http request. There are a few quirks
when using targets that one has to be weary of. These quirks are side-effects of the zero-locking and and minimal
allocation behaviour. Admittedly, this is very developer unfriendly, and changes to the API that make it more usable
while maintaining the performance are welcome.

 - We expect Targets to be instantiated then updated as required
 - Once Target.ParseHostHeader() has been called further modifications will not be respected
 - Target.HTTPClient() will cache the first set of options that are used. Future modifications are ignored
 - All operations on a target are thread safe to use



 */
package http