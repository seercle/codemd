# Docs

<!-- codemd: (import handler-start..handler-end server.go go) -->
```go
func handler() string {
	return "ok"
}

```
<!-- codemd:generated -->

<!-- codemd: (import worker-start..worker-end worker.py) -->
```python
def work(x):
    return x * 2

```
<!-- codemd:generated -->

<!-- codemd: (import /#codemd:worker-start/../#codemd:worker-end/ worker.py python strip) -->
```python
def work(x):
    return x * 2
```
<!-- codemd:generated -->

<!-- codemd: (import worker-start.. worker.py) -->
```python
def work(x):
    return x * 2

#codemd:worker-end
```
<!-- codemd:generated -->

<!-- codemd: (link handler-start server.go go) -->
[server.go:3](server.go#L3)

<!-- codemd: (link handler-start server.go go "Handler") -->
[Handler](server.go#L3)

<!-- codemd: (link /^func handler/ server.go go) -->
[server.go:4](server.go#L4)

<!-- codemd: (link worker-start worker.py) -->
[worker.py:3](worker.py#L3)
