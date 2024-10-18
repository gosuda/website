---
id: 01933e13f963942e4c9bc8c3c89fbc2b
author: snowmerak
title: Go 동시성 스타터팩
description: Go 동시성 관리를 위한 고루틴, 채널, 뮤텍스 등 다양한 도구와 기법을 소개합니다.
language: ko
date: 2024-10-18T05:51:30.010680581Z
path: /blog/posts/go-concurrency-starter-pack-z221a3399
---

## 개요

### 짧은 소개

고 언어에는 많은 동시성 관리를 위한 도구가 있습니다. 이 아티클에서는 그 중 일부와 트릭들을 소개해드리도록 하겠습니다.

### 고루틴?

goroutine은 고 언어에서 지원하는 새로운 형식의 동시성 모델입니다. 일반적으로 프로그램은 동시에 여러 작업을 수행하기 위해 OS에게서 OS 스레드를 받아서, 코어 수만큼 병렬적으로 작업을 수행합니다. 그리고 더 작은 단위의 동시성을 수행하기 위해서는 유저랜드에서 그린 스레드를 생성하여, 하나의 OS 스레드 내에서 여러 그린 스레드가 돌아가며 작업을 수행하도록 합니다. 하지만 고루틴의 경우엔 이러한 형태의 그린 스레드를 더욱 작고 효율적으로 만들었습니다. 이러한 고루틴은 스레드보다 더 적은 메모리를 사용하며, 스레드보다 더 빠르게 생성되고 교체될 수 있습니다.

고루틴을 사용하기 위해서는 단순히 `go` 키워드만 사용하면 됩니다. 이는 프로그램을 작성하는 과정에서 직관적으로 동기 코드를 비동기 코드로 실행할 수 있도록 합니다.

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    ch := make(chan struct{})
    go func() {
        defer close(ch)
        time.Sleep(1 * time.Second)
        fmt.Println("Hello, World!")
    }()

    fmt.Println("Waiting for goroutine...")
    for range ch {}
}
```

이 코드는 간단하게 1초 쉬었다가 `Hello, World!`를 출력하는 동기식 코드를 비동기 흐름으로 변경합니다. 지금의 예제는 간단하지만, 조금 복잡한 코드를 동기 코드에서 비동기 코드로 변경하게 되면, 코드의 가독성과 가시성, 이해도가 기존의 async await나 promise 같은 방식보다 더욱 좋아집니다.

다만 많은 경우에, 이러한 동기 코드를 단순히 비동기로 호출하는 흐름과 `fork & join`과 같은 흐름(마치 분할 정복과 유사한 흐름)을 이해하지 못한 상태에선 안 좋은 고루틴 코드가 만들어지기도 합니다. 이러한 경우에 대비할 수 몇가지 방법과 기법을 이 아티클에서 소개하도록 하겠습니다.

## 동시성 관리

### context

첫번째 관리 기법으로 `context`가 등장하는 건 의외일 수 있습니다. 하지만 고 언어에서 `context`는 단순한 취소 기능을 넘어서, 전체 작업 트리를 관리하는 데에 탁월한 역할을 합니다. 만약 모르시는 분들을 위해 간단히 해당 패키지를 설명하겠습니다.

```go
package main

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        <-ctx.Done()
        fmt.Println("Context is done!")
    }()

    time.Sleep(1 * time.Second)

    cancel()

    time.Sleep(1 * time.Second)
}
```

위 코드는 `context`를 사용하여, 1초 후에 `Context is done!`를 출력하는 코드입니다. `context`는 `Done()` 메소드를 통해 취소 여부를 확인할 수 있으며, `WithCancel`, `WithTimeout`, `WithDeadline`, `WithValue` 등의 메소드를 통해 다양한 취소 방법을 제공합니다.

간단한 예시를 만들어 보겠습니다. 만약 여러분들이 어떤 데이터를 가져오기 위해 `aggregator` 패턴을 사용하여, `user`, `post`, `comment`를 가져오는 코드를 작성한다고 가정해봅시다. 그리고 모든 요청이 2초 내에 이루어져야한다면, 다음과 같이 작성할 수 있습니다.

```go
package main

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 2 * time.Second)
    defer cancel()

    ch := make(chan struct{})
    go func() {
        defer close(ch)
        user := getUser(ctx)
        post := getPost(ctx)
        comment := getComment(ctx)

        fmt.Println(user, post, comment)
    }()

    select {
    case <-ctx.Done():
        fmt.Println("Timeout!")
    case <-ch:
        fmt.Println("All data is fetched!")
    }
}
```

위 코드는 2초 내에 모든 데이터를 가져오지 못하면 `Timeout!`을 출력하고, 모든 데이터를 가져오면 `All data is fetched!`를 출력합니다. 이러한 방식으로 `context`를 사용하면, 여러 고루틴이 동작하는 코드에서도 취소와 타임아웃을 쉽게 관리할 수 있습니다.

이와 관련된 다양한 context 관련 함수와 메서드가 [godoc context](https://pkg.go.dev/context)에서 확인 가능합니다. 간단한 것은 학습하여 편히 이용할 수 있게 되셨으면 합니다.

### channel

#### unbuffered channel

`channel`은 고루틴 간의 통신을 위한 도구입니다. `channel`은 `make(chan T)`로 생성할 수 있습니다. 이때, `T`는 해당 `channel`이 전달할 데이터의 타입입니다. `channel`은 `<-`로 데이터를 주고 받을 수 있으며, `close`로 `channel`을 닫을 수 있습니다.

```go
package main

