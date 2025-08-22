---
id: 208cfd8127284f07337b5f5002f3351d
author: hamori
title: 고루틴의 기본
description: Go 언어의 핵심인 고루틴에 대해 알아봅니다. 동시성, 경량성, 성능, GMP 모델 등 고루틴의 장점과 동작 원리를 상세히 설명합니다.
language: ko
date: 2025-08-22T13:28:15.766056383Z
path: /blog/posts/goroutine-basics-z3fbe209e
---

# Goroutine

Gopher들에게 golang의 장점을 이야기 해달라하면 자주 등장하는 **동시성(Concurrency)** 관련 글이 있습니다. 그 내용의 기반은 가볍고 간단하게 처리할 수 있는 **고루틴(goroutine)**입니다. 이에 대하여 간략하게 작성해보았습니다.

### **동시성(Concurrency) vs 병렬성(Parallelism)**

고루틴을 이해하기 전에, 자주 혼동되는 두 가지 개념을 먼저 짚고 넘어가려 합니다.

- 동시성: 동시성은 많은 일을 한 번에 처리하는 것에 관한 것입니다. 꼭 실제로 동시에 실행된다는 의미가 아니라, 여러 작업을 작은 단위로 나누고 번갈아 가며 실행함으로써 사용자가 보기에는 동시에 여러 작업이 처리되는 것처럼 보이게 하는 구조적, 논리적 개념입니다. 싱글 코어에서도 동시성은 가능합니다.
- 병렬성: 병렬성은 “여러개의 코어에서 여러개의 일을 동시에 처리하는것” 입니다. 말 그대로 병렬적으로 일을 진행하는 것이며, 다른 작업들을 동시에 실행합니다.

고루틴은 Go 런타임 스케줄러를 통해 동시성을 쉽게 구현하게 해주며, `GOMAXPROCS` 설정을 통해 병렬성까지 자연스럽게 활용합니다.

흔히 이용률이 높은 자바의 멀티쓰레드(Multi thread)는 병렬성의 대표 개념입니다.

# 고루틴은 왜 좋을까?

### 가볍다(lightweight)

생성비용이 다른 언어에 비해서 매우 낮습니다. 여기서 왜 golang은 적게 사용할까요? 라는 의문이 드는데 생성 위치가 Go런타임 내부에서 관리하기 때문입니다. 왜냐하면 위의 경량 논리 스레드 이기 때문입니다 OS쓰레드 단위보다 작고, 초기스택은 2KB정도의 크기를 필요로 하며 사용자의 구현에 따라 스택을 추가하여 동적으로 가변하기 때문입니다.

스택 단위로 관리하여 생성,제거가 매우 빠르고 저렴하게 처리가 가능하여 수백만개의 고루틴을 돌려도 부담스럽지 않은 처리가 가능합니다. 이로 인해 Goroutine은 런타임 스케쥴러 덕분에 OS커널 개입을 최소화 할 수 있습니다.

### 성능이 좋다(performance)

우선 Goroutine은 위의 설명처럼 OS커널 개입이 적어 사용자 수준(User-Level)에서 컨텍스트 스위칭을 할때 OS스레드 단위보다 비용이 저렴하여 빠르게 작업을 전환할 수 있습니다.

외에도 M:N모델을 이용하여 OS스레드에 할당하여 관리합니다. OS 쓰레드 풀을 만들어 많은 쓰레드가 필요없이 적은 쓰레드로도 처리가 가능합니다. 예를 들어 시스템 호출 과 같은 대기상태에 빠지면 Go런타임은 OS쓰레드에서 다른 고루틴을 실행하여 OS쓰레드는 쉬지 않고 효율적으로 CPU를 활용하여 빠른 처리가 가능합니다.

이로 인하여 Golang이 특히 I/O작업에서 다른 언어에 비해 높은 성능을 낼 수 있습니다.

### 간결하다(concise)

동시성이 필요한 경우 `go` 키워드 하나로 함수를 쉽게 처리할 수 있는것도 큰 장점입니다.

`Mutex` , `Semaphore` 등 복잡한 Lock을 이용해야 하며, Lock을 이용하면 필수적으로 고려야할 데드락(DeadLock) 상태를 고려할 수 밖에 없어 개발이전 설계단계에서 부터 복잡한 단계가 필요해집니다.

Goroutine은 "메모리를 공유하여 통신하지 말고, 통신하여 메모리를 공유하라"는 철학에 따라 `채널(Channel)`을 통한 데이터 전달을 권장하며 `SELECT` 는 채널(Channel)과 결합하여 데이터가 준비된 채널부터 처리할 수 있게 해주는 기능까지 지원합니다. 또한, `sync.WaitGroup`을 이용하면 여러 고루틴이 모두 끝날 때까지 간단하게 기다릴 수 있어 작업 흐름을 쉽게 관리할 수 있습니다. 이러한 도구들 덕분에 쓰레드 간의 데이터 경쟁 문제를 방지하고 보다 안전하게 동시성 처리가 가능합니다.

