# Facedetection example using GoCV

This example uses [GoCV](https://gocv.io/), and is based on the [Facedetection example](https://gocv.io/writing-code/face-detect/)
on the GoCV website, but converting it to use Flowbase. This way, the example
gets pipeline parallelism and is also split into three separate components. A
new component, for measuring and printing out the frames per second (FPS) is
also included.

Three files are included:

- `facedetection_orig.go` - The original code example from GoCV:s website.
- `facedetection_orig_fps.go` - The original code example, with FPS measuring added.
- `facedetection_flowbase.go` - The original code example converted to use Flowbase.

To run any of the examples, [install GoCV](https://gocv.io/getting-started/), and
then run each specific example with the accompanying shell script:

```go
./run_<orig, orig_fps or flowbase>.sh
```
