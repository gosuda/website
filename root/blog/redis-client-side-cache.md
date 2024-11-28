---
author: snowmerak
title: 레디스 클라이언트 사이드 캐시로 반응성 향상 시키기
description: Redis Server Assisted Client Side Cache를 이용한 API 반응성 향상
language: ko
---

## What is Redis?

레디스를 모르는 분은 별로 없을 거라 생각합니다. 하지만 그래도 몇가지 특성으로 짧게 언급하고 넘어가자면 다음과 같이 정리할 수 있을 것입니다.

- **단일 스레드**에서 연산이 수행되어, 모든 연산이 **원자성**을 가집니다.
- **In-Memory**에 데이터가 저장되고 연산되어, 모든 연산이 **빠릅**니다.
- 레디스는 옵션에 따라 **WAL**을 저장할 수 있어, 빠르게 최신 상태를 **백업**하고 **복구**할 수 있습니다.
- Set, Hash, Bit, List 등의 **여러가지 타입**을 지원하여, 높은 **생산성**을 가집니다.
- **큰 커뮤니티**를 가지고 있어, 다양한 경험과 이슈, 해결법을 **공유**받을 수 있습니다.
- **오랫동안 개발 및 운영**되어, 신뢰할 수 있는 **안정성**이 있습니다.

## 그래서 본론으로

### 상상해보세요?

만약 여러분들의 서비스의 캐시가 다음 두가지 조건에 부합한다면 어떨까요?

1. 자주 조회되는 데이터를 최신 상태로 사용자에게 제공해야하지만, 갱신이 불규칙하여 캐시 갱신을 빈번하게 해야할 때
2. 갱신은 안되지만, 동일한 캐시 데이터에 자주 접근해서 조회해야할 때

첫번째 케이스는 쇼핑몰 실시간 인기 순위를 고려할 수 있습니다. 쇼핑몰 실시간 인기 순위를 sorted set으로 저장했을 때, 레디스에서 사용자가 메인 페이지에 접근할 때마다 읽으면 비효율적입니다.  
두번째 케이스는 환율 데이터에 대해, 대략적으로 10분 주기로 환율 데이터가 고시되어도 실제 조회는 매우 빈번하게 발생합니다. 그것도 원-달러, 원-엔, 원-위안에 대해서는 매우 빈번하게 캐시를 조회하게 됩니다.  
이러한 케이스들에서는 API 서버가 로컬에 별도의 캐시를 가지고 있다가, 데이터가 변경되면 레디스를 다시 조회해서 갱신하는 편이 효율적인 동작일 것입니다.

그러면 어떻게 하면 데이터베이스 - 레디스 - API 서버 구조에서 이러한 동작을 구현할 수 있을까요??

### Redis PubSub으로 안되나?

> 캐시를 사용할 때, 갱신 여부를 받을 수 있는 채널을 구독하자!

- 그럼 갱신 시에 메시지를 전송하는 로직을 만들어야 합니다.
- PubSub으로 인한 추가 동작이 들어가기에 성능에 영향을 줍니다.

![pubsub-write](/assets/images/redis-client-side-cache/01-pubsub-write.png)

![pubsub-read](/assets/images/redis-client-side-cache/01-pubsub-read.png)

### 그럼 Redis가 변경을 감지한다면?

> Keyspace Notification을 사용하여 해당 키에 대한 커맨드 알림을 받으면?

- 갱신에 쓰이는 키와 커맨드를 미리 저장하고 공유해야하는 번거로움이 존재합니다.
- 예를 들어, 어떤 키에 대해선 단순 Set이 갱신 커맨드고, 어떤 키는 LPush, 혹은 RPush나 SAdd 및 SRem이 갱신 커맨드가 되는 등 복잡해집니다.
- 이는 개발 과정에서 커뮤니케이션 미스와 코딩에서 휴먼 에러를 발생시킬 가능성이 대폭 증가합니다.


> Keyevent Notification을 사용하여 커맨드 단위로 알림을 받으면?

- 갱신에 쓰이는 모든 커맨드에 대한 구독이 필요합니다. 거기서 들어오는 키에 대해 적절한 필터링이 필요합니다.
- 예를 들어, Del로 들어오는 모든 키 중 일부에 대해 해당 클라이언트는 로컬 캐시가 없을 가능성이 높습니다.
- 이는 불필요한 리소스 낭비로 이어질 수 있습니다.

### 그래서 필요한 것이 Invalidation Message!

#### Invalidation Message가 무엇?

Invalidation Messages는 Redis 6.0부터 추가된 Server Assisted Client-Side Cache의 일환으로 제공되는 개념입니다. Invalidation Message는 다음 흐름으로 전달됩니다.

1. ClientB가 이미 key를 한번 읽었다고 가정합니다.
2. ClientA가 해당 key를 새로 설정합니다.
3. Redis는 변경을 감지하고 ClientB에 Invalidation Message를 발행해서 ClientB에 캐시를 지우라고 알립니다.
4. ClientB는 해당 메시지를 받아서 적절한 조치를 취합니다.