func main() {
    ch := make(chan int)
    go func() {
        ch <- 1
        ch <- 2
        close(ch)
    }()

    for i := range ch {
        fmt.Println(i)
    }
}
```

위 코드는 `channel`을 사용하여 1과 2를 출력하는 코드입니다. 이 코드에서는 단순하게 `channel`에 값을 보내고 받는 것만을 보여주고 있습니다. 하지만 `channel`은 이보다 더 많은 기능을 제공합니다. 먼저 `buffered channel`과 `unbuffered channel`에 대해 알아보겠습니다. 시작하기에 앞서 위에 작성된 예제는 `unbuffered channel`로, 채널에 데이터를 보내는 행동과 데이터를 받는 행동이 동시에 이루어져야합니다. 만약 이러한 행동이 동시에 이루어지지 않는다면, 데드락이 발생할 수 있습니다.

#### buffered channel

만약 위 코드가 단순 출력이 아니라 무거운 작업을 수행하는 프로세스 2가지라면 어떤가요? 두번째 프로세스가 읽어서 처리를 수행하다가 장기간 행이 걸린다면, 첫번째 프로세스도 해당 시간 동안 멈추게 될 것입니다. 저희는 이러한 상황을 방지하기 위해 `buffered channel`을 사용할 수 있습니다.

```go
package main

func main() {
    ch := make(chan int, 2)
    go func() {
        ch <- 1
        ch <- 2
        close(ch)
    }()

    for i := range ch {
        fmt.Println(i)
    }
}
```

위 코드는 `buffered channel`을 사용하여 1과 2를 출력하는 코드입니다. 이 코드에서는 `buffered channel`을 사용하여, `channel`에 데이터를 보내는 행동과 데이터를 받는 행동이 동시에 이루어지지 않아도 되도록 만들었습니다. 이렇게 채널에 버퍼를 두게 되면, 해당 길이만큼 여유가 생겨 후순위 작업의 영향으로인해 발생하는 작업 지연을 방지할 수 있습니다.

#### select

여러 채널을 다룰 때, `select` 문법을 사용하면 쉽게 `fan-in` 구조를 구현할 수 있습니다.

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    ch1 := make(chan int, 10)
    ch2 := make(chan int, 10)
    ch3 := make(chan int, 10)

    go func() {
        for {
            ch1 <- 1
            time.Sleep(1 * time.Second)
        }
    }()
    go func() {
        for {
            ch2 <- 2
            time.Sleep(2 * time.Second)
        }
    }()
    go func() {
        for {
            ch3 <- 3
            time.Sleep(3 * time.Second)
        }
    }()

    for i := 0; i < 3; i++ {
        select {
        case v := <-ch1:
            fmt.Println(v)
        case v := <-ch2:
            fmt.Println(v)
        case v := <-ch3:
            fmt.Println(v)
        }
    }
}
```

위 코드는 주기적으로 1, 2, 3을 전달하는 3개의 채널을 만들고, `select`를 사용하여 채널에서 값을 받아 출력하는 코드입니다. 이러한 방식으로 `select`를 사용하면, 여러 채널에서 동시에 데이터를 전달 받으면서, 채널에서 값을 받는 대로 처리할 수 있습니다.

#### for range

`channel`은 `for range`를 사용하여 쉽게 데이터를 받을 수 있습니다. `for range`를 채널에 사용하게 되면 해당 채널에 데이터가 추가될 때마다 동작하게 되며, 채널이 닫히면 루프를 종료합니다.

