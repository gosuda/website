---
id: 548504c5a0b4857763527603f3e1059e
author: snowmerak
title: Any is not Any, but Any is Any
description: Go 언어의 `any` 타입과 내부 `EFace` 구조를 상세히 분석하고, `reflect` 패키지 및 타입 단언/스위치를 활용한 동적 타입 처리 방법을 설명합니다.
language: ko
date: 2025-09-30T15:12:53.689280321Z
path: /blog/posts/any-is-not-any-but-any-is-any-z6161856d
---

## Any

Go 언어에서 any 타입은 모든 타입을 담을 수 있는 마법같은 인터페이스, `interface{}`의 별칭입니다. 즉, 어떤 타입이든 any 타입 변수에 할당할 수 있습니다.

```go
var a any
a = 42          // int
a = "hello"     // string
a = true        // bool
a = 3.14        // float64
```

이처럼 any 타입은 다양한 타입의 값을 담을 수 있어 유연한 코드를 작성할 수 있게 해줍니다. 하지만, any 타입을 사용할 때는 몇 가지 주의할 점이 있습니다.

### Type Assertion

any 타입 변수에서 실제 값을 사용하려면 타입 단언(type assertion)을 사용해야 합니다. 타입 단언은 any 타입 변수가 특정 타입임을 컴파일러에게 알려주는 역할을 합니다.

```go
var a any = "hello"
str, ok := a.(string)
if ok {
    fmt.Println("String value:", str)
} else {
    fmt.Println("Not a string")
}
```

위 예제에서 `a.(string)`은 `a`가 string 타입임을 단언합니다. 만약 `a`가 string 타입이 아니라면, `ok`는 false가 되고, 프로그램이 패닉에 빠지는 것을 방지할 수 있습니다.

### Type Switch

타입 스위치(type switch)는 any 타입 변수의 실제 타입에 따라 다른 동작을 수행할 수 있게 해줍니다. 이는 여러 타입을 처리해야 할 때 유용합니다.

```go
var a any = 42
switch v := a.(type) {
case int:
    fmt.Println("Integer:", v)
case string:
    fmt.Println("String:", v)
case bool:
    fmt.Println("Boolean:", v)
default:
    fmt.Println("Unknown type")
}
```

위 예제에서 `a.(type)`은 `a`의 실제 타입을 검사하고, 해당 타입에 맞는 케이스를 실행합니다.

### Reflection

Go 언어의 reflect 패키지를 사용하면 any 타입 변수의 타입과 값을 런타임에 동적으로 검사하고 조작할 수 있습니다. 이는 복잡한 데이터 구조를 다룰 때 유용합니다.

```go
import (
    "fmt"
    "reflect"
)

var a any = 3.14
v := reflect.ValueOf(a)
fmt.Println("Type:", v.Type())
fmt.Println("Value:", v.Float())
```

위 예제에서 reflect 패키지를 사용하여 `a`의 타입과 값을 런타임에 확인할 수 있습니다.

## EFace

Go 언어에서 인터페이스는 두 가지 주요 형태로 나뉩니다: EFace(Empty Interface)와 IFace(Interface with Methods)입니다. 이 중 EFace가 저희가 지금까지 다룬 any 타입과 동일한 개념입니다.

### 구조

EFace는 Go 언어의 런타임 내에서만 존재하는 특별한 구조체이기에 여기서는 미믹(mimic)을 만들어서 다루겠습니다.

```go
type runtimeTypeMimic struct {
	Size_       uintptr
	PtrBytes    uintptr // number of (prefix) bytes in the type that can contain pointers
	Hash        uint32  // hash of type; avoids computation in hash tables
	TFlag       uint8   // extra type information flags
	Align_      uint8   // alignment of variable with this type
	FieldAlign_ uint8   // alignment of struct field with this type
	Kind_       uint8   // enumeration for C
	Equal       func(unsafe.Pointer, unsafe.Pointer) bool
	GCData      *byte
	Str         int32 // string form
	PtrToThis   int32 // type for pointer to this type, may be zero
}

type eFaceMimic struct {
	Type *runtimeTypeMimic
	Data unsafe.Pointer
}
```