![invalidation-message](/assets/images/redis-client-side-cache/02-invalidation-message.png)

### 어떻게 쓰는 거지

#### 기본 동작 구조

레디스에 연결된 클라이언트가 `CLIENT TRACKING ON REDIRECT <client-id>`를 실행함으로 invalidation message를 받도록 합니다. 그리고 메시지를 받아야 하는 클라이언트는 `SUBSCRIBE __redis__:invalidate`로 invalidation message를 받도록 구독합니다.

#### default tracking

```shell
# client 1
> SET a 100
```

```shell
# client 3
> CLIENT ID
12
> SUBSCRIBE __redis__:invalidate
1) "subscribe"
2) "__redis__:invalidate"
3) (integer) 1
```

```shell
# client 2
> CLIENT TRACKING ON REDIRECT 12
> GET a # tracking
```

```shell
# client 1
> SET a 200
```

```shell
# client 3
1) "message"
2) "__redis__:invalidate"
3) 1) "a"
```

#### broadcasting tracking

```shell
# client 3
> CLIENT ID
12
> SUBSCRIBE __redis__:invalidate
1) "subscribe"
2) "__redis__:invalidate"
3) (integer) 1
```

```shell
# client 2
CLIENT TRACKING ON BCAST PREFIX cache: REDIRECT 12
```

```shell
# client 1
> SET cache:name "Alice"
> SET cache:age 26
```

```shell
# client 3
1) "message"
2) "__redis__:invalidate"
3) 1) "cache:name"
1) "message"
2) "__redis__:invalidate"
3) 1) "cache:age"
```

## 구현! 구현! 구현!

### Redigo + Ristretto

저렇게만 설명하면 실제로 코드 상에서 사용할 때에 어떻게 써야할지 애매합니다. 그러니 간단하게 `redigo`와 `ristretto`로 먼저 구성해 보겠습니다.

먼저 두 디펜던시를 설치합니다.

- `github.com/gomodule/redigo`
- `github.com/dgraph-io/ristretto`

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/gomodule/redigo/redis"
)

type RedisClient struct {
	conn  redis.Conn
	cache *ristretto.Cache[string, any]
	addr  string
}

func NewRedisClient(addr string) (*RedisClient, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache: %w", err)
	}

	conn, err := redis.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		conn:  conn,
		cache: cache,
		addr:  addr,
	}, nil
}