```go
package main

func main() {
    ch := make(chan int)
    go func() {
        ch <- 1
        ch <- 2
        close(ch)
    }()

    for i := range ch {
        fmt.Println(i)
    }
}
```

위 코드는 `channel`을 사용하여 1과 2를 출력하는 코드입니다. 이 코드에서는 `for range`를 사용하여 채널에 데이터가 추가될 때마다 데이터를 받아 출력합니다. 그리고 채널이 닫히면 루프를 종료합니다.

위에 몇번 작성한 대로 이 문법은 단순 동기화 수단에 사용할 수도 있습니다.

```go
package main

func main() {
    ch := make(chan struct{})
    go func() {
        defer close(ch)
        time.Sleep(1 * time.Second)
        fmt.Println("Hello, World!")
    }()

    fmt.Println("Waiting for goroutine...")
    for range ch {}
}
```

위 코드는 1초 쉬었다가 `Hello, World!`를 출력하는 코드입니다. 이 코드에서는 `channel`을 사용하여 동기식 코드를 비동기식 코드로 변경하였습니다. 이러한 방식으로 `channel`을 사용하면, 동기식 코드를 비동기식 코드로 쉽게 변경하고, `join` 지점을 설정할 수 있습니다.

#### etc

1. nil channel에 데이터를 보내거나 받으면, 무한 루프에 빠져 데드락이 발생할 수 있습니다.
2. 채널을 닫은 후에 데이터를 보내면, panic이 발생합니다.
3. 채널을 굳이 닫지 않아도, GC가 수거하면서 채널을 닫습니다.

### mutex

#### spinlock

`spinlock`은 반복문을 돌며 계속해서 락을 시도하는 동기화 방법입니다. 고 언어에선 포인터를 사용하여 쉽게 스핀락을 구현해볼 수 있습니다.

```go
package spinlock

import (
    "runtime"
    "sync/atomic"
)

type SpinLock struct {
    lock uintptr
}

func (s *SpinLock) Lock() {
    for !atomic.CompareAndSwapUintptr(&s.lock, 0, 1) {
        runtime.Gosched()
    }
}

func (s *SpinLock) Unlock() {
    atomic.StoreUintptr(&s.lock, 0)
}

func NewSpinLock() *SpinLock {
    return &SpinLock{}
}
```

위 코드는 `spinlock` 패키지를 구현한 코드입니다. 이 코드에서는 `sync/atomic` 패키지를 사용하여 `SpinLock`을 구현하였습니다. `Lock` 메서드에서는 `atomic.CompareAndSwapUintptr`를 사용하여 락을 시도하고, `Unlock` 메서드에서는 `atomic.StoreUintptr`를 사용하여 락을 해제합니다. 이 방식은 쉬지 않고 락을 시도하기 때문에, 락을 얻을 때까지 계속해서 CPU를 사용하게 되어, 무한 루프에 빠질 수 있습니다. 따라서, `spinlock`은 단순한 동기화에 사용하거나, 짧은 시간 동안만 사용하는 경우에 사용하는 것이 좋습니다.

#### sync.Mutex

`mutex`는 고루틴 간의 동기화를 위한 도구입니다. `sync` 패키지에서 제공하는 `mutex`는 `Lock`, `Unlock`, `RLock`, `RUnlock` 등의 메소드를 제공합니다. `mutex`는 `sync.Mutex`로 생성할 수 있으며, `sync.RWMutex`로 읽기/쓰기 락을 사용할 수도 있습니다.

```go
package main

import (
    "sync"
)

func main() {
    var mu sync.Mutex
    var count int

    go func() {
        mu.Lock()
        count++
        mu.Unlock()
    }()

    mu.Lock()
    count++
    mu.Unlock()

    println(count)
}
```

위 코드에서는 거의 동시에 두 고루틴이 동일한 `count` 변수에 접근하게 됩니다. 이때, `mutex`를 사용하여 `count` 변수에 접근하는 코드를 임계 영역으로 만들어주면, `count` 변수에 대한 동시 접근을 막을 수 있습니다. 그러면 이 코드는 몇번을 실행하든 동일하게 `2`를 출력하게 됩니다.

#### sync.RWMutex

`sync.RWMutex`는 읽기 락과 쓰기 락을 구분하여 사용할 수 있는 `mutex`입니다. `RLock`, `RUnlock` 메소드를 사용하여 읽기 락을 걸고 해제할 수 있습니다.

