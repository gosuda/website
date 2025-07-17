---
author: Yunjin Lee
title: Go 인터페이스는 상속이 아니다
---

# 개요

Go 인터페이스는 동일한 인자와 반환값을 갖는 함수를 여러 구조체에서 쉽게 가질 수 있게 하지만, java의 extends 키워드처럼 그 내부 함수의 동작까지 적절히 연장하고 오버라이드하는 방식과는 다릅니다. Go의 구성적 코드 재사용을 제대로 이해해야만 상속과 헷갈리지 않겠지만, 처음부터 이론적으로 완벽한 이해를 하는 것은 어렵습니다. 실수하기 좋은 시나리오와 함께 알아봅시다.


## 자주 하는 실수
초심자 분들은 다음과 같은 실수를 하실 수 있습니다.
```go
package main
import (
	"fmt"
	"strings"
)

type Fruits interface {
	GetBrix() float64
	GetName() string
	SetLabel()
	GetLabel(string) string
	PrintAll()
}

type Apple struct {
	Label string
	Name  string
	Brix  float64
}

type Watermelon struct {
	Label string
	Name  string
	Brix  float64
}

func (a *Apple) PrintAll() {
	fmt.Printf("Fruit: %s, Label: %s, Brix: %v\n", a.Name, a.Label, a.Brix)
}

const (
	NO_LABEL = "EMPTY LABEL"
)

func (a *Apple) SetLabel(lbl string) {
	a.Brix 	= 14.5;
	a.Name 	= "apple";
	lbl_lower := strings.ToLower(lbl)
	if strings.Contains(lbl_lower, a.Name) {
		fmt.Println("Succeed: Label was ", lbl)
		a.Label = lbl;
	} else {
		fmt.Println("Failed: Label was ", lbl)
		a.Label = NO_LABEL;
	}
}

func (w *Watermelon) SetLabel(lbl string) {
	w.Brix = 10;
	w.Name = "watermelon";
	lbl_lower := strings.ToLower(lbl)
	if strings.Contains(lbl_lower, w.Name) {
		w.Label = lbl;
	} else {
		w.Label = NO_LABEL;
	}
}

func main() {
	fmt.Println("Inheritance test #1")
	apple := new(Apple)
	watermelon := apple
	apple.SetLabel("Apple_1")
	fmt.Println("Apple, before copied to Watermelon")
	apple.PrintAll()
	watermelon.SetLabel("WaterMelon_2")
	fmt.Println("Apple, after copied to Watermelon")
	apple.PrintAll()
	fmt.Println("Watermelon, which inherited Apple's Method")
	watermelon.PrintAll()
}
```

이러한 코드는 Go가 전통적인 상속을 따른다고 착각하면 문제가 없어 보입니다. 하지만 이것의 출력 결과는 다음과 같습니다.

```
Inheritance test #1
Succeed: Label was  Apple_1
Apple, before copied to Watermelon
Fruit: apple, Label: Apple_1, Brix: 14.5
Failed: Label was  WaterMelon_2
Apple, after copied to Watermelon
Fruit: apple, Label: EMPTY LABEL, Brix: 14.5
Watermelon, which inherited Apple's Method
Fruit: apple, Label: EMPTY LABEL, Brix: 14.5
```

여기서 Go의 동작은 다만 명확해집니다. 
```go
watermelon := apple
```
이 코드는 전혀 Apple을 그대로 Watermelon 클래스로 변환하지 않습니다.
다만 watermelon은 apple에 대한 포인터일 뿐입니다.

여기서 다시 강조하지만, **Go는 전통적인 상속 개념을 따르지 않습니다.**

이러한 오해를 한 상태에서 코드를 짠다면 무의미한 포인터 생성, 예기치 못한 타 구조체를 위한 함수 복사 등의 치명적 오류가 생깁니다.

그렇다면 모범적인 코드는 어떠할까요?


