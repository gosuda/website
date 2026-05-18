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
매니지드 언어는 언매니지드 언어, 즉 프로그래머가 짠 로직에서 크게 벗어나지 않고 수행만 하는 언어와 달리 GC, 런타임 최적화, 그린 쓰레드, 동시성 처리 등을 런타임에서 실행하여 사용자가 위험한 저수준 관리를 할 필요 없게 만들어 주는 언어이다.
이러한 언어의 경우, 비즈니스 로직에만 집중하여 개발에 몰입할 수 있다는 장점이 있지만, 반면 프로그래머의 직관과 실제 프로그램이 다르게 동작할 수 있어서 정교한 런타임 튜닝이 필요할 때도 있다.
먼저, 매니지드 언어 중에 가장 미니멀리스트 철학에 충실하고 어셈블리가 정직한 Go 언어를 보도록 하겠다.
### Go 언어의 바이너리 구조
| .text | .data | .gopclntab, .typelink 등 |
|---|---|---|
| 실행될 기계어 코드 | 저장될 데이터 | 언어 런타임 섹션 |
Go 언어는 사용자가 입력한 대로 1:1로 기계어 번역하지 않기 때문에, .text 섹션의 로직은 언어 런타임 섹션과도 긴밀하게 연관되어 있다.
또한, 사용자가 따로 작성하지 않은 runtime.printnl() 같은 함수들이 .text 섹션 어셈블리에 추가된다.
이러한 자동적인 코드 삽입을 통해 Go 언어는 수동 관리로부터 개발자를 벗어나게 돕는다.
### Go에서 main 함수 부분만 보기
우선, 간단한 예시 소스 main.go를 작성해서 main부터 **AMD64 머신에서** 보도록 하자.
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
Go는 쉬운 저수준 디버깅을 위해 go tool을 지원한다.
go tool에서 메인 패키지에서 메인 함수만큼의 어셈블리만 보기 위해서 이 구문을 입력한다.
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
 * 현재 고루틴 스택 프레임 공간이 충분한지 고루틴 제어 블록 레지스터(R14) 내부의 스택 가드 값과 현재 스택 포인터(SP)를 CMPQ로 비교한 후, 부족하다면 스택 확장을 위한 Entrypoint인 0x468f95 주소로 점프(JBE)한다.
 * 이전 베이스 포인터를 저장하기 위해 PUSHQ BP로 스택에 삽입한다.
 * 베이스 포인터(BP) 레지스터에 현재 스택 포인터(SP)를 복사하여 함수 시작 시의 스택 기준점을 고정한다.
 * 이후 16바이트만큼의 로컬 변수 스택 공간을 할당하고 (SUBQ $0x10, SP), NOPL을 이용해 가상 명령어를 채워 CPU 캐시 정렬을 수행한다.
 * Go 런타임에서 내부 문자열 표준 출력의 동기화를 위해 runtime.printlock(SB)를 호출하여 락을 건다.
 * LEAQ 명령을 이용해 상수로 할당된 문자열("Hello World")의 시작 주소를 범용 레지스터 중 Go ABI 규격에 따라 첫 번째 매개변수로 쓰이는 AX에 저장한다.
 * 이후 문자열 길이를 나타내는 값을 두 번째 매개변수 레지스터인 BX에 저장한다. (MOVL $0xb, BX, 즉 10진수로 11)
 * runtime.printstring(SB)를 호출하여 전달된 AX(데이터 주소)와 BX(길이) 정보를 기반으로 콘솔에 출력한다.
 * 줄바꿈 처리를 위해 runtime.printnl(SB)를 호출한다.
 * 출력이 완료되었으므로 runtime.printunlock(SB)를 통해 락을 해제한다.
 * ADDQ $0x10, SP로 할당했던 16바이트 스택 메모리를 복구한다.
 * POPQ BP로 기존의 베이스 포인터를 복원한다.
 * RET를 통해 함수를 호출한 지점으로 제어권을 반환한다.
 * 만약 최초 스택 검사에서 공간이 부족했다면 0x468f95 주소의 runtime.morestack_noctxt.abi0(SB)를 호출하여 매니지드 언어답게 스택 런타임을 동적으로 확장한다.
 * 스택 확장이 완료되면 다시 main.main(SB)의 진입점으로 복귀(JMP)한다.
보기와 같이 비즈니스 로직의 어셈블리는 꽤 명확하고, 얇은 런타임 관리만 덧붙여진 형태이다.
### 최적화가 없을 때
위의 형태는 Go 컴파일러에서 따로 떨어져 있는 두 함수를 자동으로 인라이닝해 최적화한 결과이다. 그러나, 우리는 학습을 위해 이 경우에는 sayHello를 인라이닝하지 않게 할 것입니다.
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

```
인라이닝을 해제하면, 함수 호출 규격에 맞춰 매개변수(AX, BX)를 보존하기 위해 스택 포인터 오프셋인 0x20(SP) 등에 값을 다시 적재하는 MOVQ 연산들이 삽입된다.
즉 컴파일러가 최적화하는 것은 이러한 불필요한 메모리 이동 연산 및 호출 오버헤드의 대상임이 확인되었다.
### 다음 시간
다음 시간에는 Go 언어에서의 if문, switch 문을 다루도록 하겠다. 추후 시간이 난다면 Go 런타임 섹션들도 분석할 예정이다.
