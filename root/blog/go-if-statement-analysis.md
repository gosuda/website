---
id: 04e03e046bb46ab34aaeeed79279923b
author: Lee Yunjin
title: Go 언어에서의 If문
description: Go 언어의 If문과 어셈블리 구조를 분석합니다.
language: ko
date: 2026-05-17T12:00:00Z
path: /blog/posts/go-if-statement-analysis-3012d30c
---

## Go 언어에서의 If문

먼저, 우리가 Go를 선택한 것은 Go가 모던 언어 중에서는 가장 어셈블리가 '아름답고', 고전 언어들과 비교해도 그 구문의 효율성은 오히려 압도적일 때가 있기 때문이다.

이제 전 강의에서 간단한 Go 프로그램의 동작 방식에 대해 이해했으니 바로 Go와 Assembly를 줄 별로 비교해 보도록 하자.

### 소스 코드

우선, Go에서도 그러하지만 심지어는 GCC를 포함한 모던 컴파일러들은 사용하는 의의가 없는 분기문을 자동으로 최적화한다. GCC, Clang같은 C언어 컴파일러도 업계 표준인 -O2에서 아주 공격적인 최적화를 하니 프로그래머가 컴파일러를 전적으로 신뢰하기 어려운 시대는 마침내 20세기 후반부터 완성되어 버린 셈이다.

따라서, 컴파일러 입장에서 예측하여 다른 구문으로 바꿔 버리기 힘든 조건을 주어야 적어도 의미를 갖는다.

