---
author: snowmerak
title: Go 언어에서의 MLDSA와 MLKEM 사용기
language: ko
---

## 개요

### 배경

꽤 오래 전부터 양자 컴퓨터의 빠른 연산은 기존 암호화 체계에 위협으로 인식되었습니다. 기존의 RSA나 ECC같은 경우엔 양자 컴퓨터의 이러한 연산 능력으로 인해 해독될 가능성이 있기 때문입니다. 하지만 수년 전부터 양자 컴퓨터라는 개념이 가시화되기 시작하면서, 그에 대한 대안들이 연구 및 개발되기 시작되었고, NIST에서 PQC(양자 내성 암호화) 표준화를 진행해왔습니다.

### MLDSA와 MLKEM

끝내 NIST는 2024년 8월에 CRYSTALS-Kyber와 CRYSTALS-Dilithium을 기반으로 하는 MLKEM과 MLDSA를 표준으로 채택했습니다. 두 알고리즘은 MLWE(Module Learning with Errors)라는 문제를 기반으로 동작합니다. 이러한 형식을 저희는 격자 기반 암호화라고 합니다.

격자 기반 암호화는 이름 그대로 격자 상에서 수학 문제의 어려움에 기반한 암호화 시스템입니다. 저도 이에 대한 깊이 있는 수학적 지식은 없으나 한줄로 정리하면 `모듈 격자에서 노이즈가 있는 선형 방정식을 푸는 문제`라고 합니다. 얼마나 어려운 지는 감이 안 잡히지만, 이러한 문제는 양자 컴퓨터로도 풀 수 없을 정도로 어렵다고 합니다.

## MLDSA

그럼 먼저 MLDSA에 대해 알아보겠습니다.

### 구성

MLDSA는 이름에서 보이다시피 비대칭 서명 알고리즘으로 총 다음 2단계를 거칩니다.

1. 서명 생성: 개인키를 사용하여 메시지에 대한 서명을 생성
2. 서명 검증: 공개키를 이용하여 생성된 서명의 유효성을 확인

그리고 MLDSA는 다음 3가지 특성이 있습니다.

1. strong existential unforgeability: 한 서명과 공개키로 다른 유효한 서명을 생성할 수 없습니다.
2. chosen message attack: 어떤 메시지에 대한 서명으로도 공개키를 가지고 새로운 유효한 서명을 생성할 수 없습니다.
3. side-channel attack: 서명 시 계속 새로운 랜덤 값과 메시지에서 파생된 의사 랜덤 값을 사용하여 보안이 높습니다.
4. domain separation: 서로 다른 매개변수에 대해 서로 다른 seed를 사용하도록 하여 반복적인 보안 문제를 예방합니다.

### 코드

그럼 간단한 Go 언어 예제 코드를 보여드리겠습니다.  
이 예제에서는 [cloudflare/circl](github.com/cloudflare/circl)의 mldsa를 사용했습니다.

```go
package main

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
)

func main() {
    // mldsa44 스펙으로 키를 생성합니다.
	pub, priv, err := mldsa44.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	message := []byte("Hello, World!")

    // 서명을 생성합니다.
    // 한가지 주의할 점은 24년 12월 22일 현재 버전 기준으로 crypto.Hash(0)이 아니면 에러가 발생합니다.
	signature, err := priv.Sign(rand.Reader, message, crypto.Hash(0))
	if err != nil {
		panic(err)
	}

	encodedSignature := base64.URLEncoding.EncodeToString(signature)
	fmt.Println(len(encodedSignature), encodedSignature)

    // 퍼블릭 키의 scheme을 호출해서 검증을 합니다.
	ok := pub.Scheme().Verify(pub, message, signature, nil)
	fmt.Println(ok)
}
```

```
3228 oaSaOA-...
true
```

