---
id: e10df2108115b462a5ac2a289080a901
author: iwanhae
title: Go 그리고 OpenAPI 생태계
description: Go 언어로 OpenAPI 기반 API 개발 시 사용할 수 있는 라이브러리와 전략을 소개합니다.
language: ko
date: 2024-10-18T06:07:43.092628696Z
path: /blog/posts/go-and-the-openapi-ecosystem-zd5a8e472
---

### 서론

Go 언어로 Production Backend 서버를 개발하다 보면 거의 대부분의 개발자들이 가장 처음으로 만나는 난제 중 하나는 다음과 같습니다.

> API 문서화, 어떻게 하지...?

이에 대하여 조금만 찾아보면 OpenAPI 스펙에 맞는 문서를 작성하는 것이 이롭다는 사실을 깨닫게 되고, 자연스럽게 OpenAPI와 연동되는 라이브러리를 찾게 됩니다. 하지만 이러한 결정을 세워도 그 다음 문제가 존재합니다.

> OpenAPI 관련 라이브러리 많은데.. 뭐 써야하지...?

이 문서는 이러한 상황을 경험하고 계신 Go 입문자들을 위하여 작성한 간략한 라이브러리 소개글 입니다. 2024년 말 기준으로 작성된 문서이며, 언어 생태계는 항상 유동적으로 바뀌는 만큼 참고하면서 항상 최신 근황도 살펴보는것을 추천드립니다.

### OpenAPI 를 대하는 라이브러리들의 전략

이미 알고계신 부분이겠지만, OpenAPI 는 REST API를 명확하게 정의하고 문서화하기 위한 스펙입니다. API의 엔드포인트, 요청, 응답 형식 등을 YAML 또는 JSON 형식으로 정의하여 개발자들 뿐만 아니라 프론트단, 백엔드단 코드 생성을 자동화하여 무의미한 반복을 줄여주고 소소한 휴먼에러들을 줄여주는데 큰 도움을 줍니다.

이러한 OpenAPI 를 프로젝트와 자연스럽게 결합시키기 위해 Go 생태계의 라이브러리들은 크게 다음 세가지 전략을 취합니다.

#### 1. Go 주석을 OpenAPI 스펙 문서로 조합

OpenAPI 에 맞춰서 API 를 개발할때 까다로운점 중 하나는 실제 문서와 해당 문서를 구현한 코드가 별도의 파일로 전혀 다른위치에 존재하다보니, 코드를 업데이트 했는데 문서를 업데이트 안했던가 문서는 업데이트 했는데 코드를 업데이트 하지 못하는 상황이 생각보다 잦다는 것입니다.

간단한 예시를 들어보면 

1. `./internal/server/user.go` 라는 파일 속에서 API 에 대한 로직을 수정했는데 
2. 실제 문서는 `./openapi3.yaml` 에 존재하고, 이에대한 변경을 실수로 깜빡할 수 있습니다. 
3. 이러한 변경사항에 대한 이슈를 인지하지 못하고 Pull Request 를 날리고 동료들에게 리뷰를 받을 경우
4. 리뷰어들 또한 `./openapi3.yaml` 에 대한 변경사항이 눈에 보이지 않기 때문에 
   API 스펙은 그대로인데 실제 API 구현체는 변경이 되어버리는 불상사가 발생할 수 있습니다.

Go 주석의 형태로 API 문서를 작성하면 이러한 문제를 어느 정도 해소할 수 있습니다. 코드와 문서가 한 곳에 모여 있기 때문에, 코드를 수정하면서 주석도 함께 업데이트할 수 있습니다. 이러한 주석을 기반으로 자동으로 OpenAPI 스펙 문서를 생성해주는 도구들이 존재합니다.

