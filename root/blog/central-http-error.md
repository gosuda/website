---
author: snowmerak
title: 더욱 덜 귀찮게 HTTP 에러를 처리하기 + RFC7807
language: ko
---

## 개요

Go 언어에서 http api를 생성할 때, 가장 귀찮은 건 에러 처리입니다. 대표적으로 이런 코드가 있습니다.

```go
func(w http.ResponseWriter, r *http.Request) {
    err := doSomething()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        log.Printf("error occurred: %v", err)
        return
    }
    // ...
}
```

API가 몇개 되지 않는 다면, 이런 식으로 작성해도 딱히 불편한 건 없을 겁니다. 다만, API 수가 늘어나고 내부 로직이 복잡해질 수록 세가지가 거슬리게 됩니다.

1. 적절한 에러 코드 반환
2. 많은 결과 로그 작성 수
3. 명확한 에러 메시지 전송

## 본론

### 적절한 에러 코드 반환

물론 1번, 적절한 에러 코드 반환은 제 개인적인 불만 사항이긴 합니다. 숙련된 개발자라면 적절한 코드를 찾아서 매번 잘 찾아 넣을 겁니다만, 저도 그렇고 아직 미숙한 개발자들은 로직이 복잡해지고, 회수가 많아질 수록 적합한 에러 코드를 규칙적으로 쓰는 것에 어려움을 겪을 수 있습니다. 이에 대해 여러 방법이 있을 거고, 가장 대표적으로 미리 API 로직 흐름을 설계한 후 적절한 에러를 반환하도록 코드를 작성하는 것이 있을 겁니다. ~~그렇게 하십시오~~

하지만 이는 IDE(혹은 Language Server)의 도움을 받는 인간 개발자에게 최적의 방법으로 보이진 않습니다. 또한 REST API 자체가 에러 코드에 담긴 의미를 최대한 활용하는 만큼, 또 다른 방식을 제안할 수 있을 겁니다. `HttpError`라는 에러(`error`) 인터페이스 구현체를 새로 만들어, `StatusCode`와 `Message`를 저장하게 합니다. 그리고 다음과 같은 헬퍼 함수를 제공합니다.

```go
err := httperror.BadRequest("wrong format")
```

`BadRequest` 헬퍼 함수는 `StatusCode`로 400, `Message`를 인자로 받은 값으로 설정한 `HttpError`를 반환할 겁니다. 이 외에도 당연히 `NotImplement`, `ServiceUnavailable`, `Unauthorized`, `PaymentRequired` 등의 헬퍼 함수를 자동 완성 기능으로 조회 및 추가할 수 있을 겁니다. 이는 준비된 설계서를 매번 확인하는 것보다 빠르며, 매번 숫자로 에러 코드를 입력하는 것보다 안정적일 겁니다. ~~`http.StatusCode` 상수에 다 있다구요? 쉿~~

### 많은 결과 로그 작성 수

에러 발생 시 당연히 로그를 남기게 됩니다. API가 호출되고, 요청이 성공했는지, 실패했는지에 대해 로그를 남길 때에 시작부터 모든 예상 종료 지점에 로그를 남기는 건 작성할 코드 수가 많아집니다. 이를 핸들러 자체를 한번 감싸면서, 중앙에서 관리할 수 있게 됩니다.

다음은 `chi` 라우터를 감싸는 예시입니다.

```go
package chiwrap

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gosuda/httpwrap/httperror"
)

type Router struct {
	router      chi.Router
	errCallback func(err error)
}

func NewRouter(errCallback func(err error)) *Router {
	if errCallback == nil {
		errCallback = func(err error) {}
	}
	return &Router{
		router:      chi.NewRouter(),
		errCallback: errCallback,
	}
}

type HandlerFunc func(writer http.ResponseWriter, request *http.Request) error

func (r *Router) Get(pattern string, handler HandlerFunc) {
	r.router.Get(pattern, func(writer http.ResponseWriter, request *http.Request) {
		if err := handler(writer, request); err != nil {
			he := &httperror.HttpError{}
			switch errors.As(err, &he) {
			case true:
				http.Error(writer, he.Message, he.Code)
			case false:
				http.Error(writer, err.Error(), http.StatusInternalServerError)
			}
			r.errCallback(err)
		}
	})
}
```

라우터 구조체는 `chi.Router`를 내부에 가지고 있어서, `chi.Router`의 기능을 그대로 사용하게 구성됩니다. `Get` 메서드를 보시면, 방금 위에서 제안드린 헬퍼 함수가 반환하는 `HttpError` 구조체가 반환되었는지 체크 후 적절히 반환하고, `error`일 경우엔 일괄적으로 에러 콜백 함수로 전달하게 됩니다. 이 콜백은 생성자를 통해 입력 받습니다.

다음은 이 패키지를 활용하여 작성한 코드입니다.

```go
package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gosuda/httpwrap/httperror"
	"github.com/gosuda/httpwrap/wrapper/chiwrap"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

	r := chiwrap.NewRouter(func(err error) {
		log.Printf("Router log test: Error occured: %v", err)
	})
	r.Get("/echo", func(writer http.ResponseWriter, request *http.Request) error {
		name := request.URL.Query().Get("name")
		if name == "" {
			return httperror.BadRequest("name is required")
		}

		writer.Write([]byte("Hello " + name))
		return nil
	})

	svr := http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	go func() {
		if err := svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

    <-ctx.Done()
    svr.Shutdown(context.Background())
}
```

