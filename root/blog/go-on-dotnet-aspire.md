---
author: snowmerak
title: dotnet aspire에서 확장 가능하게 Go 서버 실행해보기
language: ko
---

## dotnet aspire?

dotnet aspire는 클라우드 네이티브 환경이 늘어남에 따라, 개발자들의 클라우드 네이티브 개발 및 구성을 돕기 위해 만들어진 도구입니다. 이 도구는 .NET 개발자들이 닷넷 프로젝트 및 다양한 클라우드 네이티브 인프라, 그리고 다른 언어로 된 서비스나 컨테이너 등을 쉽게 배포할 수 있게 해줍니다.

당연히 docker에서 k8s까지 출시 및 운영되며, 기존의 온프레미스 환경에서 상당히 많은 분야, 산업, 개발자들이 클라우드 네이티브 환경으로 이전하고 있고, 이전한 상태입니다. 이제는 성숙된 분야죠. 그렇기에 호스트 이름, 포트 구성, 방화벽, 메트릭 관리 등에 대해 기존의 불편함에 대해 설명할 필요가 없을 거라 생각합니다.

그래서 당장 위의 설명들로 미루어봐도 dotnet aspire가 뭔지 도저히 감이 안 잡힐 겁니다. 왜냐면, 이건 마이크로소프트도 정확한 정의를 내리진 않고 있습니다. 그래서 저도 별다른 정의를 내리진 않겠습니다. 다만, 이 글에서 제가 이해한 dotnet aspire의 기본적인 기능을 사용할 것이므로, 참고하여 본인만의 위치를 정하시면 될 것같습니다.

## 프로젝트 구성

### dotnet aspire 프로젝트 생성

만약 dotnet aspire 템플릿이 없다면, 템플릿부터 설치해야합니다. 다음 명령어로 템플릿을 설치합니다. 만약 .net이 없다면, 그건 본인이 설치해주세요.

```bash
dotnet new install Aspire.ProjectTemplates
```

그리고 적당한 폴더에서 새로운 솔루션을 생성합니다.

```bash
dotnet new sln
```

그 후, 솔루션 폴더에서 다음 명령어를 실행하여 aspire-apphost 템플릿의 프로젝트를 생성합니다.

```bash
dotnet new aspire-apphost -o AppHost
```

그럼 세팅을 위한 간단한 코드만 존재하는 aspire-apphost 프로젝트가 생성됩니다.

### Valkey 추가

그럼 간단하게 Valkey를 추가해보겠습니다.

바로 추가하기 전에, dotnet aspire는 community hosting을 통해 다양한 서드파티 솔루션을 제공합니다.  
당연히 valkey도 이러한 커뮤니티 호스팅의 지원을 받을 수 있으며, 다음 nuget 패키지를 통해 쉽게 이용할 수 있습니다.

```bash
dotnet add package Aspire.Hosting.Valkey
```

