---
id: 8f79af5c6af89b748112aa8b77b348ab
author: prravda
title: How embedded NATS communicate with go application?
description: Go 애플리케이션에 임베드된 NATS가 어떻게 통신하는지, 공식 문서의 예시와 올바른 설정, 그리고 Go의 consumer-defined interface를 통해 자세히 알아봅니다.
language: ko
date: 2025-08-04T01:05:22.150350794Z
path: /blog/posts/how-embedded-nats-communicate-with-go-application-z36089af0
---

# Getting started

## About NATS

> Software applications and services need to exchange data. NATS is an infrastructure that allows such data exchange, segmented in the form of messages. We call this a "**message oriented middleware**".
> 
> 
> With NATS, application developers can:
> 
> - Effortlessly build distributed and scalable client-server applications.
> - Store and distribute data in realtime in a general manner. This can flexibly be achieved across various environments, languages, cloud providers and on-premises systems.
> 
> [What is NATS, NATS docs](https://docs.nats.io/nats-concepts/what-is-nats)
> 
- NATS 는 Go 로 구성된 메시지 브로커이다.

## Embedded NATS

> If your application is in Go, and if it fits your use case and deployment scenarios, you can even embed a NATS server inside your application.

[Embedding NATS, NATS docs](https://docs.nats.io/running-a-nats-service/clients#embedding-nats)
> 
- 그리고 NATS 의 특이사항이 있는데, Go 로 구성된 애플리케이션의 경우 embedded mode 를 지원한다는 것이다.
- 즉, 메시지 브로커의 일반적인 방식인 별도의 브로커 서버 구동 후 해당 서버와 애플리케이션의 클라이언트를 통해 통신하는 구조가 아닌, 브로커 자체를 Go 로 만든 애플리케이션에 내장(embed)할 수 있다는 이야기이다.

## Benefits and use cases of embedded NATS

- 잘 설명된 [Youtube 영상](https://youtu.be/cdTrl8UfcBo?si=KYcXOQpiyLUft6AN)이 있어서 영상의 링크로 갈음한다.
- 별도의 메시지 브로커 서버를 배포하지 않더라도 modular monolith applictaion 을 만들어서 separate of concern 을 달성하기도 하면서 nats 를 embedded 로 심을 수 있는 장점을 취할 수가 있다. 더하여 single binary deployment 도 가능해진다.
- platform with no network(wasm) 뿐만 아니라, offline-first application 에서 유용하게 사용할 수 있다.

# Example on official docs

```go
package main

import (
    "fmt"
    "time"

    "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
)

func main() {
    opts := &server.Options{}

    // Initialize new server with options
    ns, err := server.NewServer(opts)

    if err != nil {
        panic(err)
    }

    // Start the server via goroutine
    go ns.Start()

    // Wait for server to be ready for connections
    if !ns.ReadyForConnections(4 * time.Second) {
        panic("not ready for connection")
    }

    // Connect to server
    nc, err := nats.Connect(ns.ClientURL())

    if err != nil {
        panic(err)
    }

    subject := "my-subject"

    // Subscribe to the subject
    nc.Subscribe(subject, func(msg *nats.Msg) {
        // Print message data
        data := string(msg.Data)
        fmt.Println(data)

        // Shutdown the server (optional)
        ns.Shutdown()
    })

    // Publish data to the subject
    nc.Publish(subject, []byte("Hello embedded NATS!"))

    // Wait for server shutdown
    ns.WaitForShutdown()
}
```

- NATS 공식 문서가 걸어놓은 [Embedded NATS 의 예시](https://dev.to/karanpratapsingh/embedding-nats-in-go-19o)인데, 해당 예시 코드대로 진행하게 되면 embedding mode 로 communication 이 이뤄지지 않는다.

```bash
Every 2.0s: netstat -an | grep 127.0.0.1         pravdalaptop-home.local: 02:34:20
                                                                     in 0.017s (0)
...
tcp4       0      0  127.0.0.1.4222         127.0.0.1.63769        TIME_WAIT
```

- `watch 'netstat -an | grep 127.0.0.1'` command 를 통해 [localhost](http://localhost)(127.0.0.1) 로 오가는 network 를 확인하면서 `go run .` 로 해당 go file 을 실행시키면 NATS 의 default port 인 `4222` 에서 출발하는 새로운 네트워크 요청들이 추가되는 걸 볼 수 있다.

# Right configurations for embedding mode

- 의도하는 대로 embedded mode 로 통신을 하기 위해선 다음과 같은 두 가지 옵션이 필요하다.
    - Client: `InProcessServer` option 을 넣어주어야 한다.
    - Server: `Server.Options` 에 `DontListen` 이라는 flag 를 `true` 로 명시해야 한다.
- 해당 부분들은 공식적으로 문서화가 되어있진 않았고, 이 기능의 시작은 해당 [PR](https://github.com/nats-io/nats-server/pull/2360) 을 통해 파악할 수 있다.
    
    > This PR adds three things:
    > 
    > 1. `InProcessConn()` function to `Server` which builds a `net.Pipe` to get a connection to the NATS server without using TCP sockets
    > 2. `DontListen` option which tells the NATS server not to listen on the usual TCP listener
    > 3. `startupComplete` channel, which is closed right before we start `AcceptLoop`, and `readyForConnections` will wait for it
    > 
    > The main motivation for this is that we have an application that can run either in a monolith (single-process) mode or a polylith (multi-process) mode. We'd like to be able to use NATS for both modes for simplicity, but the monolith mode has to be able to cater for a variety of platforms where opening socket connections either doesn't make sense (mobile) or just isn't possible (WASM). These changes will allow us to use NATS entirely in-process instead.
    > 
    > An accompanying PR [nats-io/nats.go#774](https://github.com/nats-io/nats.go/pull/774) adds support to the client side.
    > 
    > This is my first PR to this project so apologies in advance if I've missed anything obvious anywhere.
    > 
    > /cc @nats-io/core
    > 

# Working Example for embedded mode

```go
package main

import (
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func main() {
	opts := &server.Options{
		// for configuring the embeded NATS server
		// set DonListen as true
		DontListen: true,
	}

	// Initialize new server with options
	ns, err := server.NewServer(opts)

	if err != nil {
		panic(err)
	}

	// Start the server via goroutine
	go ns.Start()

	// Wait for server to be ready for connections
	if !ns.ReadyForConnections(10 * time.Second) {
		panic("not ready for connection")
	}

	// Connect to server via in-process connection
	nc, err := nats.Connect(ns.ClientURL(), nats.InProcessServer(ns))

	if err != nil {
		panic(err)
	}

	subject := "my-subject"

	// Subscribe to the subject
	nc.Subscribe(subject, func(msg *nats.Msg) {
		// Print message data
		data := string(msg.Data)
		fmt.Println(data)

		// Shutdown the server (optional)
		ns.Shutdown()
	})

	// Publish data to the subject
	nc.Publish(subject, []byte("Hello embedded NATS!"))

	// Wait for server shutdown
	ns.WaitForShutdown()
}

```

```bash
Every 2.0s: netstat -an | grep 127.0.0.1         pravdalaptop-home.local: 02:37:50
                                                                     in 0.023s (0)
...no additional logs 

```

- 이제 의도한 대로 추가적인 network hop 이 발생하지 않는 것을 볼 수 있다.

# Under the hood

## TL;DR

![diagram1](/assets/images/embedded-nats/seq1.svg)

- 해당 코드를 `main.go` 에서 실행시켰을 때, 내부적으로 어떠한 함수들이 어떻게 작동되는지를 나타낸 sequence diagram 이며, 골자를 설명하자면 아래와 같다.
    - `DontListen: true` 를 통해 서버는 `AcceptLoop` 라는 client listening phase 를 생략한다.
    - client 의 Connect option 중 `InProcessServer` 가 활성화 된다면 in-memory connection 을 생성하고 `net.Pipe` 를 통해 pipe 를 만든 뒤 end of pipe 를 client 에게 `net.Conn` type 으로 반환한다.
    - client 와 server 가 해당 connection 을 통해 in-process communication 을 진행한다.

## Server

### AcceptLoop

```go
// nats-server/server/server.go

// Wait for clients.
if !opts.DontListen {
	s.AcceptLoop(clientListenReady)
}
```

- 먼저 `DontListen` 이 true 인 경우, `AcceptLoop` 라는 client listening phase 를 생략한다.

```go
// nats-server/server/server.go

// AcceptLoop is exported for easier testing.
func (s *Server) AcceptLoop(clr chan struct{}) {
	// If we were to exit before the listener is setup properly,
	// make sure we close the channel.
	defer func() {
		if clr != nil {
			close(clr)
		}
	}()

	if s.isShuttingDown() {
		return
	}

	// Snapshot server options.
	opts := s.getOpts()

	// Setup state that can enable shutdown
	s.mu.Lock()
	hp := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	l, e := natsListen("tcp", hp)
	s.listenerErr = e
	if e != nil {
		s.mu.Unlock()
		s.Fatalf("Error listening on port: %s, %q", hp, e)
		return
	}
	s.Noticef("Listening for client connections on %s",
		net.JoinHostPort(opts.Host, strconv.Itoa(l.Addr().(*net.TCPAddr).Port)))

	// Alert of TLS enabled.
	if opts.TLSConfig != nil {
		s.Noticef("TLS required for client connections")
		if opts.TLSHandshakeFirst && opts.TLSHandshakeFirstFallback == 0 {
			s.Warnf("Clients that are not using \"TLS Handshake First\" option will fail to connect")
		}
	}

	// If server was started with RANDOM_PORT (-1), opts.Port would be equal
	// to 0 at the beginning this function. So we need to get the actual port
	if opts.Port == 0 {
		// Write resolved port back to options.
		opts.Port = l.Addr().(*net.TCPAddr).Port
	}

	// Now that port has been set (if it was set to RANDOM), set the
	// server's info Host/Port with either values from Options or
	// ClientAdvertise.
	if err := s.setInfoHostPort(); err != nil {
		s.Fatalf("Error setting server INFO with ClientAdvertise value of %s, err=%v", opts.ClientAdvertise, err)
		l.Close()
		s.mu.Unlock()
		return
	}
	// Keep track of client connect URLs. We may need them later.
	s.clientConnectURLs = s.getClientConnectURLs()
	s.listener = l

	go s.acceptConnections(l, "Client", func(conn net.Conn) { s.createClient(conn) },
		func(_ error) bool {
			if s.isLameDuckMode() {
				// Signal that we are not accepting new clients
				s.ldmCh <- true
				// Now wait for the Shutdown...
				<-s.quitCh
				return true
			}
			return false
		})
	s.mu.Unlock()

	// Let the caller know that we are ready
	close(clr)
	clr = nil
}
```

- 참고로 AcceptLoop 함수는 다음과 같은 과정들을 진행한다. `TLS` 나 `hostPort` 와 같이 network 통신과 관련된 부분으로, in-process communication 을 하게 되면 필요 없는 부분들이니 생략해도 무방한 부분들이다.

## Client

### InProcessServer

```go

// nats-go/nats.go

// Connect will attempt to connect to the NATS system.
// The url can contain username/password semantics. e.g. nats://derek:pass@localhost:4222
// Comma separated arrays are also supported, e.g. urlA, urlB.
// Options start with the defaults but can be overridden.
// To connect to a NATS Server's websocket port, use the `ws` or `wss` scheme, such as
// `ws://localhost:8080`. Note that websocket schemes cannot be mixed with others (nats/tls).
func Connect(url string, options ...Option) (*Conn, error) {
	opts := GetDefaultOptions()
	opts.Servers = processUrlString(url)
	for _, opt := range options {
		if opt != nil {
			if err := opt(&opts); err != nil {
				return nil, err
			}
		}
	}
	return opts.Connect()
}
```

```go
// nats-go/nats.go

// Options can be used to create a customized connection.
type Options struct {
	// Url represents a single NATS server url to which the client
	// will be connecting. If the Servers option is also set, it
	// then becomes the first server in the Servers array.
	Url string

	// InProcessServer represents a NATS server running within the
	// same process. If this is set then we will attempt to connect
	// to the server directly rather than using external TCP conns.
	InProcessServer InProcessConnProvider
	
	//...
}
```

```go
// nats-go/nats.go

type InProcessConnProvider interface {
	InProcessConn() (net.Conn, error)
}
```

- nats server 와 nats client 의 연결을 진행하는 `Connect` 함수는 client URL 과 connect Option 을 설정할 수 있고, 해당 Option 들을 모아놓은 Options struct 엔 `InProcesConnProvider` interface type 의  `InProcessServer` 라는 field 가 존재한다.

```go
// main.go of example code

// Initialize new server with options
ns, err := server.NewServer(opts)

//...

// Connect to server via in-process connection
nc, err := nats.Connect(ns.ClientURL(), nats.InProcessServer(ns))
```

- nats client 에서 Connect 를 진행할 때, `InProcessServer` field 로 `nats.InProcessServer(ns)` 를 넘겨주게 되면

```go
// nats-go/nats.go

// InProcessServer is an Option that will try to establish a direction to a NATS server
// running within the process instead of dialing via TCP.
func InProcessServer(server InProcessConnProvider) Option {
	return func(o *Options) error {
		o.InProcessServer = server
		return nil
	}
}
```

- option 의 InProcessServer 가 embedded nats server 로 대체되고

```go
// nats-go/nats.go

// createConn will connect to the server and wrap the appropriate
// bufio structures. It will do the right thing when an existing
// connection is in place.
func (nc *Conn) createConn() (err error) {
	if nc.Opts.Timeout < 0 {
		return ErrBadTimeout
	}
	if _, cur := nc.currentServer(); cur == nil {
		return ErrNoServers
	}

	// If we have a reference to an in-process server then establish a
	// connection using that.
	if nc.Opts.InProcessServer != nil {
		conn, err := nc.Opts.InProcessServer.InProcessConn()
		if err != nil {
			return fmt.Errorf("failed to get in-process connection: %w", err)
		}
		nc.conn = conn
		nc.bindToNewConn()
		return nil
	}
	
	//...
}
```

- 해당 interface 는 connection 을 만들어주는 `createConn` 함수에서 `InProcessServer` option 이 nil 이 아닌(valid 한)경우 option 에 있는 InProcessServer 의 `InProcesConn` 을 실행하면서

```go
// nats-server/server/server.go

// InProcessConn returns an in-process connection to the server,
// avoiding the need to use a TCP listener for local connectivity
// within the same process. This can be used regardless of the
// state of the DontListen option.
func (s *Server) InProcessConn() (net.Conn, error) {
	pl, pr := net.Pipe()
	if !s.startGoRoutine(func() {
		s.createClientInProcess(pl)
		s.grWG.Done()
	}) {
		pl.Close()
		pr.Close()
		return nil, fmt.Errorf("failed to create connection")
	}
	return pr, nil
}
```

- server 에 구현된 `InProcessConn` 을 호출해 실행한다.
- 해당 function 은 nats 의 go client 인 `nats.go` 에서 nc(nats connection) 의 `InProcessServer` 가 nil 이 아닐 경우 호출이 되어 connection(`net.Conn`) 을 만들고, 이를 server 의 connection 에 bind 시킨다.

# Consumer driven interface of Go

> A type implements an interface by implementing its methods. There is no explicit declaration of intent, no "implements" keyword. Implicit interfaces decouple the definition of an interface from its implementation, which could then appear in any package without prearrangement.

[Interfaces are implemented implicitly, A Tour of Go](https://go.dev/tour/methods/10)
> 

> If a type exists only to implement an interface and will never have exported methods beyond that interface, there is no need to export the type itself. 

[Generality, Effective Go](https://go.dev/doc/effective_go#generality)
> 
- 해당 interface design 은 Go 에서 흔히 말 하는 consumer defined interface 와 structural typing(duck typing) 을 잘 담고 있어서 해당 주제도 같이 소개해 보려고 한다.

```go
// nats-go/nats.go

// Options can be used to create a customized connection.
type Options struct {
	// Url represents a single NATS server url to which the client
	// will be connecting. If the Servers option is also set, it
	// then becomes the first server in the Servers array.
	Url string

	// InProcessServer represents a NATS server running within the
	// same process. If this is set then we will attempt to connect
	// to the server directly rather than using external TCP conns.
	InProcessServer InProcessConnProvider
	
	//...
}
```

```go
// nats-go/nats.go

type InProcessConnProvider interface {
	InProcessConn() (net.Conn, error)
}
```

- 다시 코드로 넘어가보자. nats.go client 에서 `InProcessServer` option struct field 는 `InProcessConn` 만을 수행하는 `InProcessConnProvider` interface 로 정의되었다.

```go
// nats-server/server/server.go

// InProcessConn returns an in-process connection to the server,
// avoiding the need to use a TCP listener for local connectivity
// within the same process. This can be used regardless of the
// state of the DontListen option.
func (s *Server) InProcessConn() (net.Conn, error) {
	pl, pr := net.Pipe()
	if !s.startGoRoutine(func() {
		s.createClientInProcess(pl)
		s.grWG.Done()
	}) {
		pl.Close()
		pr.Close()
		return nil, fmt.Errorf("failed to create connection")
	}
	return pr, nil
}
```

- 그러나 그것에 들어가는 type 은 nats-server 의 `Server` 로, InProcessConn 뿐만 아니라 다양한 기능들을 수행하고 있다.
- 왜냐면 해당 상황에서의 client 의 관심사는 `InProcessConn` 이라는 interface 를 제공했느냐 아니냐 뿐이지, 다른 것들은 크게 중요하지 않기 때문이다.
- 따라서 nats.go client 는 `InProcessConn() (net.Conn, error)` 이라는 기능만을 정의한 `InProcessConnProvider` 라는 consumer defined interface 만을 만들어서 사용하고 있다.

# Conclusion

- NATS 의 embedded mode 와 그 작동방식, 그리고 NATS 의 code 를 통해 확인할 수 있는 Go 의 consumer defined interface 에 대해 간략하게 다루어 보았다.
- 해당 정보가 위와 같은 목적으로 NATS 를 사용하는 사람들에게 도움이 되길 바라며 이 글을 마치고자 한다.