```go
package cmap

import (
    "sync"
)

type ConcurrentMap[K comparable, V any] struct {
    sync.RWMutex
    data map[K]V
}

func (m *ConcurrentMap[K, V]) Get(key K) (V, bool) {
    m.RLock()
    defer m.RUnlock()

    value, ok := m.data[key]
    return value, ok
}

func (m *ConcurrentMap[K, V]) Set(key K, value V) {
    m.Lock()
    defer m.Unlock()

    m.data[key] = value
}
```

위 코드는 `sync.RWMutex`를 사용하여 `ConcurrentMap`을 구현한 코드입니다. 이 코드에서는 `Get` 메소드에서 읽기 락을 걸고, `Set` 메소드에서 쓰기 락을 걸어 `data` 맵에 안전하게 접근하고 수정할 수 있습니다. 읽기 락이 필요한 이유는 단순한 읽기 작업이 많은 경우, 쓰기 락을 걸지 않고 읽기 락만 걸어 여러 고루틴이 동시에 읽기 작업을 수행할 수 있도록 하기 위함입니다. 이를 통해, 굳이 상태의 변경이 없어서 쓰기 락을 걸지 않아도 되는 경우에는 읽기 락만 걸어 성능을 향상시킬 수 있습니다.

#### fakelock

`fakelock`은 `sync.Locker`를 구현하는 간단한 트릭입니다. 이 구조체는 `sync.Mutex`와 동일한 메서드를 제공하지만, 실제 동작은 하지 않습니다.

```go
package fakelock

type FakeLock struct{}

func (f *FakeLock) Lock() {}

func (f *FakeLock) Unlock() {}
```

위 코드는 `fakelock` 패키지를 구현한 코드입니다. 이 패키지는 `sync.Locker`를 구현하여 `Lock`, `Unlock` 메서드를 제공하지만, 실제로는 아무 동작도 하지 않습니다. 왜 이러한 코드가 필요한지는 기회가 되면 서술하겠습니다.

### waitgroup

#### sync.WaitGroup

`sync.WaitGroup`은 고루틴의 작업이 모두 끝날 때까지 기다리는 도구입니다. `Add`, `Done`, `Wait` 메소드를 제공하며, `Add` 메소드로 고루틴의 개수를 추가하고, `Done` 메소드로 고루틴의 작업이 끝났음을 알립니다. 그리고 `Wait` 메소드로 모든 고루틴의 작업이 끝날 때까지 기다립니다.

```go
package main

import (
    "sync"
    "sync/atomic"
)

func main() {
    wg := sync.WaitGroup{}
    c := atomic.Int64{}

    for i := 0; i < 100 ; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            c.Add(1)
        }()
    }

    wg.Wait()
    println(c.Load())
}
```

위 코드는 `sync.WaitGroup`을 사용하여 100개의 고루틴이 동시에 `c` 변수에 값을 더하는 코드입니다. 이 코드에서는 `sync.WaitGroup`을 사용하여 모든 고루틴이 끝날 때까지 기다린 후, `c` 변수에 더한 값을 출력합니다. 단순하게 몇몇개의 작업을 `fork & join`하는 경우엔 채널만을 이용해도 충분하지만, 다량의 작업을 `fork & join`하는 경우엔 `sync.WaitGroup`을 사용하는 것도 좋은 선택지입니다.

#### with slice

슬라이스와 함께 쓰인다면, `waitgroup`은 락 없이 훌륭한 동시 실행 작업을 관리하는 도구가 될 수 있습니다.

```go
package main

import (
	"fmt"
	"sync"
    "rand"
)

func main() {
	var wg sync.WaitGroup
	arr := [10]int{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			arr[id] = rand.Intn(100)
		}(i)
	}

	wg.Wait()
	fmt.Println("Done")

    for i, v := range arr {
        fmt.Printf("arr[%d] = %d\n", i, v)
    }
}
```

위 코드는 `waitgroup`만을 사용하여 각 고루틴이 동시에 10개의 랜덤 정수를 생성하여, 할당받은 인덱스에 저장하는 코드입니다. 이 코드에서는 `waitgroup`을 사용하여 모든 고루틴이 끝날 때까지 기다린 후, `Done`을 출력합니다. 이러한 방식으로 `waitgroup`을 사용하면, 여러 고루틴이 동시에 작업을 수행하고, 모든 고루틴이 끝날 때까지 락 없이 데이터를 저장하고, 작업 종료 후에 일괄적으로 후처리를 할 수 있습니다.

