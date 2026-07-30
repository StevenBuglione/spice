# Spring Petclinic feedback comparison

Spice keeps a pinned, reproducible comparison with the canonical Spring
Petclinic rather than comparing unrelated toy programs. The manifest at
[`benchmarks/spring-petclinic.json`](../benchmarks/spring-petclinic.json)
selects Spring Petclinic commit
`88e37c15cf6fc8490b01bc3e8e2c800cec1ac272`, Spring Boot 4.1.0, and Java 25.

Prepare the external checkout explicitly:

```text
git clone https://github.com/spring-projects/spring-petclinic.git .tmp/spring-petclinic
git -C .tmp/spring-petclinic checkout 88e37c15cf6fc8490b01bc3e8e2c800cec1ac272
.tmp/spring-petclinic/mvnw -DskipTests compile
```

Then run the comparison without network access:

```text
make benchmark-spring SPRING_PETCLINIC=.tmp/spring-petclinic
```

The Go harness verifies the exact Spring commit, performs two warmups and seven
measured edits, restores both source files on every success or failure path,
and emits the complete samples plus p90 values as JSON. Maven runs with
`-o`; missing dependencies fail with an explicit preparation instruction.

The current body-edit compile budget is Spice p90 below five seconds and no
slower than the same Spring p90. On the reviewed Windows workstation, the
first recorded run measured Spice at 2.160 seconds p90 and Spring at 6.174
seconds p90 (ratio 0.350). These are host observations, not universal product
claims; the committed thresholds are the contract.

This scenario measures the compiler feedback portion of a save. The separate
`spice dev` event stream reports generation reuse, build, graceful restart,
and application-ready phases so restart regressions can be diagnosed instead
of hidden inside one aggregate number.
