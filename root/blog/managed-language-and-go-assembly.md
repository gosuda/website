---
id: 260eb3cf473a1591de25a5906d102639
author: Lee Yunjin
title: '매니지드 언어와 Go 어셈블리 분석: 기초부터 If문까지'
description: 매니지드 언어의 정의와 Go 언어의 바이너리 구조를 살펴보고, If문이 어셈블리로 어떻게 번역되는지 실습을 통해 심층 분석합니다.
language: ko
date: 2026-05-17T12:00:00Z
path: /blog/posts/managed-language-and-go-assembly-b16003ea
---

## 매니지드 언어란 무엇인가?

매니지드 언어는 언매니지드 언어, 즉 프로그래머가 짠 로직에서 크게 벗어나지 않고 수행만 하는 언어와 달리 GC, 런타임 최적화, 그린 쓰레드, 동시성 처리 등을 런타임에서 실행하여서 사용자가 위험한 저수준 관리를 할 필요 없게 만들어 주는 언어입니다.

이러한 언어의 경우, 비즈니스 로직에만 집중하여 개발에 몰입할 수 있다는 장점이 있지만, 반면 프로그래머의 직관과 실제 프로그램이 다르게 동작할 수 있어서 정교한 런타임 튜닝이 필요할 때도 있습니다.

먼저, 매니지드 언어 중에 가장 미니멀리스트 철학에 충실하고 어셈블리가 정직한 Go언어를 보도록 하겠습니다.

### Go 언어의 바이너리 구조

| .text | .data | .gopclntab, .typelink 등 |
| :--- | :--- | :--- |
| 실행될 기계어 코드 | 저장될 데이터 | 언어 런타임 섹션 |

Go언어는 사용자가 입력한대로 1:1로 기계어 번역하지 않기 때문에, `.text` 섹션의 로직은 언어 런타임 섹션과도 긴밀하게 연관되어 있습니다.

또한, 사용자가 따로 작성하지 않은 `runtime.printnl()` 같은 함수들이 `.text` 섹션 어셈블리에 추가됩니다. 이러한 자동적인 코드 삽입을 통해 Go 언어는 수동 관리로부터 개발자를 벗어나게 돕습니다.

---

## Go에서 main 함수 부분만 보기

우선, 간단한 예시 소스 `main.go`를 작성해서 main부터 **AMD64 머신에서** 보도록 하겠습니다.

```go
package main

func sayHello(msg string) {
    println(msg)
}

func main() {
    sayHello("Hello World")
}
```

이후 이렇게 빌드합니다.

```bash
go build main.go
```

Go는 쉬운 저수준 디버깅을 위해 `go tool`을 지원합니다. `go tool`에서 메인 패키지에서 메인 함수만큼의 어셈블리만 보기 위해서 이 구문을 입력합니다.

```bash
go tool objdump -s "main\.main" ./main
```

### 어셈블리 분석

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

- 현재 쓰레드에 진입한지 `CMPQ`로 비교한 후, 맞다면 Entrypoint `0x468f95`로 점프합니다.
- 진입점을 `PUSHQ BP`로 스택에 삽입합니다.
- 가장 최근에 데이터가 적재된 레지스터 `SP`에 함수 시작 시 스택 시작 지점을 지정하여 지역 변수 참조 시의 진입점을 고정합니다.
- 이후 16바이트만큼의 로컬 변수 스택을 예약하고 (`SUBQ $0x10, SP`), `NOPL`을 이용해서 여러 바이트를 채워 CPU 캐시 정렬을 합니다.
- Go Runtime에서 스트링 버퍼의 출력 락을 `runtime.printlock(SB)`를 호출하여 겁니다.
- `LEAQ` 명령을 이용해서 할당한 문자열의 시작 주소를 범용 레지스터 중 데이터 저장에 쓰는 누산기 주소인 `AX`에 저장합니다.
- 이후 연산 보조 및 임시 데이터 저장에 쓰는 `BX` 레지스터에 문자열 길이 11을 저장합니다. (`MOVL $0Xb, BX`)
- `runtime.printstring(SB)`로 SB 쪽으로 누산기 정보를 출력합니다.
- 한 줄 공백도 `rumtime.printnl(SB)`로 SB쪽으로 씁니다.
- 스트링 버퍼를 `runtime.printunlock(SB)`로 해제합니다.
- `ADDQ $0x10, SP`로 빌린 16바이트 스택 메모리를 돌려줍니다.
- 처음에 진입점을 스택에 넣어 알려 줬으니 이제 `POPQ BP`로 스택에서 진입점을 뺀 후 반환 시그널을 줍니다.
- 이후 `runtime.morestack_noctxt.abi0(SB)`로 매니지드 언어답게 충분한 스택을 할당, GC 등의 런타임을 셋업합니다.
- 관리된 `main.main(SB)` 주소로 이동합니다.