> 서명 값은 너무 길어서 생략했습니다. 전문이 보고 싶으시다면, [playground](https://go.dev/play/p/nqbUnKpFW3j)에서 실행해보세요.

base64로 인코딩 했다지만 3228 바이트가 나오는 건 조금 부담스럽긴 할 것입니다.  
조만간 저희는 양자 컴퓨터에 대항하는 서명으로 이 크기를 주고 받아야 할 지 모른다고 생각하니 조금 부담스럽긴 하네요..

## MLKEM

### 구성

MLKEM은 키 캡슐화 메커니즘(Key Encapsulation Mechanism)입니다. KEM은 공개키 암호화 방식을 사용하여 두 당사자 간의 공유키를 생성할 수 있도록 하는 알고리즘입니다. MLKEM의 키 교환 메커니즘은 다음 과정을 거칩니다.

1. 키 캡슐화: 송신자는 수신자의 공개키를 사용하여 암호화된 메시지(cipher text)와 공유기(shared key)를 생성합니다. 이 암호화된 메시지를 초기에 수신자에게 전달하여 이용하게끔 합니다.
2. 키 캡슐 해제: 수신자는 자신의 개인키를 사용하여 암호화된 메시지에서 공유키를 추출합니다. 

MLKEM에는 총 3가지 패러미터가 존재합니다. MLKEM-512, MLKEM-768, MLKEM-1024가 존재하며 적을 수록 작은 키와 암호화 텍스트가 나오며, 클 수록 더 긴 키와 암호화 텍스트가 나오며, 보안 수준이 더 높습니다.

### 코드

MLKEM은 go 1.24에서 추가될 예정이라 현 시점에서 사용할 수 있는 go 1.24rc1을 사용하였습니다.

```go
package main

import (
	"crypto/mlkem"
	"encoding/base64"
	"fmt"
)

func main() {
    // 수신자의 PrivateKey를 생성합니다.
	receiverKey, err := mlkem.GenerateKey1024()
	if err != nil {
		panic(err)
	}

    // MLKEM에선 PublicKey가 아니라 EncapsulationKey라는 용어를 사용합니다.
	receiverPubKey := receiverKey.EncapsulationKey()

    // 간단하게 EncapsulationKey의 Bytes()와 NewEncapsulationKeyX로 키를 추출하고 다시 사용할 수 있음을 보여주기 위해 복제했습니다.
    // 물론 현실에서 사용하면 이 과정이 텍스트로 공개되어 있던 수신자의 EncapsulationKey키를 전송자가 객체로 만드는 과정이라 보시면 되겠습니다.
	clonedReceiverPubKey, err := mlkem.NewEncapsulationKey1024(receiverPubKey.Bytes())
	if err != nil {
		panic(err)
	}

    // Encapsulate로 전송자가 암호화 텍스트와 공유키를 생성합니다.
	cipherText, SenderSharedKey := clonedReceiverPubKey.Encapsulate()

    // 수신자의 개인키를 저장하고 꺼내는 걸 보여드리려고 일부러 복제했습니다.
	clonedReceiverKey, err := mlkem.NewDecapsulationKey1024(receiverKey.Bytes())
	if err != nil {
		panic(err)
	}

    // 수신자는 개인키를 사용하여 암호화 텍스트를 Decapsulate하여 또 다른 공유키를 생성합니다.
	sharedKeyReceiver, err := clonedReceiverKey.Decapsulate(cipherText)
	if err != nil {
		panic(err)
	}

	fmt.Println(base64.StdEncoding.EncodeToString(SenderSharedKey))
	fmt.Println(base64.StdEncoding.EncodeToString(sharedKeyReceiver))
}
```

```sh
Q1ciS818WFHTK7D4MTvsQvciMTGF+dSGqMllOxW80ew=
Q1ciS818WFHTK7D4MTvsQvciMTGF+dSGqMllOxW80ew=
```

결과적으로 같은 크기의 공유키가 생성되는 걸 확인할 수 있습니다!

이 코드는 [플레이그라운드](https://go.dev/play/p/n_cxNp435Qn?v=gotip)에서도 확인할 수 있습니다.

## 결론

각 알고리즘의 스펙, 보안 수준이나 개인 키, 공개 키, 서명이나 암호문의 크기는 다음처럼 정리할 수 있습니다. 각각 PQC라는 이름에 부끄럽지 않게 큼직한 사이즈를 자랑합니다.

|알고리즘|NIST 보안 수준|개인 키 크기|공개 키 크기|서명/암호문 크기|
|---|---|---|---|---|
|ML-DSA-44|2|2,560|1,312|2,420|
|ML-DSA-65|3|4,032|1,952|3,309|
|ML-DSA-87|5|4,896|2,592|4,627|
|ML-KEM-512|1|1,632|800|768|
|ML-KEM-768|3|2,400|1,184|1,088|
|ML-KEM-1024|5|3,168|1,568|1,568|

이들 알고리즘에 의해 저희는 양자 컴퓨터 상에서도 충분히 안전한 인터넷을 사용할 수 있길 기대합니다만, 상대적으로 커진 키와 서명/암호문 크기에 의해 더 많은 연산이 있을 것은 피할 수 없어 보입니다.

그래도 고 언어는 각 알고리즘이 효과적으로 구현되어 있기에, 적절한 위치에서 여러분들의 보안을 지키는 데에 적극적으로 활용되길 기대합니다!