어떤가요? 단순히 `HttpError`만 헬퍼 함수로 만들어 반환하면, 상위 스코프에서 적절한 에러 코드와 메시지로 응답을 돌려주면서 구현 서비스 마다 적절한 로그를 남길 수 있도록 콜백을 등록하여 처리할 수 있습니다. 추가적으로 필요하다면 확장해서, `RequestID` 등을 사용하여 상세한 로깅이 가능할 겁니다.

### 명확한 에러 메시지 전송

이를 위한 문서로 RFC7807이 있습니다. RFC7807은 주로 다음과 같은 요소를 정의하여 사용합니다.

- `type`: 에러 유형을 식별하는 URI. 주로 에러에 대해 설명하는 문서입니다.
- `title`: 어떤 에러인지에 대한 한줄 설명입니다.
- `status`: HTTP Status Code와 동일합니다.
- `detail`: 해당 에러에 대해 사람이 읽을 수 있는 자세한 설명입니다.
- `instance`: 에러가 발생한 URI입니다. 예를 들어 `GET /user/info`에서 에러가 발생하였다면, `/user/info`가 그 값이 될 겁니다.
- `extensions`: JSON Object 형태로 구성되는 에러를 설명하기 위한 부차적인 요소입니다.
  - 예를 들어, `BadRequest`일 경우에는 사용자의 입력이 포함될 수 있습니다.
  - 혹은 `TooManyRequest`일 경우에는 가장 최근 요청 시점을 포함할 수도 있습니다.

이를 쉽게 사용하기 위해 `HttpError`와 같은 위치인 `httperror` 패키지에 새로운 파일을 생성하고 `RFC7807Error` 구조체를 생성하고, 메서드 체이닝 패턴으로 생성할 수 있게 합니다.

```go
func NewRFC7807Error(status int, title, detail string) *RFC7807Error {
	return &RFC7807Error{
		Type:   "about:blank", // Default type as per RFC7807
		Title:  title,
		Status: status,
		Detail: detail,
	}
}

func BadRequestProblem(detail string, title ...string) *RFC7807Error {
	t := "Bad Request"
	if len(title) > 0 && title[0] != "" {
		t = title[0]
	}
	return NewRFC7807Error(http.StatusBadRequest, t, detail)
}

func (p *RFC7807Error) WithType(typeURI string) *RFC7807Error { ... }
func (p *RFC7807Error) WithInstance(instance string) *RFC7807Error { ... }
func (p *RFC7807Error) WithExtension(key string, value interface{}) *RFC7807Error { ... }
```

`Type`의 `"about:blank"`는 기본값입니다. 없는 페이지를 의미합니다. 아래는 잘못된 요청에 대한 에러 생성 예제입니다.

```go
problem := httperror.BadRequestProblem("invalid user id format", "Bad User Input")

problem = problem.WithType("https://example.com/errors/validation")
         .WithInstance("/api/users/abc")
         .WithExtension("invalid_field", "user_id")
         .WithExtension("expected_format", "numeric")
```

간단한 메서드 체이닝으로 사용자에 대한 구조화된 에러 메시지를 생성할 수 있습니다. 또한 위에 먼저 작성한 중앙화된 라우터를 이용하기 위해 다음 메서드를 지원할 수 있습니다.

```go
func (p *RFC7807Error) ToHttpError() *HttpError {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		// If marshaling fails, fall back to just using the detail
		return New(p.Status, p.Detail)
	}
	return New(p.Status, string(jsonBytes))
}
```

이를 그대로 사용하여 위의 예제를 수정하면 이렇게 됩니다.

```go
package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gosuda/httpwrap/httperror"
	"github.com/gosuda/httpwrap/wrapper/chiwrap"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

	r := chiwrap.NewRouter(func(err error) {
		log.Printf("Router log test: Error occured: %v", err)
	})
	r.Get("/echo", func(writer http.ResponseWriter, request *http.Request) error {
		name := request.URL.Query().Get("name")
		if name == "" {
			return httperror.BadRequestProblem("name is required", "Bad User Input").
                WithType("https://example.com/errors/validation").
                WithInstance("/api/echo").
                WithExtension("invalid_field", "name").
                WithExtension("expected_format", "string").
                WithExtension("actual_value", name).
                ToHttpError()
		}

		writer.Write([]byte("Hello " + name))
		return nil
	})

	svr := http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	go func() {
		if err := svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

    <-ctx.Done()
    svr.Shutdown(context.Background())
}
```

## 결론

이렇게 중앙화된 라우터를 사용하여 에러를 처리하면, 매번 에러 코드를 확인하고, 적절한 에러 메시지를 작성하는 것에 대한 부담을 줄일 수 있습니다. 또한 RFC7807을 활용하여 구조화된 에러 메시지를 제공함으로써, 클라이언트가 에러를 이해하고 처리하는 데 도움을 줄 수 있습니다. 이러한 방법을 통해 Go 언어로 작성된 HTTP API의 에러 처리를 더욱 간편하고 일관되게 만들 수 있습니다.

해당 글의 코드는 [gosuda/httpwrap](https://github.com/gosuda/httpwrap) 레포지토리에서 확인하실 수 있습니다.
