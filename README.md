
## crypto rates websocket


### How to run 
To run

```
  make run
```
To run in docker

```
  make run-docker
```

To run tests

```
  make test
```

To run stress test

```
artillery run artillery-test.yaml
```
Stress test creates 5000 client connections, subscribes each to rates channel and waits for 2min

### Decription 
Server connects to kraken exchange websocket to listen for crypto rate updates in realtime.
It proxies all updates to all clients that are subscribed to rates.

```ws://localhost:8080/ws``` local proxy endoint.

```{"event":"subscribe","channel":"rates"}``` message to subscribe client to rates


### TODOS/Improvements
- Track usd/cad conversion rate and show rate in CAD
- Use config file/env vars for configurable vars such as crypto pairs or exchange api
- Ping pong implementation to keep track active clients
- Binary communication with client for "lighter" payloads
- Integration tests
- Load test benchmarks with thousands active clients