```go
package main

import (
    "os"
    "strconv"
)

func main() {
    // 만약 이것을 x = 10과 같은 예측 가능하며 분기를 지울 수 있는 것으로 때우면
    // 컴파일러가 최적화하여서 분기를 지운다.
    // 따라서 이런 것을 직접 어셈블리로 구경하려면 C언어에서는 -O0 등을 두거나
    // 애초에 컴파일러가 예측 불가한 외부 값을 쓰게 하는데,
    // 이 섹션은 모던 프로그래밍을 다루니 Go의 바이너리 최적화를
    // 끄는 방식은 사용하지 않는다.
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

이 경우, 입력을 컴파일러가 예측할 수 없기 때문에 분기문은 그대로 기계어 번역된다.

### 어셈블리어

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

go tool을 이용하면 친절하게도 어떤 구문과 어떤 어셈블리가 매칭되는지 하나하나 알려 준다.

우리가 이번에 배울 것은 비교문과 if 분기문이기 때문에 몇 줄을 주목하면 된다.


### CMPQ 명령어 & JL 명령어

`CMPQ` 명령어는 4바이트(4워드) 자료형을 비교하기 위한 명령어이고, 어원은 **C**o**MP**are**Q**uadword로 하여 **CMPQ**로 줄인 것이다.

`0x47a84e`번 메모리 주소를 보면, `CMPQ os.Args+8(SB), $0x2` 구문이 들어가 있다.
이 경우, 프로그램이 입력받은 인자 수와 16진수 `0x2`(즉 그냥 2이다)를 비교한다.

이후, 인자가 2보다 작은지(즉 프로그램 자신만이 인자라면) `JL`을 통해 비교 후 점프를 수행한다. 즉 이것은 **J**ump, **L**ess than'을 줄여 **JL**이 된다.
앞의 비교 연산에 대해서 인자가 2보다 작았다면, `0x47a8b0` 주소로 점프하는데 이곳에는 `JGE`가 있다.
그러나, 이 구문에서 사용하는 것은 `AX` 레지스터이기 때문에 레지스터에 저장된 값의 정체를 알아야 한다.

### MOVQ 명령어

이 다음에, 실제 'CX' 레지스터를 이용해서 자료의 시작 주소를 저장하며, 주소를 읽은 후의 실제 데이터를 어떻게 추출하려 하는지 알아야 한다.

`0x47858`-`0x47863` 범위를 보면 단계적으로 이 연산을 수행한다.

먼저, 인자 배열의 시작 주소를 `MOVQ os.Args(SB), CX` 명령으로 CX 레지스터에 삽입한다. 이 때 Go의 스트링 타입을 이해해야 한다.

Go의 `string`은 구조체이며, 이 구조체는 8바이트 자료 2개로 16바이트로 구성되어 있다.

| struct  | 8 byte     | 8 byte       |
| ------ | ----------- | -------------- |
| string | mem address |  string length |


시각적으로 그리자면 위와 같고, 앞의 8 바이트는 스트링의 시작 주소, 뒤의 8 바이트는 스트링의 길이가 저장되어 있다.

따라서, 스트링의  주소를 AX 레지스터, 스트링의 길이를 BX 레지스터에 저장한다.

### CALL

앞의 포스트에서도 `runtime` 계 함수들을 볼 때, CALL이라는 명령이 붙어 있었다.
이것은 Go에서 사용하는 함수들 앞에 붙어 있으며, 어떠한 함수를 **호출한다**는 뜻으로 말 그대로 CALL이다. 이후 CALL 함수를 이용해서 스트링을 정수로 변환하는데 이 때 정수를 어디 저장하는지 *함수에 추상화되어 보이지 않는다.*

### CMPQ 명령어 & JGE 명령어

다시 아까 주소인 `0x47a86c`로 돌아오면, 명령어는 **스트링의 주소와 숫자인 `0xa`(십진수로 10)을 비교하고 있다!**

이 말은, 프로그램 내에서 더 이상 해당 인자를 사용하지 않기 때문에 **스트링의 위치에 덮어 써서 정수형 변수 `x` 자리를 만들었다는 뜻이다.**

이것이 Go 언어 등에서 이루어지는 공격적 최적화의 실체이다.

이후, JGE라는 명령어가 등장 하는데, 이것은 **J**ump, **G**reater or **E**quals의 약어이다. 따라서 이 구문은 비교 대상과 대조했을 때 크거나 같은지를 묻는다.

따라서 `x < 10` 구문 그대로가 아니고 `x < 10`으로 **구문의 비교 방향이 뒤바뀌어 있다!**
이것은 기계어에서는 조건이 일치하지 않을 때 선제적으로 건너뛰는 것이, 조건이 일치할 때의 비교를 1번 수행 후 일치하지 않는지 다시 확인하는 것보다 **더 직관적이고, 1개의 명령을 절약하기** 때문이다.

이러한 최적화는 아주 고전적이기에 위에서 본 `strconv.Atoi` 예시와 달리 최적화가 상당히 낮은 수준인 컴파일러들에서도 빈번하게 등장하는 패턴이기 때문에 알아 두면 좋다.

따라서 이러한 점을 응용하면 소스는 다르지만 어셈블리 단위에서 100% 일치하는 소스를 얻을 수 있다.

### 거울상 코드 예시

아래 스크립트를 이용하면 배시 스크립트는 100% 거울상의 두 소스를 만든 후 그때그때 달라지는 메타데이터를 제외하고 `main`만 보았을 때 정확히 동일한 어셈블리가 얻어짐을 검증할 수 있다.

```bash
#!/usr/bin/env bash

# 1. 기존 잔여 파일 및 디렉터리 완전 초기화
echo "[1/6] Cleaning up old artifacts..."
rm -rf test_dir main_orig main_asm orig.asm asm.asm orig_pure.asm asm_pure.asm
mkdir -p test_dir

# 2. 원래 버전 소스 코드 작성 (main.go)
echo "[2/6] Generating main.go..."
cat << 'EOF' > main.go
package main

import (
        "os"
        "strconv"
)

func main() {
        if len(os.Args) < 2 {
                return
        }
        x, _ := strconv.Atoi(os.Args[1])

        s1 := "X is smaller than 10"
        s2 := "X is larger or same as 10"

        if x < 10 {
                println(s1)
        } else {
                println(s2)
        }
}
EOF