#### golang.org/x/sync/errgroup.ErrGroup

`errgroup`은 `sync.WaitGroup`을 확장한 패키지입니다. `errgroup`은 `sync.WaitGroup`과 달리, 고루틴의 작업 중 하나라도 에러가 발생하면 모든 고루틴을 취소하고 에러를 반환합니다.

```go
package main

import (
    "context"
    "fmt"
    "golang.org/x/sync/errgroup"
)

func main() {
    g, ctx := errgroup.WithContext(context.Background())
    _ = ctx

    for i := 0; i < 10; i++ {
        i := i
        g.Go(func() error {
            if i == 5 {
                return fmt.Errorf("error")
            }
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        fmt.Println(err)
    }
}
```

위 코드는 `errgroup`을 사용하여 10개의 고루틴을 생성하고, 5번째 고루틴에서 에러를 발생시키는 코드입니다. 의도적으로 다섯번째 고루틴에서 에러를 발생시켜, 에러가 발생하는 경우를 보여드렸습니다. 다만 실제로 사용할 때에는 `errgroup`을 사용하여 고루틴을 생성하고, 각 고루틴에서 에러가 발생하는 경우에 대해 다양한 후처리를 진행하는 방식으로 사용하면 됩니다.

### once

한 번만 실행되어야 하는 코드를 실행하는 도구입니다. 아래 생성자를 통해 관련 코드를 실행할 수 있습니다.

```go
func OnceFunc(f func()) func()
func OnceValue[T any](f func() T) func() T
func OnceValues[T1, T2 any](f func() (T1, T2)) func() (T1, T2)
```

#### OnceFunc

`OnceFunc`는 단순히 해당 함수가 전체에 걸쳐 딱 한번만 실행될 수 있게 해줍니다.

```go
package main

import "sync"

func main() {
    once := sync.OnceFunc(func() {
        println("Hello, World!")
    })

    once()
    once()
    once()
    once()
    once()
}
```

위 코드는 `sync.OnceFunc`을 사용하여 `Hello, World!`를 출력하는 코드입니다. 이 코드에서는 `sync.OnceFunc`을 사용하여 `once` 함수를 생성하고, `once` 함수를 여러 번 호출해도 `Hello, World!`가 한 번만 출력됩니다.

#### OnceValue

`OnceValue`는 단순히 해당 함수가 전체에 걸쳐 딱 한번만 실행되는 것이 아니라, 해당 함수의 반환값을 저장하여 다시 호출할 때 저장된 값을 반환합니다.

```go
package main

import "sync"

func main() {
    c := 0
    once := sync.OnceValue(func() int {
        c += 1
        return c
    })

    println(once())
    println(once())
    println(once())
    println(once())
    println(once())
}
```

위 코드는 `sync.OnceValue`를 사용하여 `c` 변수를 1씩 증가시키는 코드입니다. 이 코드에서는 `sync.OnceValue`를 사용하여 `once` 함수를 생성하고, `once` 함수를 여러 번 호출해도 `c` 변수가 한 번만 증가한 1을 반환합니다.

#### OnceValues

`OnceValues`는 `OnceValue`와 동일하게 작동하지만, 여러 값을 반환할 수 있습니다.

```go
package main

import "sync"

func main() {
    c := 0
    once := sync.OnceValues(func() (int, int) {
        c += 1
        return c, c
    })

    a, b := once()
    println(a, b)
    a, b = once()
    println(a, b)
    a, b = once()
    println(a, b)
    a, b = once()
    println(a, b)
    a, b = once()
    println(a, b)
}
```

위 코드는 `sync.OnceValues`를 사용하여 `c` 변수를 1씩 증가시키는 코드입니다. 이 코드에서는 `sync.OnceValues`를 사용하여 `once` 함수를 생성하고, `once` 함수를 여러 번 호출해도 `c` 변수가 한 번만 증가한 1을 반환합니다.

### atomic

`atomic` 패키지는 원자적 연산을 제공하는 패키지입니다. `atomic` 패키지는 `Add`, `CompareAndSwap`, `Load`, `Store`, `Swap` 등의 메소드를 제공하지만, 최근에는 `Int64`, `Uint64`, `Pointer` 등의 타입 사용을 권장합니다.

```go
package main

import (
    "sync"
    "sync/atomic"
)

func main() {
    wg := sync.WaitGroup{}
    c := atomic.Int64{}

    for i := 0; i < 100 ; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            c.Add(1)
        }()
    }

    wg.Wait()
    println(c.Load())
}
```

