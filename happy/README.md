# Howijd Prototyping Framework and SDK

DRAFT

Package howijd is the ultimate prototyping SDK written in Go. SDK which makes developers happy by giving them a great API to solve any domain problem and quickly create a working prototype (maybe even MVP). It's a tool for hackers and creators to realize their ideas when a software architect is not at hand or technical knowledge about infrastructure planning is minimal.
Howijd interface's first design is to make bleeding-edge APIs and technologies quickly available, hence the compromise to prioritize prototyping before production readiness. In addition to the above, Happy addons and services add value to the SDK by favoring the sharing and reuse of implementations in the idea phase between domains.

SDK statmens which always must be true:

- Default implementations of Happy interfaces.
- Ready to use addons, plugins, services and commands you can simply attach to your prototype application.
- Testsuite:
  -  Test your prototype applications built with this SDK.
  -  Test your interface implementations against different use cases
  -  Benchmark your interface implementations against other implementations. 
- Development UI to monitor your prototype application.


---

**to be removed from initial release**

## TODO before OS

- [ ] Revice and finalize API v0 interface only
- [ ] Revice `x` experimental packages 
- [ ] Finetune `x/sdk` which would become default implenetation `happy/sdk`
- [ ] Ensure that interfaces actually enable to replace implementations.
- [ ] Enusre that API is easy to undestand
- [ ] Prevent deadlocking
- [ ] Improve performance
- [ ] implement telemetry / metrics collection
- [ ] Add tests and test suite packages to test own interface implemntations.
- [ ] Add extensive set of examples to showcase different domain use cases
- [ ] Developemet UI for monitoring, node graph, pipelines/crons etc.??? 
- [ ] Future plans which interfaces to convert to concrete types?
- [ ] Plugin system with wasms or go plugin

## Draft Notes

Prototyping SDK for software archidects and developers alike
merging tools like https://softwarearchitecture.tools/ and actual prototype.
### Goals and key principles

- Quickly prototype any domain application for proof of concept.
- Replace SDK packages with your own implementations of required interfaces moving towards MVP. 
- Not intended for long term production use.
- 0 Dependency and no vendor lock-in


- [ ] To list of pontential of names [eng|est]

