FlowBase
========

A Flow-based Programming (FBP) micro-framework for Go (Golang).

The aim of FlowBase, as opposed to being a full-blown framework, is to provide just enough functionality on top of the existing FBP-like primives in Golang (channels with bounded buffers, asynchronous go-routines), to enable developing data processing applications with it. Thus the term "FBP micro-framework".

The pattern has previously been described in the following blog posts on [GopherAcademy](https://gopheracademy.com/):

- [Patterns for composable concurrent pipelines in Go](https://blog.gopheracademy.com/composable-pipelines-pattern/)
- [Composable Pipelines Improved](https://blog.gopheracademy.com/advent-2015/composable-pipelines-improvements/)


Installations
-----

```
go get github.com/flowbase/flowbase/...
```

(The ellipsis, `...`, is important, to get the `flowbase` commandline tool as well)

Usage
-----

Create a new FlowBase component stub:

```bash
flowbase new-component MyComponentName
```

(More helper commands coming later ...)

Code examples
-------------

For a real-world example, see [this code defining an app to transform from semantic RDF data to wiki pages in MediaWiki XML format](https://github.com/rdfio/rdf2smw/blob/e7e2b39c889bd68bfdc7d1f7ed507755c4435900/main.go#L100-L125) (the network connection code is highlighted, to help you find the interesting parts quick :) ).

Libraries based on FlowBase
---------------------------

- [SciPipe](http://scipipe.org) - A Scientific Workflow engine library
- [RDF2SMW](https://github.com/samuell/rdf2smw) - A tool to convert RDF triples
  to a Semantic MediaWiki XML import file

References
----------

- [FBP website](http://www.jpaulmorrison.com/fbp/)
- [FBP Wikipedia article](en.wikipedia.org/wiki/Flow-based_programming)
- [FBP google group](https://groups.google.com/forum/#!forum/flow-based-programming)

Other Go FBP frameworks
-----------------------

- [GoFlow](https://github.com/trustmaster/goflow) - The true and original Go FBP framework
- [Cascades](https://github.com/cascades-fbp/cascades)

Even more Go FBP (like) frameworks
----------------------------------

Seemingly less mature and/or well-known...

- [Ryan Peach's GoFlow](https://github.com/ryanpeach/goflow)
- [go-flow](https://github.com/7ing/go-flow)