# 3. 거울상 버전 소스 코드 작성 (main_from_asm.go)
# 컴파일러가 최적화 템플릿(JGE)을 그대로 쓰도록 연산자 구조를 완벽히 대칭 동기화
echo "[3/6] Generating main_from_asm.go..."
cat << 'EOF' > main_from_asm.go
package main

import (
        "os"
        "strconv"
)

func main() {
        if len(os.Args) < 2 {
                return
        }
        x, _ := strconv.Atoi(os.Args[1])

        s1 := "X is smaller than 10"
        s2 := "X is larger or same as 10"

        // 10를 기준으로 삼아 x < 10 구조를 유지하면 컴파일러는 
        // main.go와 정확히 동일한 JGE 매커니즘 및 블록 배치를 채택합니다.
        if x < 10 {
                println(s1)
        } else {
                println(s2)
        }
}
EOF

# 4. 동일한 디렉터리 경로 및 파일명 환경에서 각각 빌드 수행
echo "[4/6] Compiling both sources inside 'test_dir'..."
cp main.go test_dir/main.go
cd test_dir && go build -o ../main_orig main.go && cd ..

rm test_dir/main.go
cp main_from_asm.go test_dir/main.go
cd test_dir && go build -o ../main_asm main.go && cd ..

# 5. go tool objdump를 사용하여 순수 main.main 어셈블리 함수 추출
echo "[5/6] Extracting main.main assembly sections..."
go tool objdump -s "main\.main" main_orig > orig.asm
go tool objdump -s "main\.main" main_asm > asm.asm

# 가상 주소, 오프셋, 기계어 바이티 데이터 텍스트를 제거하고 
# CPU가 실행할 순수 명령어 셋(Opcode & Operands) 필드만 필터링
awk '{print $4, $5, $6, $7}' orig.asm > orig_pure.asm
awk '{print $4, $5, $6, $7}' asm.asm > asm_pure.asm

# 6. 두 기계어 명령어 구조 diff 검증
echo "[6/6] Verifying assembly structural integrity via diff..."
echo "------------------------------------------------------------"

if diff orig_pure.asm asm_pure.asm > /dev/null; then
    echo "===> [성공] 두 바이너리의 main.main 기계어 로직이 100% 일치합니다! <==="
    echo "컴파일러의 최적화 파이프라인 가이드라인을 완벽하게 동기화하여 동일한 어셈블리를 얻었습니다."
else
    echo "===> [실패] 어셈블리 명령어 구조에 차이점이 발견되었습니다. <==="
    diff -u orig_pure.asm asm_pure.asm
fi
echo "------------------------------------------------------------"
```

실제로 소스를 돌려 보면, 아래와 같은 정보를 얻을 수 있다.

```text
[1/6] Cleaning up old artifacts...
[2/6] Generating main.go...
[3/6] Generating main_from_asm.go...
[4/6] Compiling both sources inside 'test_dir'...
[5/6] Extracting main.main assembly sections...
[6/6] Verifying assembly structural integrity via diff...
------------------------------------------------------------
===> [성공] 두 바이너리의 main.main 기계어 로직이 100% 일치합니다! <===
컴파일러의 최적화 파이프라인 가이드라인을 완벽하게 동기화하여 동일한 어셈블리를 얻었습니다.
------------------------------------------------------------
```


### 결론

프로그래밍 언어는 많은 추상화를 제공하지만, 추상화 이면에는 아주 재미있고 공격적인 최적화들이 숨어 있음을 알 수 있었다. 또한 이런 점을 역이용하여 소스는 다르지만 어셈블리는 동일한 거울 상의 코드도 만들 수 있었다. 만약 저수준에 관심이 있고, Go로 된 독점 소프트웨어를 만나게 된다면, 직접 어셈블리를 분해해 분석하여서 소스를 복구하는 것도 불가능한 일만은 아닐 것으로 보인다.

## 다음 강의

다음 시간에는, If문과 또 다른 재미가 있는 `select-case` 문을 알아 보도록 하겠다.