EFace는 두 개의 필드를 가지고 있습니다:
- `Type`: 값의 실제 타입에 대한 메타데이터를 담고 있는 포인터입니다.
- `Data`: 실제 값을 담고 있는 포인터입니다.

그리고 EFace는 내부적으로 `runtimeType` 구조체를 통해 타입 정보를 관리합니다. 이 구조체는 타입의 크기, 정렬, 해시 값 등 다양한 메타데이터를 포함하고 있습니다.

이 안에서 저희가 주목해야 할 부분은 `Kind_`와 `Hash`, `Equal` 필드입니다. `Kind_`는 타입의 종류를 나타내며, `Hash`와 `Equal`은 해시 테이블에서 타입을 비교하고 해시 값을 계산하는 데 사용됩니다.

### 포장

이제 이 EFace를 래핑해서 좀 더 편하게 흑마술을 쓸 수 있게 만들어봅시다.

```go
type TypeInfo struct {
	Hash  uint32
	TFlag uint8
	Kind  uint8
	Equal func(unsafe.Pointer, unsafe.Pointer) bool
	This  unsafe.Pointer
}

func GetTypeInfo(v any) TypeInfo {
	eface := *(*eFaceMimic)(unsafe.Pointer(&v))
	return TypeInfo{
		Hash:  eface.Type.Hash,
		TFlag: eface.Type.TFlag,
		Kind:  eface.Type.Kind_,
		Equal: eface.Type.Equal,
		This:  eface.Data,
	}
}
```

위 함수는 any 타입 변수를 받아서 내부의 EFace 구조체를 추출하고, 그 안에서 타입 정보를 꺼내어 `TypeInfo` 구조체로 반환합니다. 이제 이 `TypeInfo`를 사용하여 타입의 해시 값, 종류, 비교 함수 등을 쉽게 접근할 수 있습니다.

### 활용

활용은 정말 쉽습니다. 그냥 `GetTypeInfo` 함수에 any 타입 변수를 넘겨주면 됩니다.

```go
func main() {
	i := 42
	eface := GetTypeInfo(i)
	fmt.Printf("%+v\n", eface)

	f := 3.14
	eface2 := GetTypeInfo(f)
	fmt.Printf("%+v\n", eface2)

	// Compare the two values using the Equal function from the type info
	log.Println(eface.Equal(eface.This, eface.This))
	log.Println(eface2.Equal(eface2.This, eface2.This))
	log.Println(eface2.Equal(eface.This, eface2.This))
	log.Println(eface.Equal(eface2.This, eface.This))
}
```

위 예제에서는 정수와 부동 소수점 값을 `GetTypeInfo` 함수에 넘겨서 각각의 타입 정보를 출력하고, `Equal` 함수를 사용하여 값들을 비교합니다. 이를 통해 좀 더 빠르게 타입 정보를 얻고, 타입을 비교할 수 있습니다.

또한 이 방식은 구조체와 포인터, 인터페이스 등에도 동일하게 사용할 수 있으며, 적지 않게 발생하는 이슈인 타입 있는 nil(typed nil) 문제도 해결할 수 있습니다. `Hash`는 타입이 있으면 할당되어 있겠지만, `This`는 nil이면 0이기 때문입니다.

## 결론

EFace는 현재 Go 언어에서 any 타입을 다루는 데 있어 중요한 역할을 합니다. 이를 통해 다양한 타입의 값을 유연하고 빠르게 처리할 수 있으며, 타입 정보를 자세히 얻고 활용할 수 있습니다. 다만, 이는 런타임 내에 숨겨진 기능으로 언어 스펙 변경의 영향을 받을 수 있으므로 주의해야합니다.