아까 쓰였던 예제입니다. `atomic.Int64` 타입을 사용하여 `c` 변수를 원자적으로 증가시키는 코드입니다. `Add` 메서드와 `Load` 메서드로 원자적으로 변수를 증가시키고, 변수를 읽어올 수 있습니다. 또한 `Store` 메서드로 값을 저장하고, `Swap` 메서드로 값을 교체하며, `CompareAndSwap` 메서드로 값을 비교 후 적합하면 교체할 수 있습니다.

### cond

#### sync.Cond

`cond` 패키지는 조건 변수를 제공하는 패키지입니다. `cond` 패키지는 `sync.Cond`로 생성할 수 있으며, `Wait`, `Signal`, `Broadcast` 메소드를 제공합니다.

```go
package main

import (
    "sync"
)

func main() {
    c := sync.NewCond(&sync.Mutex{})
    ready := false

    go func() {
        c.L.Lock()
        ready = true
        c.Signal()
        c.L.Unlock()
    }()

    c.L.Lock()
    for !ready {
        c.Wait()
    }
    c.L.Unlock()

    println("Ready!")
}
```

위 코드는 `sync.Cond`를 사용하여 `ready` 변수가 `true`가 될 때까지 기다리는 코드입니다. 이 코드에서는 `sync.Cond`를 사용하여 `ready` 변수가 `true`가 될 때까지 기다린 후, `Ready!`를 출력합니다. 이러한 방식으로 `sync.Cond`를 사용하면, 여러 고루틴이 동시에 특정 조건을 만족할 때까지 기다리게 할 수 있습니다.

이를 활용하여 간단한 `queue`를 구현할 수 있습니다.

```go
package queue

import (
    "sync"
    "sync/atomic"
)

type Node[T any] struct {
    Value T
    Next  *Node[T]
}

type Queue[T any] struct {
    sync.Mutex
    Cond *sync.Cond
    Head *Node[T]
    Tail *Node[T]
    Len  int
}

func New[T any]() *Queue[T] {
    q := &Queue[T]{}
    q.Cond = sync.NewCond(&q.Mutex)
    return q
}

func (q *Queue[T]) Push(value T) {
    q.Lock()
    defer q.Unlock()

    node := &Node[T]{Value: value}
    if q.Len == 0 {
        q.Head = node
        q.Tail = node
    } else {
        q.Tail.Next = node
        q.Tail = node
    }
    q.Len++
    q.Cond.Signal()
}

func (q *Queue[T]) Pop() T {
    q.Lock()
    defer q.Unlock()

    for q.Len == 0 {
        q.Cond.Wait()
    }

    node := q.Head
    q.Head = q.Head.Next
    q.Len--
    return node.Value
}
```

이렇게 `sync.Cond`를 활용하면, `spin-lock`으로 많은 CPU 사용량을 사용하는 대신에 효율적으로 대기하고, 조건이 만족되면 다시 동작할 수 있습니다.

### semaphore

#### golang.org/x/sync/semaphore.Semaphore

`semaphore` 패키지는 세마포어를 제공하는 패키지입니다. `semaphore` 패키지는 `golang.org/x/sync/semaphore.Semaphore`로 생성할 수 있으며, `Acquire`, `Release`, `TryAcquire` 메소드를 제공합니다.

```go
package main

import (
    "context"
    "fmt"
    "golang.org/x/sync/semaphore"
)

func main() {
    s := semaphore.NewWeighted(1)

    if s.TryAcquire(1) {
        fmt.Println("Acquired!")
    } else {
        fmt.Println("Not Acquired!")
    }

    s.Release(1)
}
```

위 코드는 `semaphore`를 사용하여 세마포어를 생성하고, 세마포어를 사용하여 `Acquire` 메소드로 세마포어를 획득하고, `Release` 메소드로 세마포어를 해제하는 코드입니다. 이 코드에서는 `semaphore`를 사용하여 세마포어를 획득하고 해제하는 방법을 보여드렸습니다.

## 마치며

기본적인 내용은 여기까지만 있으면 될 것같습니다. 이 아티클의 내용을 토대로, 여러분들이 고루틴을 사용하여 동시성을 관리하는 방법을 이해하고, 실제로 사용할 수 있게 되셨으면 좋겠습니다. 이 아티클이 여러분들에게 도움이 되었으면 좋겠습니다. 감사합니다.