## Go언어에서 적절한 예시
```go
package main
import (
	"fmt"
	"strings"
)

type Fruits interface {
	GetBrix() float64
	GetName() string
	SetLabel()
	GetLabel(string) string
	PrintAll()
}

type BaseFruit struct {
	Name  string
	Brix  float64
}

type Apple struct {
	Label string
	Fruit BaseFruit
}

type Watermelon struct {
	Label string
	Fruit BaseFruit

}

func (b *BaseFruit) PrintAll() {
	fmt.Printf("Fruit: %s, Brix: %v\n", b.Name, b.Brix)
}


const (
	NO_LABEL = "EMPTY LABEL"
)

func (a *Apple) SetLabel(lbl string) {
	a.Fruit.Brix 	= 14.5;
	a.Fruit.Name 	= "apple";
	lbl_lower := strings.ToLower(lbl)
	if strings.Contains(lbl_lower, a.Fruit.Name) {
		fmt.Println("Succeed: Label was ", lbl)
		a.Label = lbl;
	} else {
		fmt.Println("Failed: Label was ", lbl)
		a.Label = NO_LABEL;
	}
	fmt.Printf("Fruit %s label set to %s\n", a.Fruit.Name, a.Label);
	a.Fruit.PrintAll()
}

func (w *Watermelon) SetLabel(lbl string) {
	w.Fruit.Brix = 10;
	w.Fruit.Name = "Watermelon";
	lbl_lower := strings.ToLower(lbl)
	if strings.Contains(lbl_lower, w.Fruit.Name) {
		w.Label = lbl;
	} else {
		w.Label = NO_LABEL;
	}
	fmt.Printf("Fruit %s label set to %s\n", w.Fruit.Name, w.Label);
	w.Fruit.PrintAll()
}

func main() {
	apple := new(Apple)
	watermelon := new(Watermelon)
	apple.SetLabel("Apple_1")
	watermelon.SetLabel("WaterMelon_2")
}
```
그러나, Go에서도 상속**처럼** 보이게 하는 것은 가능합니다.
익명 임베딩이라는 예시입니다. 이것은 내부 구조체를 이름 없는 구조체로 선언하면 가능합니다.
이러한 경우에는 **하위 구조체의 필드들을 명시 없이 사용해도** 그대로 접근이 가능합니다.
이렇게 하위 구조체의 필드를 상위 구조체로 **승격**하는 패턴을 이용하면 경우에 따라 가독성의 향상이 가능합니다. 그러나 하위 구조체를 명시적으로 보여줘야 하는 경우에는 사용하지 않기를 권장합니다.
```go
package main
import (
	"fmt"
	"strings"
)

type Fruits interface {
	GetBrix() float64
	GetName() string
	SetLabel()
	GetLabel(string) string
	PrintAll()
}

type BaseFruit struct {
	Name  string
	Brix  float64
}

type Apple struct {
	Label string
	BaseFruit
}

type Watermelon struct {
	Label string
	BaseFruit

}

func (b *BaseFruit) PrintAll() {
	fmt.Printf("Fruit: %s, Brix: %v\n", b.Name, b.Brix)
}


const (
	NO_LABEL = "EMPTY LABEL"
)

func (a *Apple) SetLabel(lbl string) {
	a.Brix 	= 14.5;
	a.Name 	= "apple";
	lbl_lower := strings.ToLower(lbl)
	if strings.Contains(lbl_lower, a.Name) {
		fmt.Println("Succeed: Label was ", lbl)
		a.Label = lbl;
	} else {
		fmt.Println("Failed: Label was ", lbl)
		a.Label = NO_LABEL;
	}
	fmt.Printf("Fruit %s label set to %s\n", a.Name, a.Label);
	a.PrintAll()
}

func (w *Watermelon) SetLabel(lbl string) {
	w.Brix = 10;
	w.Name = "Watermelon";
	lbl_lower := strings.ToLower(lbl)
	if strings.Contains(lbl_lower, w.Name) {
		w.Label = lbl;
	} else {
		w.Label = NO_LABEL;
	}
	fmt.Printf("Fruit %s label set to %s\n", w.Name, w.Label);
	w.PrintAll()
}

func main() {
	apple := new(Apple)
	watermelon := new(Watermelon)
	apple.SetLabel("Apple_1")
	watermelon.SetLabel("WaterMelon_2")
}
```


이 예시에서는 이러한 차이점이 있습니다.
```go
w.PrintAll() // w.Friut.PrintAll()이 아닌, 이름 없는 구조체를 통한 자동 승격 호출
두 예제 모두 중요한 지점은 이러합니다.
- main은 간소하게, 함수는 기능 별로
- 다른 구조체라면 다른 객체를
- 공유가 필요할 경우 내부 구조체를 사용

이와 같은 프로그래밍 철학에 어떠한 이점이 있을까요?

## 이점

- 공유가 필요한 메서드와 아닌 것의 구별 명확
- 개별 구조체, 메서드에 책임 소재 분리
- 필요한 기능 명세에 따라 구조적으로 분리된 코드


처음에는 Go언어는 전통적인 OOP와 달라 익숙하지 않을 수 있으나, 익숙해지면 명시적인 프로그래밍이 가능합니다.

## 요약
- 책임 소재를 고립시키자
- 구조체 단위로 세부적으로 나누자
- 메서드는 자바의 추상 클래스처럼 이해해선 안 된다
- 명시적이고 구체적인 프로그래밍을 하자
Go언어는 전통적인 OOP 모델보다 간단명료하고 개별적으로 다뤄져야 합니다. 확장적이게 프로그래밍하기보다 단계적이고 구조적으로 분리하여 작성하도록 합시다.