또한, 컨텍스트(context)를 이용하여 이를 사용자 수준(User-Level)에서 생명주기, 취소, 타임아웃, 데드라인, 요청범위를 제어 할 수 있어 어느정도의 안정성을 보장할 수 있습니다.

# Goroutine의 병렬 작업(GOMAXPROCS)

goroutine의 동시성이 좋은점을 말했지만 병렬은 지원하지 않나? 라는 의문이 드실겁니다. 최근 CPU의 코어의수는 과거와 다르게 두자리 수가 넘어가며, 가정용 PC또한 코어가 적지않은 수가 들어가 있기 때문입니다.

하지만 Goroutine은 병렬작업까지 진행합니다 그것이 `GOMAXPROCS`입니다.

`GOMAXPROCS` 를 설정하지 않으면 버전별로 다르게 설정됩니다.

1. 1.5 이전: 기본값 1, 1 이상 필요시 `runtime.GOMAXPOCS(runtime.NumCPU())` 와 같은 방식으로 설정이 필수
2. 1.5 ~ 1.24: 사용 가능한 모든 논리 코어수로 변경되었습니다. 이때부터 개발자가 크게 제약을 필요한 경우가 아니면 설정할 필요가 없습니다
3. 1.25: 컨테이너 환경에서 유명한 언어답게, linux상의 cGroup을 확인하여 컨테이너에 설정된 `CPU제한` 을 확인합니다.

   그러면 논리 코어수가 10개이고, CPU제한값이 5일 경우 `GOMAXPROCS` 는 더 낮은 수인 5로 설정합니다.

1.25 의 수정은 굉장히 큰 수정점을 가집니다. 바로 컨테이너 환경에서의 언어 활용도가 올라갔기 때문입니다. 이로 인해서 불필요한 스레드 생성과, 컨텍스트 스위칭을 줄여 CPU 스로틀링(throttling)을 방지할 수 있게 되었습니다.

```go
package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

func exe(name int, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Goroutine %d: 시작\n", name)
	time.Sleep(10 * time.Millisecond) // 작업 시뮬레이션을 위한 지연
	fmt.Printf("Goroutine %d: 시작\n", name)
}

func main() {
	runtime.GOMAXPROCS(2) // CPU 코어 2개만 사용
	wg := sync.WaitGroup();
  goroutineCount := 10
	wg.Add(goroutineCount)

	for i := 0; i < goroutineCount; i++ {
		go exe(i, &wg)
	}

	fmt.Println("모든 goroutine이 끝날 때까지 대기합니다...")
	wg.Wait()
	fmt.Println("모든 작업이 완료되었습니다.")

}

```

# Goroutine의 스케쥴러 (M:N모델)

앞의 내용인 M:N모델을 이용하여 OS스레드에 할당하여 관리합니다 부분에서 조금 더 구체적으로 들어가면 goroutine GMP모델이 있습니다.

- G (Goroutine): Go에서 실행되는 가장 작은 작업 단위
- M (Machine): OS 쓰레드 (실제 작업 위치)
- P (Processor): Go런타임이 관리하는 논리적인 프로세스

입니다. P는 추가적으로 로컬 실행 큐(Local Run Queue)를 가지며, 할당된 G를 M에 배정하는 스케쥴러 역활을 합니다. 간단하게 goroutine은

GMP의 동작과정은 아래와 같습니다

1. G(Gorutine)가 생성되면 P(Processor)의 로컬 실행 큐에 할당을 진행합니다
2. P(Processor)는 로컬 실행 큐에 있는 G(Goroutine)을 M(Machine)에 할당합니다.
3. M(Machine)은 G(Goroutine)의 상태인 block, complete, preempted을 반환합니다.
4. Work-Stealing (작업 훔치기): 만약 P의 로컬 실행 큐가 비게 될 경우, 다른 P는 글로벌 큐를 확인합니다. 그곳에도 G(Goroutine)이 없다면 다른 로컬 P(Processsor)의 작업을 훔쳐와 모든 M이 쉬지 않고 동작하도록 만듭니다.
5. 시스템 콜 처리 (Blocking): G(Goroutine)가 실행중 Block이 발생할 경우 M(Machine)은 대기상태가 되는데, 이때 P(Processor)는 Block이 된 M(Machine)과 분리하여 다른 M(Machine)과 결합하여 다음 G(Goroutine)을 실행합니다. 이때 I/O 작업도중 대기시간에서도 CPU낭비가 없습니다.
6. 하나의 G(Goroutine)이 오래 선점(preempted)할 경우 다른 G(Goroutine)에게 실행 기회를 줍니다.

Golang은 GC(Garbage Collector)또한 Goroutine위에서 실행되어, 애플리케이션의 실행을 최소한으로 중단시키면서(STW) 병렬적으로 메모리를 정리할 수 있어 시스템 자원을 효율적으로 사용합니다.

마지막으로 Golang은 언어의 강한 장점중 하나이며, 이외에도 많으니 많은 개발자 분들이 고랭을 즐기셨으면 좋겠습니다.

감사합니다.