func (r *RedisClient) Close() error {
	err := r.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	return nil
}
```

먼저 간단하게 ristretto와 redigo를 포함하는 `RedisClient`를 생성합니다.

```go
func (r *RedisClient) Tracking(ctx context.Context) error {
	psc, err := redis.Dial("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	clientId, err := redis.Int64(psc.Do("CLIENT", "ID"))
	if err != nil {
		return fmt.Errorf("failed to get client id: %w", err)
	}
	slog.Info("client id", "id", clientId)

	subscriptionResult, err := redis.String(r.conn.Do("CLIENT", "TRACKING", "ON", "REDIRECT", clientId))
	if err != nil {
		return fmt.Errorf("failed to enable tracking: %w", err)
	}
	slog.Info("subscription result", "result", subscriptionResult)

	if err := psc.Send("SUBSCRIBE", "__redis__:invalidate"); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	psc.Flush()

	for {
		msg, err := psc.Receive()
		if err != nil {
			return fmt.Errorf("failed to receive message: %w", err)
		}

		switch msg := msg.(type) {
		case redis.Message:
			slog.Info("received message", "channel", msg.Channel, "data", msg.Data)
			key := string(msg.Data)
			r.cache.Del(key)
		case redis.Subscription:
			slog.Info("subscription", "kind", msg.Kind, "channel", msg.Channel, "count", msg.Count)
		case error:
			return fmt.Errorf("error: %w", msg)
		case []interface{}:
			if len(msg) != 3 || string(msg[0].([]byte)) != "message" || string(msg[1].([]byte)) != "__redis__:invalidate" {
				slog.Warn("unexpected message", "message", msg)
				continue
			}

			contents := msg[2].([]interface{})
			keys := make([]string, len(contents))
			for i, key := range contents {
				keys[i] = string(key.([]byte))
				r.cache.Del(keys[i])
			}
			slog.Info("received invalidation message", "keys", keys)
		default:
			slog.Warn("unexpected message", "type", fmt.Sprintf("%T", msg))
		}
	}
}
```

코드가 조금 복잡합니다. 

- Tracking을 하기 위해 커넥션을 하나 더 맺습니다. 이는 PubSub이 다른 동작의 방해가 될 것을 고려한 조치입니다.
- 추가된 커넥션의 아이디를 조회하여, 데이터를 조회할 커넥션에서 Tracking을 해당 커넥션으로 Redirect하게 합니다.
- 그리고 invalidation message를 구독합니다.
- 구독을 처리하는 코드가 조금 복잡합니다. redigo가 무효화 메시지에 대한 파싱이 되지 않기에, 파싱 전 응답을 받아서 처리해야합니다.

```go
func (r *RedisClient) Get(key string) (any, error) {
	val, found := r.cache.Get(key)
	if found {
		switch v := val.(type) {
		case int64:
			slog.Info("cache hit", "key", key)
			return v, nil
		default:
			slog.Warn("unexpected type", "type", fmt.Sprintf("%T", v))
		}
	}
	slog.Info("cache miss", "key", key)

	val, err := redis.Int64(r.conn.Do("GET", key))
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	r.cache.SetWithTTL(key, val, 1, 10*time.Second)
	return val, nil
}
```

`Get` 메시지는 다음과같이 리스트레토를 먼저 조회하고, 없다면 레디스에서 가져오도록 합니다.

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client, err := NewRedisClient("localhost:6379")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	go func() {
		if err := client.Tracking(ctx); err != nil {
			slog.Error("failed to track invalidation message", "error", err)
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	done := ctx.Done()

	for {
		select {
		case <-done:
			slog.Info("shutting down")
			return
		case <-ticker.C:
			v, err := client.Get("key")
			if err != nil {
				slog.Error("failed to get key", "error", err)
				return
			}
			slog.Info("got key", "value", v)
		}
	}
}
```

테스트하기 위한 코드는 위와 같습니다. 한번 테스트 해보시면 레디스에서 데이터가 갱신될 때마다 새로 값을 갱신하는 걸 확인할 수 있을 것입니다.

하지만 이는 너무 복잡합니다. 무엇보다 클러스터에 대해 확장하기 위해 필연적으로 모든 마스터, 혹은 레플리카에대해 Tracking을 활성화할 필요가 있습니다.

### Rueidis

Go 언어를 쓰는 이상, 저희에겐 가장 모던하고 발전한 `rueidis`가 있습니다. rueidis를 사용한 레디스 클러스터 환경에서의 server assisted client side cache를 사용하는 코드를 작성해 보겠습니다.

먼저, 의존성을 설치합니다.

- `github.com/redis/rueidis`

그리고 레디스에 데이터를 조회하는 코드를 작성합니다.

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/redis/rueidis"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{"localhost:6379"},
	})
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	done := ctx.Done()

	for {
		select {
		case <-done:
			slog.Info("shutting down")
			return
		case <-ticker.C:
			const key = "key"
			resp := client.DoCache(ctx, client.B().Get().Key(key).Cache(), 10*time.Second)
			if resp.Error() != nil {
				slog.Error("failed to get key", "error", resp.Error())
				continue
			}
			i, err := resp.AsInt64()
			if err != nil {
				slog.Error("failed to convert response to int64", "error", err)
				continue
			}
			switch resp.IsCacheHit() {
			case true:
				slog.Info("cache hit", "key", key)
			case false:
				slog.Info("missed key", "key", key)
			}
			slog.Info("got key", "value", i)
		}
	}
}
```

rueidis에는 client side cache를 사용하기 위해 그저 `DoCache`만 하면 됩니다. 그러면 로컬 캐시에서 얼마나 유지할 것인지와 같이 로컬 캐시에 추가하고, 동일하게 `DoCache`를 호출하면 로컬 캐시 내에서 데이터를 조회해서 가져옵니다. 당연하게도 무효화 메시지도 정상적으로 처리합니다.

### 왜 안 redis-go?

`redis-go`는 아쉽게도 공식 API로 server assisted client side cache를 지원하지 않습니다. 심지어 PubSub을 생성할 때 새로운 커넥션을 만들면서 해당 커넥션에 직접 접근하는 API가 없어서 client id를 알 수도 없습니다. 그래서 `redis-go`는 구성 자체가 불가능 하다고 판단하여 패스했습니다.

## 섹시하군

### client side cache 구조를 통해

- 미리 준비할 수 있는 데이터라면 이 구조를 통해 레디스에 대한 쿼리 및 트래픽을 최소화하며 항상 최신 데이터를 제공할 수 있을 것입니다.
- 이를 통해 일종의 CQRS 구조를 만들어서 읽기 성능을 비약적으로 올릴 수 있습니다.

![cqrs](/assets/images/redis-client-side-cache/03-cqrs.jpg)

### 얼마나 더 섹시해졌는지?

실제로 현장에 마침 이러한 구조로 사용 중이므로 두 API에 대해 간단한 레이턴시를 찾아봤습니다. 매우 추상적으로밖에 쓰지 못 하는 점 양해 부탁드립니다.

1. 첫번째 API
   1. 최초 조회 시: 평균 14.63ms
   2. 이후 조회 시: 평균 2.82ms
   3. 평균 격차: 10.98ms
2. 두번째 API
   1. 최초 조회 시: 평균 14.05ms
   2. 이후 조회 시: 평균 1.60ms
   3. 평균 격차: 11.57ms

많게는 82% 정도의 추가적인 레이턴시 개선이 있었습니다!

## 결론

개인적으로는 만족스러운 아키텍처 구성요소였고, 레이턴시 및 API 서버에 대한 스트레스도 굉장히 적었습니다. 앞으로도 가능하다면 이런 구조로 아키텍처를 구성하면 좋겠다고 생각하고 있습니다.