외에 다양한 통합된 호스팅을 제공하니, [이곳](https://learn.microsoft.com/ko-kr/dotnet/aspire/fundamentals/integrations-overview)에서 확인할 수 있습니다.  
다시 valkey로 돌아와서 AppHost 프로젝트에서 Program.cs 파일을 열어 다음과 같이 수정합니다.

```csharp
var builder = DistributedApplication.CreateBuilder(args);

var cache = builder.AddValkey("cache")
    .WithDataVolume(isReadOnly: false)
    .WithPersistence(interval: TimeSpan.FromMinutes(5),
        keysChangedThreshold: 100);

builder.Build().Run();
```

`cache`는 valkey 서비스를 빌드할 수 있는 정보를 가진 `IResourceBuilder` 인터페이스의 구현체입니다.  
`WithDataVolume`은 캐시 데이터를 저장할 볼륨을 생성하며, `WithPersistence`는 캐시 데이터를 지속적으로 저장할 수 있도록 합니다.  
여기까지 보면 `docker-compose`의 `volumes`와 비슷한 역할을 하는 것으로 보입니다.  
당연하게도 이는 여러분들도 어렵지 않게 만드실 수 있습니다.  
하지만 이 글의 범위를 넘어가므로 지금 이야기하지는 않겠습니다.

### Go 언어 에코 서버 생성

그럼 간단한 Go 언어 서버를 추가해보겠습니다.  
일단 솔루션 폴더에서 `go work init`을 통해 워크스페이스를 생성합니다.  
닷넷 개발자에겐 Go 워크스페이스는 솔루션과 유사한 거라 보시면 됩니다.

그리고 EchoServer라는 폴더를 만들고, 안으로 이동한 후 `go mod init EchoServer`를 실행합니다.  
이 명령어를 통해 Go 모듈을 생성합니다. 모듈은 닷넷 개발자에게 프로젝트와 유사한 것으로 인지하시면 됩니다.  
그리고 `main.go` 파일을 만들어 다음과 같이 작성합니다.

```go
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("PORT")
	log.Printf("Server started on %s", addr)

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		name := request.URL.Query().Get("name")
		writer.Write([]byte("Hello, " + name))
	})

	http.ListenAndServe(":"+addr, nil)
}
```

이 서버는 Aspire AppHost가 실행될 때, listening을 해야할 `PORT` 환경 변수를 주입해주면, 해당 포트를 읽어서 서버를 실행합니다.  
간단하게 `name` 쿼리를 받아서 `Hello, {name}`을 반환하는 서버입니다.

이제 이 서버를 dotnet aspire에 추가해보겠습니다.

### 에코 서버를 aspire에 추가하기

다시 Valkey를 추가했던 Aspire AppHost로 프로젝트로 이동해서 Go 언어를 위한 커뮤니티 호스팅을 추가합니다.

```bash
dotnet add package CommunityToolkit.Aspire.Hosting.Golang
```

그리고 Program.cs 파일을 열어 다음 구문을 추가합니다.

```csharp
var builder = DistributedApplication.CreateBuilder(args);

var cache = builder.AddValkey("cache")
    .WithDataVolume(isReadOnly: false)
    .WithPersistence(interval: TimeSpan.FromMinutes(5),
        keysChangedThreshold: 100);

var echoServer = builder.AddGolangApp("echo-server", "../EchoServer")
    .WithHttpEndpoint(port: 3000, env: "PORT")
    .WithExternalHttpEndpoints();

builder.Build().Run();
```

여기서 `echoServer`는 Go 언어 서버를 빌드할 수 있는 정보를 가진 `IResourceBuilder` 인터페이스의 구현체입니다.  
방금 추가한 `AddGolangApp` 메서드는 Go 언어 서버를 추가하기 위한 커스텀 호스트의 확장 메서드입니다.  
고정적으로 3000 포트를 사용하며, `PORT` 환경 변수를 주입해주는 것을 확인할 수 있습니다.  
마지막으로 `WithExternalHttpEndpoints`는 외부에서 접근할 수 있도록 하는 것입니다.

테스트를 위해 `http://localhost:3000/?name=world`로 접속해보시면, `Hello, world`가 출력되는 걸 확인하실 수 있을 겁니다.

하지만 현재 dotnet aspire에는 non-dotnet project에게 주어지는 무거운 패널티가 있습니다.  
그것은 바로...

## 프로젝트 확장

### 수평 확장은 그럼 어떻게 해?

현재 dotnet aspire는 `AddProject` 메서드로 추가된 닷넷 프로젝트에 대한 빌더에만 `WithReplica` 옵션을 제공합니다.  
하지만 Go 언어 호스트나 `AddContainer`같은 외부 프로젝트에 대해서는 이 옵션을 제공하지 않습니다.

그렇기에 별도의 load balancer나 reverse proxy를 사용해서 직접 구현해야 합니다.  
하지만 이러면 해당 reverse proxy가 SPOF가 될 수 있기에, reverse proxy는 `WithReplica` 옵션을 제공하는 것이 좋습니다.  
그렇다면 필연적으로 reverse proxy는 닷넷 프로젝트여야 합니다.

여태 이러한 문제에 대해서 nginx, trafik, 직접 구현 등의 방법을 써왔지만, 닷넷 프로젝트라는 제한이 걸리면 제 손에서는 당장 방법이 없었습니다.  
그래서 닷넷으로 구현된 reverse proxy를 찾아보았고, 다행히 [YARP](https://microsoft.github.io/reverse-proxy/)라는 선택지가 있었습니다.  
YARP는 닷넷으로 구현된 reverse proxy로, load balancer 역할도 할 수 있고, 다양한 기능을 제공하고 있었기에 좋은 선택이라고 판단했습니다.

그럼 이제 YARP를 추가해보겠습니다.

### YARP로 reverse proxy 구성

먼저 YARP를 사용하기 위한 프로젝트를 생성합니다.

```bash
dotnet new web -n ReverseProxy
```

그리고 프로젝트로 이동해서 YARP를 설치합니다.

```bash
dotnet add package Yarp.ReverseProxy --version 2.2.0
```

설치가 끝나면, Program.cs 파일을 열어 다음과 같이 작성합니다.

```csharp
using Yarp.ReverseProxy.Configuration;

var builder = WebApplication.CreateBuilder(args);

var routes = new List<RouteConfig>();
var clusters = new List<ClusterConfig>();

builder.Services.AddReverseProxy()
    .LoadFromMemory(routes, clusters);

var app = builder.Build();

app.MapReverseProxy();
app.Run(url: $"http://0.0.0.0:{Environment.GetEnvironmentVariable("PORT") ?? "5000"}");
```

이 코드는 YARP를 사용하기 위한 기본적인 코드입니다.  
`routes`는 reverse proxy가 사용할 라우트 정보를, `clusters`는 reverse proxy가 사용할 클러스터 정보를 담고 있습니다.  
이 정보들은 `LoadFromMemory` 메서드로 reverse proxy에 로드됩니다.  
마지막으로 `MapReverseProxy` 메서드를 사용하여 reverse proxy를 매핑하고 실행합니다.

그리고 실사용을 위해 aspire apphost 프로젝트에서 reverse proxy 프로젝트를 참조로 넣어주고, Program.cs 파일에 다음 구문을 추가 및 수정합니다.

```bash
dotnet add reference ../ReverseProxy
```

```csharp
var echoServer = builder.AddGolangApp("echo-server", "../EchoServer")
    .WithHttpEndpoint(env: "PORT");

var reverseProxy = builder.AddProject<Projects.ReverseProxy>("gateway")
    .WithReference(echoServer)
    .WithHttpEndpoint(port: 3000, env: "PORT", isProxied: true)
    .WithExternalHttpEndpoints();
```

이제 reverse proxy가 echo server를 참조할 수 있습니다.  
외부에서 들어오는 요청은 reverse proxy에서 받고 echo server로 넘겨주는 구조로 변경되고 있습니다.

### Reverse proxy 수정

일단은 reverse proxy에 할당된 프로젝트의 listening 주소를 변경해야합니다.  
`Properties/launchSettings.json` 파일 내부의 `applicationUrl`을 제거합니다.  
그리고 Program.cs 파일을 열어 아래와 같이 대대적으로 수정합니다.

```csharp
using Yarp.ReverseProxy.Configuration;

var builder = WebApplication.CreateBuilder(args);

var routes = new List<RouteConfig>
{
    new RouteConfig
    {
        ClusterId = "cluster-echo",
        RouteId = "route-echo",
        Match = new RouteMatch
        {
            Path = "/"
        }
    }
};

var echoServerAddr = Environment.GetEnvironmentVariable("services__echo-server__http__0") ?? "http://localhost:8080";

var clusters = new List<ClusterConfig>
{
    new ClusterConfig
    {
        ClusterId = "cluster-echo",
        Destinations = new Dictionary<string, DestinationConfig>
        {
            { "destination-echo", new DestinationConfig { Address = echoServerAddr } }
        }
    }
};

builder.Services.AddReverseProxy()
    .LoadFromMemory(routes, clusters);

var app = builder.Build();

app.MapReverseProxy();
app.Run(url: $"http://0.0.0.0:{Environment.GetEnvironmentVariable("PORT") ?? "5000"}");
```

먼저 `routes`와 `clusters`에 대한 정보를 수정합니다.  
각각 `echo-route`와 `echo-cluster`를 추가하여, echo server로 요청을 보내도록 설정합니다.  
그리고 echo server의 주소를 환경 변수로부터 읽어와서 사용하도록 수정합니다.

이 주소의 규칙은 `services__{service-name}__http__{index}`입니다.  
echo server의 경우, 서비스 이름이 `echo-server`이고, 단일 인스턴스이기에 인덱스로 `0`을 사용합니다.  
만약 asp .net core 서버를 추가한다면, `WithReplica`를 통해 여러 인스턴스가 생성될 수 있으므로 인덱스를 증가시켜 사용하면 됩니다.  
예외 처리되어 있는 `http://localhost:8080`은 아무런 뜻이 없는 쓰레기 값입니다.

이제 프로젝트를 실행하고, `http://localhost:3000/?name=world`로 접속해보시면, 여전히 `Hello, world`가 출력되는 걸 확인하실 수 있을 겁니다.

### 확장 아이디어

이제 dotnet aspire에 Go 서버를 추가하고, reverse proxy를 통해 요청을 전달하는 것을 확인했습니다.  
그러면 이제 이 과정을 programmatic하게 구현할 수 있도록 확장할 수 있을 겁니다.  
예를 들어, echo server에 대해서 서비스 이름 뒤에 넘버링을 추가하여 여러 인스턴스를 생성하고, reverse proxy에 대한 설정을 자동으로 추가할 수 있습니다.

aspire apphost 프로젝트의 Program.cs 파일에 reverse proxy와 echo server를 사용하는 코드를 다음과 같이 수정합니다.

```csharp
var reverseProxy = builder.AddProject<Projects.ReverseProxy>("gateway")
    .WithHttpEndpoint(port: 3000, env: "PORT", isProxied: true)
    .WithExternalHttpEndpoints();

for (var i = 0; i < 8; i++)
{
    var echoServer = builder.AddGolangApp($"echo-server-{i}", "../EchoServer")
        .WithHttpEndpoint(env: "PORT");
    reverseProxy.WithReference(echoServer);
}
```

그리고 reverse proxy 프로젝트의 Program.cs 파일을 다음과 같이 수정합니다.

```csharp
var echoServerDestinations = new Dictionary<string, DestinationConfig>();
for (var i = 0; i < 8; i++)
{
    echoServerDestinations[$"destination-{i}"] = new DestinationConfig
    {
        Address = Environment.GetEnvironmentVariable($"services__echo-server-{i}__http__0") ?? "http://localhost:8080"
    };
}

var clusters = new List<ClusterConfig>
{
    new ClusterConfig
    {
        ClusterId = "cluster-echo",
        Destinations = echoServerDestinations
    }
};
```

8개로 늘어간 echo server 인스턴스에 대해 목적지 설정을 추가합니다.  
이제 reverse proxy는 늘어난 echo server들에 대한 목적지 정보를 가지고, 요청을 전달해줄 수 있게 되었습니다.  
기존의 `http://localhost:3000/?name=world`로 접속해보시면, 여전히 `Hello, world`가 출력되는 걸 확인하실 수 있을 겁니다.

## 마치며

이 글에서는 dotnet aspire에 Go 서버를 추가하고, reverse proxy를 통해 요청을 전달하는 과정을 설명했습니다.  
다만 확장에 관해서는 아직 모두 작성하진 않았고, 환경 변수를 통해 좀 더 programmatic하게 구현할 수 있는 예제를 별도 레포에 작성해놓았습니다.  
자세한 프로젝트 구성과 코드는 [snowmerak/AspireStartPack](https://github.com/snowmerak/AspireStarterPack)을 참고해주세요.

저는 개인적으로 dotnet aspire가 docker compose의 대안으로써, 그리고 클라우드 배포 툴로써 자신만의 롤을 수행할 수 있다고 기대합니다.  
이미 docker compose나 k8s manifest를 생성하는 [제너레이터](https://prom3theu5.github.io/aspirational-manifests/generate-command.html)가 존재하여, 일반 개발자가 인프라 도구에 대한 접근성이 더 좋아지지 않았나 생각합니다.
