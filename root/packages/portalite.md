---
author: Lemon Mint
title: Portalite를 소개합니다: 앱에 포털 Ingress를 추가하는 초경량 오픈소스 클라이언트와 SDK
path: /portalite
go_package: gosuda.org/portalite
go_repourl: https://github.com/gosuda/portalite.git
---

앞서 [Portal](https://github.com/gosuda/portal)을 소개하면서 우리는 현대 웹의 문제를 이야기했습니다. AI 코딩 도구의 등장으로 누구나 웹사이트를 만들 수 있게 되었지만, 정작 그것을 세상에 공개하는 일은 여전히 거대 플랫폼과 클라우드의 독점 영역에 남아 있습니다. Portal은 웹의 호스팅과 콘텐츠 제작을 분리하고, 개인의 로컬 서비스를 종단 간 암호화된 릴레이를 통해 전 세계에 연결함으로써, 공개할 권한을 다시 개인에게 돌려주는 프로젝트입니다.

## Portalite를 발표합니다

오늘 저는 [Portalite](https://github.com/gosuda/portalite)를 발표합니다. Portalite는 **최소 의존성으로 만들어진 초경량 Portal 클라이언트이자 Go SDK**입니다.

Portal이 "웹을 어떻게 바꿀 것인가"라는 철학이라면, Portalite는 그 철학을 개발자의 앱에 붙이는 가장 얇은 접착제입니다. 개발자는 복잡한 클라우드 설정이나 배포 파이프라인 없이, SDK를 임포트하고 몇 줄의 코드만 추가하면 자신의 앱에 포털 ingress를 넣을 수 있습니다. 앱이 실행되는 순간, 로컬에서만 살아 있던 서버는 인터넷 어디서든 접근 가능한 퍼블릭 서비스가 됩니다.

## 1. 코드 두 줄로 끝나는 퍼블릭 Ingress

Portalite는 Go의 표준 인터페이스인 [`net.Listener`](https://pkg.go.dev/net#Listener)를 지원합니다. 기존 웹 프레임워크나 서버 코드의 리스너만 교체하면 즉시 동작합니다.

```go
identity, err := portalite.GenerateIdentity("my-service")

listener, err := portalite.Expose(ctx, portalite.ExposeConfig{
    Relays:   portalite.DefaultRelays(),
    Identity: identity,
})

http.Serve(listener, handler)
```

* **무설정(Zero-config):** 포트 포워딩, 고정 IP, 방화벽 인바운드 규칙 설정이 일절 필요 없습니다.
* **표준 호환:** `http.Server`, `grpc.Server`, Gin, Echo, Chi 등 표준 `net.Listener`를 사용하는 모든 Go 라이브러리와 100% 호환됩니다.

## 2. 핵심: 멀티 릴레이 동시 바인딩 (Multi-Relay Binding)

대부분의 터널링 도구는 단일 엔드포인트에 의존하기 때문에 릴레이 서버가 죽으면 서비스도 함께 끊어집니다. **Portalite는 여러 릴레이 서버에 동시에 연결되어 동작합니다.**

* **무중단 장애 격리 (Failover):** 여러 릴레이 중 일부가 다운되거나 네트워크 지연이 발생해도, 살아있는 나머지 릴레이들이 동일한 가상 리스너를 통해 트래픽을 계속 인입합니다.
* **단일 장애점(SPOF) 제거:** 모든 릴레이 세션이 동시에 끊어지지 않는 한 애플리케이션의 `Accept()` 루프는 중단되지 않습니다.
* **종단 간 암호화 (E2EE):** 트래픽은 암호화되어 중계되므로 릴레이 서버는 데이터를 들여다보거나 변조할 수 없습니다.
* **자동 재연결 및 자가 치유(Self-Healing)**: 일시적인 네트워크 단절이나 릴레이 재부팅 시에도 백그라운드에서 세션과 토큰을 자동으로 재협상하여 연결을 복구합니다.

## 3. CLI 명령어 한 줄로 즉시 사용

Go 코드가 아니거나 빠른 테스트가 필요하다면 CLI로 포트 번호만 넘겨주면 됩니다.

```sh
$ go install gosuda.org/portalite/cmd/portalite@latest && portalite expose --name "my-service" 3000
URL https://my-service.rly.best
URL https://my-service.s-h.day

```

명령어 실행 즉시 다중 릴레이 URL이 발급되며, 곧바로 외부에서 접근 가능한 상태가 됩니다.

## 4. 가볍지만 강력한 기능

* **최소 의존성:** 서드파티 의존성을 덜어내 바이너리가 가볍고 빌드가 빠릅니다.
* **TLS & UDP 지원:** HTTP/gRPC 같은 TLS 트래픽은 물론, QUIC 데이터그램 백홀을 통해 게임 서버 같은 UDP 서비스도 손쉽게 노출할 수 있습니다.

## 시작하기

* GitHub 저장소: [https://github.com/gosuda/portalite](https://github.com/gosuda/portalite)
* Go 패키지: `go get gosuda.org/portalite`