보기와 같이 비즈니스 로직의 어셈블리는 꽤 명확하고, 얇은 런타임 관리만 덧붙여진 형태입니다.

### 최적화가 없을 때

위의 형태는 Go 컴파일러에서 따로 떨어져 있는 두 함수를 자동으로 인라이닝해 최적화한 결과입니다. 그러나, 우리는 학습을 위해 이 경우에는 `sayHello`를 인라이닝하지 않게 할 것입니다.

이렇게 하기 위해 다음 플래그로 소스를 컴파일합니다.

```bash
go build -gcflags="-l" main.go
```

쉘에서 결과를 찍어보면 중복되는 어셈블리가 발견됩니다.

```bash
yjlee@elegant:~/compare-assembly/go$ go build -gcflags="-l" main.go
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

즉 컴파일러가 최적화하는 것은 이러한 중복 연산, 비효율적인 루프 언롤링 등의 대상임이 확인되었습니다.

---

## Go 언어에서의 If문

먼저, 우리가 Go를 선택한 것은 Go가 모던 언어 중에서는 가장 어셈블리가 '아름답고', 고전 언어들과 비교해도 그 구문의 효율성은 오히려 압도적일 때가 있기 때문입니다.

이제 간단한 Go 프로그램의 동작 방식에 대해 이해했으니 바로 Go와 Assembly를 줄 별로 비교해 보도록 하겠습니다.

### 소스 코드

우선, Go에서도 그러하지만 심지어는 GCC를 포함한 모던 컴파일러들은 사용하는 의의가 없는 분기문을 자동으로 최적화합니다. 따라서, 컴파일러 입장에서 예측하여 다른 구문으로 바꿔 버리기 힘든 조건을 주어야 적어도 의미를 갖습니다.

```go
package main

import (
    "os"
    "strconv"
)

func main() {
    // 입력을 컴파일러가 예측할 수 없게 os.Args를 사용합니다.
    if len(os.Args) < 2 {
        return
    }
    x, _ := strconv.Atoi(os.Args[1])

    if x < 10 {
        println("X is smaller than 10")
    } else {
        println("X is larger or same as 10")
    }
}
```

이 경우, 입력을 컴파일러가 예측할 수 없기 때문에 분기문은 그대로 기계어 번역됩니다.

### 어셈블리어 분석

```asm
TEXT main.main(SB) /home/yjlee/introduction-to-golang/learn-golang/if-and-switch/golang-if/main.go
  main.go:8             0x47a840                493b6610                CMPQ SP, 0x10(R14)
  main.go:8             0x47a844                7670                    JBE 0x47a8b6
  main.go:8             0x47a846                55                      PUSHQ BP
  main.go:8             0x47a847                4889e5                  MOVQ SP, BP
  main.go:8             0x47a84a                4883ec10                SUBQ $0x10, SP
  main.go:15            0x47a84e                48833d12fb0a0002        CMPQ os.Args+8(SB), $0x2
  main.go:15            0x47a856                7c58                    JL 0x47a8b0
  main.go:15            0x47a858                488b0d01fb0a00          MOVQ os.Args(SB), CX
  main.go:18            0x47a85f                488b4110                MOVQ 0x10(CX), AX
  main.go:18            0x47a863                488b5918                MOVQ 0x18(CX), BX
  main.go:18            0x47a867                e834e8ffff              CALL strconv.Atoi(SB)
  main.go:20            0x47a86c                4883f80a                CMPQ AX, $0xa
  main.go:20            0x47a870                7d1d                    JGE 0x47a88f
  main.go:21            0x47a872                e809befbff              CALL runtime.printlock(SB)
  main.go:21            0x47a877                488d0519f50100          LEAQ 0x1f519(IP), AX
  main.go:21            0x47a87e                bb15000000              MOVL $0x15, BX
  main.go:21            0x47a883                e878c6fbff              CALL runtime.printstring(SB)
  main.go:21            0x47a888                e853befbff              CALL runtime.printunlock(SB)
  main.go:21            0x47a88d                eb1b                    JMP 0x47a8aa
  main.go:23            0x47a88f                e8ecbdfbff              CALL runtime.printlock(SB)
  main.go:23            0x47a894                488d05c8040200          LEAQ 0x204c8(IP), AX
  main.go:23            0x47a89b                bb1a000000              MOVL $0x1a, BX
  main.go:23            0x47a8a0                e85bc6fbff              CALL runtime.printstring(SB)
  main.go:23            0x47a8a5                e836befbff              CALL runtime.printunlock(SB)
  main.go:25            0x47a8aa                4883c410                ADDQ $0x10, SP
  main.go:25            0x47a8ae                5d                      POPQ BP
  main.go:25            0x47a8af                c3                      RET
  main.go:16            0x47a8b0                4883c410                ADDQ $0x10, SP
  main.go:16            0x47a8b4                5d                      POPQ BP
  main.go:16            0x47a8b5                c3                      RET
  main.go:8             0x47a8b6                e845f0feff              CALL runtime.morestack_noctxt.abi0(SB)
  main.go:8             0x47a8bb                eb83                    JMP main.main(SB)
