---
id: 191ebec04cf97317ba418a120521d30b
author: Lee Yunjin
title: 매니지드 언어란 무엇인가?
description: 매니지드 언어의 정의와 Go 언어의 바이너리 구조를 살펴봅니다.
language: ko
date: 2026-05-17T12:00:00Z
path: /blog/posts/managed-languages-intro-656b7470
---

## 매니지드 언어란 무엇인가?

매니지드 언어는 언매니지드 언어, 즉 프로그래머가 짠 로직에서 크게 벗어나지 않고 수행만 하는 언어와 달리 GC, 런타임 최적화, 그린 쓰레드, 동시성 처리 등을 런타임에서 실행하여서 사용자가 위험한 저수준 관리를 할 필요 없게 만들어 주는 언어이다.

이러한 언어의 경우, 비즈니스 로직에만 집중하여 개발에 몰입할 수 있다는 장점이 있지만, 반면 프로그래머의 직관과 실제 프로그램이 다르게 동작할 수 있어서 정교한 런타임 튜닝이 필요할 때도 있다.

먼저, 매니지드 언어 중에 가장 미니멀리스트 철학에 충실하고 어셈블리가 정직한 Go언어를 보도록 하겠다.

### Go 언어의 바이너리 구조

|   .text            |  .data        | .gopclntab, .typelink 등 |
| -----------------  | ------------  | ---------   |
| 실행될 기계어 코드 | 저장될 데이터 | 언어 런타임 섹션 |

Go언어는 사용자가 입력한대로 1:1로 기계어 번역하지 않기 때문에, .text 섹션의 로직은 언어 런타임 섹션과도 긴밀하게 연관되어 있다.

또한, 사용자가 따로 작성하지 않은 rumtime.printnl()같은 함수들이 .text 섹션 어셈블리에 추가된다.
이러한 자동적인 코드 삽입을 통해 Go 언어는 수동 관리로부터 개발자를 벗어나게 돕는다.

### Go에서 main 함수 부분만 보기

우선, 간단한 예시 소스 `main.go`를 작성해서 main부터 **AMD64 머신에서** 보도록 하자.

```go
package main

func sayHello(msg string) {
    println(msg)
}

func main() {
    sayHello("Hello World")
}
```

이후 이렇게 빌드한다.

```bash
go build main.go
```

Go는 쉬운 저수준 디버깅을 위해 `go tool`을 지원한다.
`go tool`에서 메인 패키지에서 메인 함수만큼의 어셈블리만 보기 위해서 이 구문을 입력한다.

```bash
go tool objdump -s "main\.main" ./main
```

### 어셈블리

```go
TEXT main.main(SB) /home/yjlee/compare-assembly/go/main.go
  main.go:7             0x468f60                493b6610                CMPQ SP, 0x10(R14)
  main.go:7             0x468f64                762f                    JBE 0x468f95
  main.go:7             0x468f66                55                      PUSHQ BP
  main.go:7             0x468f67                4889e5                  MOVQ SP, BP
  main.go:7             0x468f6a                4883ec10                SUBQ $0x10, SP
  main.go:8             0x468f6e                90                      NOPL
  main.go:4             0x468f6f                e8cca3fcff              CALL runtime.printlock(SB)
  main.go:4             0x468f74                488d05da290100          LEAQ 0x129da(IP), AX
  main.go:4             0x468f7b                bb0b000000              MOVL $0xb, BX
  main.go:4             0x468f80                e83bacfcff              CALL runtime.printstring(SB)
  main.go:4             0x468f85                e8f6a5fcff              CALL runtime.printnl(SB)
  main.go:4             0x468f8a                e811a4fcff              CALL runtime.printunlock(SB)
  main.go:9             0x468f8f                4883c410                ADDQ $0x10, SP
  main.go:9             0x468f93                5d                      POPQ BP
  main.go:9             0x468f94                c3                      RET
  main.go:7             0x468f95                e8e6afffff              CALL runtime.morestack_noctxt.abi0(SB)
  main.go:7             0x468f9a                ebc4                    JMP main.main(SB)
```

- 현재 쓰레드에 진입한지 CMPQ로 비교한 후, 맞다면 Entrypoint 0x468f95로 점프한다.
- 진입점을 `PUSHQ BP`로 스택에 삽입한다.
- 가장 최근에 데이터가 적재된 레지스터 SP에 함수 시작 시 스택 시작 지점을 지정하여 지역 변수 참조 시의 진입점을 고정한다.
- 이후 16바이트만큼의 로컬 변수 스택을 예약하고 (`SUBQ $0x10, SP`), NOPL을 이용해서 여러 바이트를 채워 CPU 캐시 정렬을 한다.
- Go Runtime에서 스트링 버퍼의 출력 락을 `runtime.printlock(SB)`를 호출하여 건다.
- LEAQ 명령을 이용해서 할당한 문자열의 시작 주소를 범용 레지스터 중 데이터 저장에 쓰는 누산기 주소인 AX에 저장한다.
- 이후 연산 보조 및 임시 데이터 저장에 쓰는 BX 레지스터에 문자열 길이 11을 저장한다. (`MOVL $0Xb, BX`)
- runtime.printstring(SB)로 SB 쪽으로 누산기 정보를 출력한다.
- 한 줄 공백도 rumtime.printnl(SB)로 SB쪽으로 쓴다.
- 스트링 버퍼를 runtime.printunlock(SB)로 해제한다.
- ADDQ $0x10, SP로 빌린 16바이트 스택 메모리를 돌려준다. - 처음에 진입점을 스택에 넣어 알려 줬으니 이제 POPQ BP로 스택에서 진입점을 뺀 후 반환 시그널을 준다.
- 이후 runtime.morestack_noctxt.abi0(SB)로 매니지드 언어답게 충분한 스택을 할당, GC 등의 런타임을 셋업한다.
- 관리된 main.main(SB) 주소로 이동한다.

