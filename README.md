FlowBase
========

A Flow-based Programming (FBP) micro-framework for Go (Golang).

The aim of FlowBase, as opposed to being a full-blown framework, is to provide just enough functionality on top of the existing FBP-like primives in Golang (channels with bounded buffers, asynchronous go-routines), to enable developing data processing applications with it. Thus the term "FBP micro-framework".

Libraries based on FlowBase
---------------------------

- [SciPipe](http://scipipe.org) - A Scientific Workflow engine library
- [RDF2SMW](https://github.com/samuell/rdf2smw) - A tool to convert RDF triples
  to a Semantic MediaWiki XML import file

References
----------

The pattern has previously been described in the following blog posts on [GopherAcademy](https://gopheracademy.com/):

- [Patterns for composable concurrent pipelines in Go](https://blog.gopheracademy.com/composable-pipelines-pattern/)
- [Composable Pipelines Improved](https://blog.gopheracademy.com/advent-2015/composable-pipelines-improvements/)

See also
--------

Other FBP frameworks in Go:
- [GoFlow](https://github.com/trustmaster/goflow)
- [Cascades](https://github.com/cascades-fbp/cascades)
