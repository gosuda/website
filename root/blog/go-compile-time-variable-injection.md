---
id: 1872edeaa3c5fc09a1eba338cf2b01b1
author: wHoIsDReAmer
title: Go의 Compile-Time Variable Injection
description: Go 빌드 시 `ldflags -X`를 활용하여 컴파일 타임에 변수 값을 주입하는 방법을 알아보세요.
language: ko
date: 2025-06-01T15:56:46.352281271Z
path: /blog/posts/compile-time-variable-injection-of-go-z5420556c
---

## 배경

이번에 회사 프로젝트 작업을 하면서 .env 없이 컴파일 타임 때 변수를 주입하는 방법이 있나 찾아보았다.
이렇게 하면 `.env`나 여러 콘피그 파일 없이 단일 파일만 있으면 되기 때문에 굉장히 편리할 거라 생각했다.

일단 처음에 찾아봤던 건 `.env`를 바탕으로 상수 파일을 생성해주는 `go:generate` CLI 같은 걸 찾아봤는데, 없었다.

근데 그럴거면 그냥 constants를 수정하면 되지 번거롭게 gen까지 할 필요는 없었고, 러스트에서 `include_bytes!` 매크로 같이 컴파일 타임 때 값을 주입할 수 있는 방법을 찾았다.

## 컴파일 타임 변수 주입

사용법은 간단하다.

```go
package main

import "fmt"

var Foo string

func main() {
    fmt.Println(Foo)
}
```

위와 같은 코드가 있다고 하면, 빌드 시 다음과 같이 주입할 수 있다.

```bash
go build -ldflags="-X 'main.Foo=bar'"
```

ldflag로 인자를 넘겨주면 빌드 시 컴파일 타임에 `Foo` 변수에 `bar` 값이 주입된다. 간단하다.

만약 프로젝트에서 같은 이름의 패키지나 변수가 여러 곳에 있다면 **패키지 경로를 명시하면** 된다.

```go
// github.com/myproject/config/config.go
package config

var Version string

// github.com/myproject/internal/config/config.go  
package config

var BuildTime string
```

이런 경우 다음과 같이 구분해서 주입할 수 있다:

```bash
go build -ldflags="-X 'github.com/myproject/config.Version=v1.0.0' -X 'github.com/myproject/internal/config.BuildTime=2025-05-27'"
```

같은 패키지에 여러 변수가 있는 경우에도 그냥 추가적인 인자를 전달해주면 된다.

```bash
go build -ldflags="-X 'main.Version=v1.0.0' -X 'main.BuildTime=2025-05-27' -X 'main.GitCommit=abc123'"
```

특징은 다음과 같다:

- **변수가 public이 아니어도 됨**: 소문자로 시작하는 변수도 주입 가능하다
- **string 타입만 지원**: 다른 타입은 지원하지 않는다

## 참조

- [https://stackoverflow.com/questions/47509272/how-to-set-package-variable-using-ldflags-x-in-golang-build](https://stackoverflow.com/questions/47509272/how-to-set-package-variable-using-ldflags-x-in-golang-build)