보기와 같이 비즈니스 로직의 어셈블리는 꽤 명확하고, 얇은 런타임 관리만 덧붙여진 형태이다.

### 최적화가 없을 때

위의 형태는 Go 컴파일러에서 따로 떨어져 있는 두 함수를 자동으로 인라이닝해 최적화한 결과이다. 그러나, 우리는 학습을 위해 이 경우에는 `sayHello`를 인라이닝하지 않게 할 것입니다.

이렇게 하기 위해 다음 플래그로 소스를 컴파일한다.

```bash
 go build -gcflags="-l" main.go
```

쉘에서 결과를 찍어보면 중복되는 어셈블리가 발견된다.

```bash
yjlee@elegant:~/compare-assembly/go$ go build -gcflags="-l" main.go

go tool objdump -s "main\.sayHello" ./main
TEXT main.sayHello(SB) /home/yjlee/compare-assembly/go/main.go
  main.go:3             0x468f60                493b6610               CMPQ SP, 0x10(R14)
  main.go:3             0x468f64                7636                   JBE 0x468f9c
  main.go:3             0x468f66                55                     PUSHQ BP
  main.go:3             0x468f67                4889e5                 MOVQ SP, BP
  main.go:3             0x468f6a                4883ec10               SUBQ $0x10, SP
  main.go:5             0x468f6e                4889442420             MOVQ AX, 0x20(SP)
  main.go:5             0x468f73                48895c2428             MOVQ BX, 0x28(SP)
  main.go:4             0x468f78                e8c3a3fcff             CALL runtime.printlock(SB)
  main.go:4             0x468f7d                488b442420             MOVQ 0x20(SP), AX
  main.go:4             0x468f82                488b5c2428             MOVQ 0x28(SP), BX
  main.go:4             0x468f87                e834acfcff             CALL runtime.printstring(SB)
  main.go:4             0x468f8c                e8efa5fcff             CALL runtime.printnl(SB)
  main.go:4             0x468f91                e80aa4fcff             CALL runtime.printunlock(SB)
  main.go:5             0x468f96                4883c410               ADDQ $0x10, SP
  main.go:5             0x468f9a                5d                     POPQ BP
  main.go:5             0x468f9b                c3                     RET
  main.go:3             0x468f9c                4889442408             MOVQ AX, 0x8(SP)
  main.go:3             0x468fa1                48895c2410             MOVQ BX, 0x10(SP)
  main.go:3             0x468fa6                e8d5afffff             CALL runtime.morestack_noctxt.abi0(SB)
  main.go:3             0x468fab                488b442408             MOVQ 0x8(SP), AX
  main.go:3             0x468fb0                488b5c2410             MOVQ 0x10(SP), BX
  main.go:3             0x468fb5                eba9                   JMP main.sayHello(SB)
yjlee@elegant:~/compare-assembly/go$ go tool objdump -s "main\.sayHello" ./main
TEXT main.sayHello(SB) /home/yjlee/compare-assembly/go/main.go
  main.go:3             0x468f60                493b6610               CMPQ SP, 0x10(R14)
  main.go:3             0x468f64                7636                   JBE 0x468f9c
  main.go:3             0x468f66                55                     PUSHQ BP
  main.go:3             0x468f67                4889e5                 MOVQ SP, BP
  main.go:3             0x468f6a                4883ec10               SUBQ $0x10, SP
  main.go:5             0x468f6e                4889442420             MOVQ AX, 0x20(SP)
  main.go:5             0x468f73                48895c2428             MOVQ BX, 0x28(SP)
  main.go:4             0x468f78                e8c3a3fcff             CALL runtime.printlock(SB)
  main.go:4             0x468f7d                488b442420             MOVQ 0x20(SP), AX
  main.go:4             0x468f82                488b5c2428             MOVQ 0x28(SP), BX
OVQ 0x20(SP), AX
  main.go:4             0x468f82                488b5c2428             MOVQ 0x28(SP), BX
  main.go:4             0x468f87                e834acfcff             CALL runtime.printstring(SB)
  main.go:4             0x468f8c                e8efa5fcff             CALL runtime.printnl(SB)
  main.go:4             0x468f91                e80aa4fcff             CALL runtime.printunlock(SB)
  main.go:5             0x468f96                4883c410               ADDQ $0x10, SP
  main.go:5             0x468f9a                5d                     POPQ BP
  main.go:5             0x468f9b                c3                     RET
  main.go:3             0x468f9c                4889442408             MOVQ AX, 0x8(SP)
  main.go:3             0x468fa1                48895c2410             MOVQ BX, 0x10(SP)
  main.go:3             0x468fa6                e8d5afffff             CALL runtime.morestack_noctxt.abi0(SB)
  main.go:3             0x468fab                488b442408             MOVQ 0x8(SP), AX
  main.go:3             0x468fb0                488b5c2410             MOVQ 0x10(SP), BX
  main.go:3             0x468fb5                eba9                   JMP main.sayHello(SB)
yjlee@elegant:~/compare-assembly/go$
```

즉 컴파일러가 최적화하는 것은 이러한 중복 연산, 비효율적인 루프 언롤링 등의 대상임이 확인되었다.

### 다음 시간

다음 시간에는 Go 언어에서의 if문, switch 문을 다루도록 하겠다. 추후 시간이 난다면 Go 런타임 섹션들도 분석할 예정이다.