대표적인 프로젝트로는 [Swag](https://github.com/swaggo/swag)가 있습니다. Swag는 Go 코드의 주석을 파싱하여 OpenAPI 2 형식의 문서를 생성해 줍니다. 사용 방법은 간단합니다. 핸들러 함수 위에 각 라이브러리에서 정한 형식에 맞게 주석을 작성하면 됩니다.

```go
// @Summary 유저 생성
// @Description 새로운 유저를 생성합니다.
// @Tags Users
// @Accept json
// @Produce json
// @Param user body models.User true "유저 정보"
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Router /users [post]
func CreateUser(c *gin.Context) {
    // ...
}
```

이렇게 주석을 작성하면 Swag 라는 CLI 는 이 주석들을 파싱해서 OpenAPI 2 문서를 생성합니다. 일반적으로 CI 과정에 이러한 작업이 행해지며, 생성된 OpenAPI 스펙의 문서는 Git Repository, 최종 빌드 결과물, 별도의 외부 API 문서 관리 시스템에 배포되어 다른 프로젝트와의 협업때 사용되게 됩니다.

**장점:**

- 주석이 코드와 함께 있기 때문에 **실제 코드와 문서의 형상이 달라질 가능성이 줄어듭니다.**
- 별도의 도구나 복잡한 설정 없이 주석만으로 **간편하고 자유롭게 문서화**를 할 수 있습니다.
- 주석이 실제 API 로직에 영향을 주진 않기때문에, **문서로 공개하기 부담스러운 임시 기능을 추가**하기 좋습니다.

**단점:**

- 주석의 라인수가 많아지면서 단일 코드 파일에 대한 **가독성이 떨어질 수 있습니다.**
- **주석의 형태**로 모든 API 스펙을 **표현하기 어려울 수 있습니다.**
- 문서가 코드를 강제하는것은 아니기때문에 OpenAPI **문서와 실제 로직이 일치한다는 보장을 할 수 없습니다.**

## 2. OpenAPI 스펙의 문서로 Go 코드를 생성

Single source of Truth (SSOT) 를 Go 코드가 아니라 문서쪽에 두는 방법도 존재합니다. 바로 OpenAPI 스펙을 먼저 정의하고, 정의된 내용을 기반으로 Go 코드를 생성하는 방식입니다. API 스펙이 곧 코드를 생성해주기 때문에 개발 문화적으로 API 설계를 먼저 하는것을 강제할 수 있으며 개발 순서적으로 API 스펙을 정의하는것이 가장먼저 시작이 되기때문에 개발이 완료되고 나서야 놓친부분을 인지하고 API 스펙 변경과 함께 전체 코드가 수정되는 불상사를 조기에 방지할 수 있는 강점을 가지고 있습니다.

이 방식을 채택하는 대표적인 프로젝트로는 [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) 과 [OpenAPI Generator](https://github.com/OpenAPITools/openapi-generator/tree/master/samples/server/petstore/go-echo-server) 가 존재합니다. 사용법은 간단합니다. 

1. OpenAPI 스펙에 맞게 yaml 혹은 json 문서를 작성하고 
2. CLI 를 실행하면
3. 그에 대응되는 Go stub 코드가 생성됩니다.
4. 이제 이 stub 이 사용할 수 있도록 개별 API 에 대한 세부 로직만 직접 구현하면 됩니다.

다음은 oapi-codegen 에서 생성해주는 코드의 예시입니다.

```go
// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// ...
	// Returns all pets
	// (GET /pets)
	FindPets(ctx context.Context, request FindPetsRequestObject) (FindPetsResponseObject, error)
	// ...
}
```

위 interface 를 매개로 oapi-codegen 이 생성해준 코드는 query parameters, header, body 파싱 및 Validation 등의 로직을 수행하고 interface 에 선언된 적절한 method 를 호출해주는 구조입니다. 사용자는 위 interface 에 대한 구현체만 구현하면 API 구현에 필요한 작업이 완료되게 됩니다.

**장점:**

- **스펙이 먼저 나오고 개발이 진행**되기 떄문에 여러 팀에서 협업하는경우 **업무를 병렬적으로 진행**하기 유리합니다.
- **반복성 노가다로 작업**하던 부분에 대한 코드가 **자동으로 생성**되기 때문에, **업무 효율이 상승**하면서도 **디버깅에 여전히 유리**합니다.
- **문서와 코드의 형상이 항상 일치**하다는것을 보장하기 쉽습니다.

**단점:**

- OpenAPI 스펙 자체에 무지한 상태일경우 **초기 러닝커브가 다소 존재**합니다.
- API 를 핸들링하는 코드의 형상이 프로젝트에 의해서 자동으로 생성되기 떄문에 **커스터마이징이 필요한경우 대응하기 어려울 수 있습니다.**

> 저자의 코멘트.
> 2024년 10월 기준 OpenAPI Generator 가 생성한 Go 코드는 API 로직뿐만 아니라 전체 프로젝트 형상을 강제하며 프로젝트의 구조가 경직되어있어 실제 Production 환경에 필요한 다양한 기능들을 추가하기에는 부적합한 형태의 코드를 생성하고 있습니다. 이 방식을 채택하시는 분들은 oapi-codegen 을 사용하시는것을 적극적으로 권장드립니다. 저자는, oapi-codege + echo + StrictServerInterface 를 사용하고 있습니다.



## 3. Go 코드로 OpenAPI 스펙 문서를 생성

수십, 수백명의 사람들이 같은 서버에대해서 개발을 진행하다보면 필연적으로 발생하는 이슈가 개별 API 별로 통일성이 깨질 수 있다는 것입니다. 직관적인 예시로 100개가 넘어가는 API Endpoint 에 대한 명세를 하나의 OpenAPI yaml 파일에 선언할경우 해당 파일은 1만 라인이 넘어가는 괴물이 되어있을 것이고 새로운 API Endpoint 를 선언하면서 필연적으로 같은 모델을 중복해서 선언한다던가 몇몇 필드를 누락한다던가, 컨벤션에 맞지 않는 Path 네이밍이 탄생한다던가와 같은 전체저인 API 의 통일성이 깨지기 시작하게 됩니다.

이러한 이슈를 해결하기위해 OpenAPI yaml 을 관리하는 Owner 를 따로 둔다던가, Linter 를 개발해서 CI 과정중에 자동으로 잡아낼 수 있도록 조치를 취할수도 있겠지만 Go 언어로 Domain-specific language (DSL) 를 정의하여 모든 API 가 일관적인 통일성을 가질 수 있도록 강제할 수 있습니다.

이러한 기법을 사용하는 대표적인 프로젝트가 Kubernetes 이며 (별도 라이브러리 없이 자체적으로 구축), [go-restful](https://github.com/emicklei/go-restful), [goa](https://goa.design/) 등의 프로젝트를 사용해서 사용해볼수도 있습니다. 다음은 `goa` 의 사용 예시입니다.

```go
var _ = Service("user", func() {
    Method("create", func() {
        Payload(UserPayload)
        Result(User)
        HTTP(func() {
            POST("/users")
            Response(StatusOK)
        })
    })
})
```

위와같이 컴파일 가능한 Go 코드를 작성하면 `POST /users` API 에 대한 구현과 문서에 대한 정의가 동시에 완료되는 강점을 얻을 수 있습니다. 

**장점:**

- 코드로부터 모든게 나오기때문에 **프로젝트 전체에 대한 API 일관성을 가지고가기가 쉽습니다**.
- Go 의 강타입 시스템을 활용하여, OpenAPI3 의 모든 기능을 활용했을때보다 **더 정확하고 논란이 없는 스펙**을 얻을 수 있습니다.

**단점:**

- **각 프레임워크에서 정의한 DSL 을 익혀야**하며, **기존 코드에 적용하기는 어려울 수 있습니다.**
- 프레임워크에서 제안한 규칙을 강제로 따라야 하므로 **자유도 및 유연성이 떨어질 수 있습니다.**

**마무리하며**

각 방법은 장단점이 있으며, 프로젝트의 요구사항과 팀의 선호도에 따라 적합한 방법을 선택하는 것이 중요합니다. 언제나 제일 중요한것은 어떤 방식을 사용하는게 좋느냐가 아니라, 현재 자신이 처한 상황에 가장 적합한 솔루션은 무엇인지 가치판단을 수행하고 개발 생산성을 높게 가져가 **빠른 퇴근**과 **흡족스러운 워라벨**을 즐기는 것입니다.

현재 2024년 10월을 기준으로 글을 작성하긴 했지만 Go와 OpenAPI 생태계는 지속적으로 발전하고 있으므로, 이 글을 읽는 시점간의 간격을 고려하여 각 라이브러리들 및 프로젝트들의 근황과 그들의 변경된 장단점도 지속적으로 팔로업하시길 바랍니다.

행복한 Go 라이프 되세요~ 😘
