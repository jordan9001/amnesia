AMNESIA
=======

Goals
-----

Amnesia is a fuzzer designed to enable vulnerability research. It is a
gray box fuzzing library that allows for the quick creation of effective
smart fuzzers. Its main goals are to:

-   Enable fuzzer creation for a wide range of targets.

-   Allow quick memory fuzzing.

-   Allow instrumentation for fuzzing performance.

-   Provide useful features, for quick creation of smart fuzzers.

Major Features
--------------

Amnesia provides the following features for its users:

-   A Native Golang library.

-   64-bit Linux support (with more system support planned).

-   Mutation tools.

-   Fork Server instrumentation.

-   File Descriptor Hijacking.

-   In memory buffer fuzzing.

-   Fault Reporting.

-   Simple Callback interface.

-   High levels of Concurrency.

Architecture
------------

Amnesia provides a backend that
will start the process, manage infection, and provide data channels for
the user to use to send input and receive output or faults. The user’s
code can then decide what faults and outputs to report as valid “hits”.
The infection mechanism is operating system specific.

On 64-bit linux machines, the infection works in the following steps:

1.  A “hook” is created by customizing a precompiled segment of assembly
    code with program specific variables.

2.  The hook is placed at the specified location of the target binary.
    The overwritten code is remembered by the fuzzer.

3.  A “package” is created by customizing a larger precompiled segment
    of assembly code that will manage the advanced features of the
    fuzzer, such as the fork server, file descriptor hijacking, and in
    memory buffer fuzzing.

4.  The fuzzer executes the modified binary, and allows the user to
    handle any actions that must be completed prior to execution of the
    hook.

5.  Once the hook is reached, the package is mapped into memory as
    executable and called.

6.  The package restores the original binary contents over the hook, and
    starts the fork server.

7.  Forked off children perform the actions as specified by the user,
    and are returned to execution of the target binary.

Future Work
-----------

In the future, we will seek to bring amnesia an the same infection
techniques to many platforms. The cross-platform nature of golang
enables this greatly, as well as the infections techniques used. Amnesia
also requires a lot of speed testing and optimization to ensure it is
competitive with other fuzzing alternatives.