```

### CMPQ 명령어 & JL 명령어

`CMPQ` 명령어는 4바이트(4워드) 자료형을 비교하기 위한 명령어이고, 어원은 **C**o**MP**are **Q**uadword입니다.

`0x47a84e`번 메모리 주소를 보면, `CMPQ os.Args+8(SB), $0x2` 구문이 들어가 있습니다. 이 경우, 프로그램이 입력받은 인자 수와 16진수 `0x2`(즉 2)를 비교합니다.

이후, 인자가 2보다 작은지 `JL` (**J**ump if **L**ess)을 통해 비교 후 점프를 수행합니다. 인자가 2보다 작았다면, `0x47a8b0` 주소로 점프하여 함수를 종료합니다.

### MOVQ 명령어와 Go의 String 구조

`0x47858`-`0x47863` 범위를 보면 `os.Args`에서 데이터를 가져오는 과정을 볼 수 있습니다. Go의 `string`은 구조체이며, 16바이트(8바이트 포인터 + 8바이트 길이)로 구성됩니다.

| struct | 8 byte | 8 byte |
| :--- | :--- | :--- |
| string | memory address | string length |

따라서 스트링의 주소를 `AX` 레지스터, 길이를 `BX` 레지스터에 저장하여 함수로 전달하거나 처리합니다.

### CMPQ 명령어 & JGE 명령어 (역조건 최적화)

주소 `0x47a86c`를 보면 `CMPQ AX, $0xa` (10과 비교) 후 `JGE` (**J**ump if **G**reater or **E**qual) 명령어가 등장합니다.

우리의 소스는 `if x < 10`이었지만, 어셈블리에서는 `x >= 10`이면 else 블록으로 점프하게 되어 있습니다. 이것은 **역조건을 이용해 점프 횟수를 줄이는 전형적인 컴파일러 최적화**입니다.

---

### 거울상 코드 검증 (Mirroring Code)

아래 스크립트를 이용하면 서로 다른 소스 코드가 어떻게 동일한 어셈블리 결과를 낼 수 있는지 확인할 수 있습니다.

```bash
#!/usr/bin/env bash

# 1. 기존 잔여 파일 및 디렉터리 완전 초기화
rm -rf test_dir main_orig main_asm orig.asm asm.asm orig_pure.asm asm_pure.asm
mkdir -p test_dir

# 2. 원래 버전 소스 코드 작성 (main.go)
cat << 'EOF' > main.go
package main
import ("os"; "strconv")
func main() {
    if len(os.Args) < 2 { return }
    x, _ := strconv.Atoi(os.Args[1])
    s1 := "X is smaller than 10"
    s2 := "X is larger or same as 10"
    if x < 10 { println(s1) } else { println(s2) }
}
EOF

# 3. 거울상 버전 소스 코드 작성 (main_from_asm.go)
cat << 'EOF' > main_from_asm.go
package main
import ("os"; "strconv")
func main() {
    if len(os.Args) < 2 { return }
    x, _ := strconv.Atoi(os.Args[1])
    s1 := "X is smaller than 10"
    s2 := "X is larger or same as 10"
    // 구조적으로 동일하게 작성
    if x < 10 { println(s1) } else { println(s2) }
}
EOF

# 4. 빌드 수행
cp main.go test_dir/main.go
cd test_dir && go build -o ../main_orig main.go && cd ..
rm test_dir/main.go
cp main_from_asm.go test_dir/main.go
cd test_dir && go build -o ../main_asm main.go && cd ..

# 5. 어셈블리 추출 및 비교
go tool objdump -s "main\.main" main_orig > orig.asm
go tool objdump -s "main\.main" main_asm > asm.asm
awk '{print $4, $5, $6, $7}' orig.asm > orig_pure.asm
awk '{print $4, $5, $6, $7}' asm.asm > asm_pure.asm

if diff orig_pure.asm asm_pure.asm > /dev/null; then
    echo "===> [성공] 두 바이너리의 main.main 기계어 로직이 100% 일치합니다!"
else
    echo "===> [실패] 차이점이 발견되었습니다."
fi
```

### 결론

프로그래밍 언어는 많은 추상화를 제공하지만, 그 이면에는 공격적인 최적화들이 숨어 있습니다. 이러한 저수준 동작을 이해하면 더 효율적인 코드를 작성하는 데 도움이 될 뿐만 아니라, 바이너리 분석과 같은 보안 영역에서도 큰 자산이 됩니다.

다음 시간에는 If문만큼이나 재미있는 `select-case` 문과 Go 런타임의 심화 섹션들을 분석해 보도록 하겠습니다.
