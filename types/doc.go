/*
Package types defines mobydig's information model. Which is rather simple and
mainly revolves around [QualifiedAddress] and [NamedAddress], as well as the
verification [Quality] of addresses. [NamedAddress] is a [QualifiedAddress] with
an additional (DNS) name corresponding with the address value.

# Extending QualifiedAddress

Depending on how mobydig gets integrated into other applications, when using
Pingers there might be the need to add application-specific information to
qualified addresses. Basically, Pinger accepts anything that satisfies the
[QualifiedAddress] interface.

In case an implementation chooses to embed [QualifiedAddressValue] into its own
type, it is essential to (re)implement the
[QualifiedAddressValue.WithNewQuality] method. Failing to do so will cause the
embedded QualifiedAddressValue.WithNewQuality method to be propagated to the new
type, yet it won't return the proper new type, but instead only a stock
QualifiedAddressValue, loosing the additional information in the process.

# Design Rationale

The seemingly peculiar separation into a [QualifiedAddress] interface as well as
a [QualifiedAddressValue] struct type is necessary in order to allow
polymorphism. One of the fundamental fai...,erm, design decisions of Go 1 is to
not support polymorphism in form of structural subtyping (or “subclassing”).
Instead, Go provides polymorphism through interface types. So far, so bad...

Now, a [github.com/siemens/mobydig/ping/Pinger] validates addresses of unknown
quality into (un)qualified addresses. While “This is fine” in itself, there are
other situations where the qualified addresses not useful without a specific
context: for instance, a DNS name can give a qualified address meaningful
context. From the perspective of a Pinger whatever the context or concrete
structural type is, this is fine as long as it looks and smells like a qualified
address by supporting the expected interface. But “extending” the address type
given only embedding gets tricky when passing things around through different
layers.

And no, “any”/“interface{}” doesn't appear to be a sensible architectural option
here. Unfortunately, Generics don't seem to be able to improve this situation
either and appear to be extremely heavy handed. On the other hand, if there is
an idea and preferably a PR let's check and then do it.

Please keep in mind that mobydig is inherently concurrent wherever possible:
digging multiple names and pinging (validation) lots of addresses can be carried
out concurrently. Now that we're passing interface pointers(!) around through
channels (as opposed to the underlying struct values which we can't as we would
otherwise need to the concrete structural type as there is no generic “*interf”
dereference allowed), we then need to back in value semantics and immutability
through a careful [QualifiedAddress] interface design offering only getters.
This not only avoids a locking mess, but also tons of subtle bugs. The price to
pay is the ugly interface/struct types schism.
*/
package types
