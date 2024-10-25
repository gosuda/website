---
id: 974f36e12dfcece861bfda1af9303980
author: Yunjin Lee
title: Go에서 Tk로 GUI 만들기
description: Go 언어로 Tcl/Tk를 이용해 간단하고 효과적인 GUI 애플리케이션을 만드는 방법과 SVG, PNG, ICO 이미지 처리 기법을 배우세요.
language: ko
date: 2024-10-22T11:51:27.614782351Z
path: /blog/posts/creating-a-gui-with-tk-in-go-z004dd008
lang_canonical:
    ko: https://blog.naver.com/bugaku/223629101405
---

파이썬에는 Tkinter 와 같은 GUI 라이브러리가 기본적으로 내장되어 있습니다.
최근에 Go 언어에서도 Tcl/Tk를 사용할 수 있도록 [CGo-Free, Cross Platform Tk 라이브러리](https://pkg.go.dev/modernc.org/tk9.0)가 개발되었습니다. 오늘은 그 기초적인 사용법을 살펴보겠습니다.

## Hello, Tk 만들기

먼저 간단한 'Hello, TK!' 예제로 시작해보겠습니다.

```go
package main

import tk "modernc.org/tk9.0"

func main() {
    tk.Pack(
        tk.TButton(
            tk.Txt("Hello, TK!"),
            tk.Command(func() {
                tk.Destroy(tk.App)
            })),
        tk.Ipadx(10), tk.Ipady(5), tk.Padx(15), tk.Pady(10),
    )
    tk.App.Wait()
}
```

![hello-tk 실행 결과](/assets/images/go-tk-helloworld-svg-etc/go-tk-hello.png)

---

위 예제 코드와 실행 결과를 자세히 살펴보겠습니다.

파이썬의 Tk를 사용해본 경험이 있는 분이라면, 창 안에 위젯이 패킹되거나 창 하위에 직접 위젯이 패킹되는 구조를 이해하실 것입니다. 위젯의 종류에 따라 라벨 등이 그 안에 포함됩니다.

*Ipadx와 Ipady는 Internal padding의 약자로, 내부 위젯들의 여백을 조절합니다. 이 예제에서는 버튼의 여백이 조정됩니다.*

이 라이브러리에는 Window 구조체가 있으며, App이라는 변수가 최상위 창을 관리합니다. 이는 라이브러리 내부에 미리 정의되어 있습니다. 따라서 tk.App.Wait()를 종료하는 tk.App.Destroy() 함수가 최상위 창을 닫는 역할을 합니다.

이제 GitLab의 _examples 폴더에 있는 몇 가지 예제를 살펴보겠습니다.

## SVG 파일 처리하기

다음은 SVG 파일을 라벨 위젯에 표시하는 예제입니다.

```go
package main

import . "modernc.org/tk9.0"

// https://en.wikipedia.org/wiki/SVG
const svg = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg width="391" height="391" viewBox="-70.5 -70.5 391 391" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
<rect fill="#fff" stroke="#000" x="-70" y="-70" width="390" height="390"/>
<g opacity="0.8">
    <rect x="25" y="25" width="200" height="200" fill="lime" stroke-width="4" stroke="pink" />
    <circle cx="125" cy="125" r="75" fill="orange" />
    <polyline points="50,150 50,200 200,200 200,100" stroke="red" stroke-width="4" fill="none" />
    <line x1="50" y1="50" x2="200" y2="200" stroke="blue" stroke-width="4" />
</g>
</svg>`

func main() {
    Pack(Label(Image(NewPhoto(Data(svg)))),
        TExit(),
        Padx("1m"), Pady("2m"), Ipadx("1m"), Ipady("1m"))
    App.Center().Wait()
}
```

![go-tk-svg 실행 결과](/assets/images/go-tk-helloworld-svg-etc/go-tk-svg.png)

이 라이브러리에서 SVG를 처리하는 방법은 다음과 같습니다:

1. SVG 파일의 내용을 문자열로 읽어들입니다(또는 위 예시처럼 직접 포함시킵니다).
2. 이 내용을 Data 함수에 전달하여 옵션이 포함된 문자열로 변환합니다(-data 옵션).
3. 변환된 바이트값은 NewPhoto 함수로 전달되어 Tcl/Tk 이미지를 표현하는 Img 구조체 포인터를 반환합니다.
4. Image 함수를 통과하면서 Img 구조체 포인터는 -Image 옵션이 추가된 문자열로 변환됩니다.
5. 구조체 RAW 값을 담은 문자열로 변환하는 이유는 Label 위젯 생성을 위해서입니다.

ICO와 PNG 파일도 비슷한 방식으로 처리됩니다.

## PNG 파일 처리하기

```go
package main

import _ "embed"
import . "modernc.org/tk9.0"

//go:embed gopher.png
var gopher []byte

func main() {
    Pack(Label(Image(NewPhoto(Data(gopher)))),
        TExit(),
        Padx("1m"), Pady("2m"), Ipadx("1m"), Ipady("1m"))
    App.Center().Wait()
}
```

![go-tk-png 실행결과](/assets/images/go-tk-helloworld-svg-etc/go-tk-png.png)

PNG 파일 처리 과정은 다음과 같습니다:

1. 임베딩된 gopher.png를 옵션이 포함된 문자열 타입으로 변환합니다.
2. NewPhoto 함수를 통해 *Img 타입으로 변환합니다.
3. Image 함수를 거쳐 RAW 문자열로 변환된 후, 라벨 위젯으로 생성됩니다.

ICO 파일도 동일한 방식으로 처리되며, SVG 포맷과의 차이점은 Data 함수 내부 처리 방식뿐입니다.

여기서 "옵션이 포함된 문자열"의 정체를 살펴보겠습니다:

```go
type rawOption string
```

앞서 언급한 옵션이 포함된 문자열은 단순히 포맷팅된 문자열에 불과합니다.

```go
func (w *Window) optionString(_ *Window) string {
    return w.String()
}
```

optionString 메서드는 Window 포인터에 대한 메서드로, 문자열을 반환합니다.

마지막으로 Data 함수의 내부 구조를 간단히 살펴보겠습니다:

```go
func Data(val any) Opt {
    switch x := val.(type) {
    case []byte:
        switch {
        case bytes.HasPrefix(x, pngSig):
            // ok
        case bytes.HasPrefix(x, icoSig):
            b := bytes.NewBuffer(x)
            img, err := ico.Decode(bytes.NewReader(x))
            if err != nil {
                fail(err)
                return rawOption("")
            }

            b.Reset()
            if err := png.Encode(b, img); err != nil {
                fail(err)
                return rawOption("")
            }

            val = b.Bytes()
        }
    }
    return rawOption(fmt.Sprintf(`-data %s`, optionString(val)))
}
```

코드를 보면 ICO나 PNG 파일의 경우 인코딩/디코딩 과정이 필요합니다.
그 외의 경우에는 특별한 변환 없이 바이트형으로 변환된 문자열에 -data 옵션만 추가하여 Data 함수의 결과물임을 표시합니다.


### 메뉴 위젯을 이용하여 이미지 불러오기

앞서 연습한 PNG,ICO 불러오기 예제에 메뉴 위젯을 추가하면 필요한 이미지를 불러와서 표시하는 애플리케이션을 만들어 볼 수 있습니다.

먼저, 간단한 메뉴 위젯 예제를 보겠습니다.
```go
package main

import (
	"fmt"
	. "modernc.org/tk9.0"
	"runtime"
)

func main() {
	menubar := Menu()

	fileMenu := menubar.Menu() //메뉴바에 파일 메뉴 열을 생성
	fileMenu.AddCommand(Lbl("New"), Underline(0), Accelerator("Ctrl+N")) //1행 생성, New 레이블을 가지고 Ctrl+N 단축키를 가진 라벨을 추가, 인덱스 0
	fileMenu.AddCommand(Lbl("Open..."), Underline(0), Accelerator("Ctrl+O"), Command(func() { GetOpenFile() })) //앞의 것과 같으나 Command 옵션이 추가됨(해당 커맨드를 클릭하면 GetOpenFile() 함수가 실행됨.
    //인덱스 1에 해당
    //GetOpenFile 함수는 파일 불러오기 다이얼로그.
    //Underline(0)는 어느 위치에 밑줄을 그을지 선택, New와 Open...에서는 0이므로 각각 N과 O
	Bind(App, "<Control-o>", Command(func() { fileMenu.Invoke(1) })) //최상위 앱에서 Ctrl+O (Open 버튼의 단축키) 를 누르면 바로 파일 메뉴에서 인덱스 1에 해당하는 Open 커맨드를 실행하도록 바인드
	fileMenu.AddCommand(Lbl("Save As..."), Underline(5)) //A에 밑줄이 그어지게 됨. //인덱스 2
	fileMenu.AddSeparator() //Separator 추가, 인덱스 3
	fileMenu.AddCommand(Lbl("Exit"), Underline(1), Accelerator("Ctrl+Q"), ExitHandler()) //애플리케이션 종료 핸들러인 ExitHandler를 옵션으로 받음, 이것으로 애플리케이션을 종료. 인덱스 4
	Bind(App, "<Control-q>", Command(func() { fileMenu.Invoke(4) }))
	menubar.AddCascade(Lbl("File"), Underline(0), Mnu(fileMenu)) //파일 메뉴 레이블 설정


	App.WmTitle(fmt.Sprintf("%s on %s", App.WmTitle(""), runtime.GOOS)) //프로그램 실행파일 이름 on 운영체제
	App.Configure(Mnu(menubar), Width("8c"), Height("6c")).Wait() //앱 실행 시의 창 가로/세로 길이를 설정
}
```

![go-tk-메뉴바-실행결과](/assets/images/go-tk-helloworld-svg-etc/go-tk-menu-bar.png)
![go-tk-다이얼로그](/assets/images/go-tk-helloworld-svg-etc/go-tk-dialogue.png)

이 예제에서는 메뉴 바/메뉴 생성, 글자 강조, Command 옵션, 단축키 바인딩, 그리고 애플리케이션 창의 초기 크기를 설정해 보았습니다.
이제, 단순히 GetOpenFile로 지정되어 있는 Command 함수의 인자를 따로 정의하여 이미지를 불러와서 표시하는 프로그램을 만들어 보겠습니다.

먼저 응용에 앞서, 프로그램 작성 계획을 세워 봅시다.

이 프로그램은, PNG와 ICO 파일만을 열 수 있도록 범위를 한정해 설계할 것입니다. 따라서 파일 확장자를 검사할 필요가 있습니다.
또한, 기존 프로그램은 파일을 열기 위한 다이얼로그는 있었지만 불러온 파일을 어떻게 사용할 것인지 정의하지 않았습니다.
따라서 다이얼로그를 연 후에 이미지 위젯을 만들기 위한 구문이 추가된 새 함수를 만들어서 사용해야 합니다.
또한, 불러온 파일을 읽어낼 필요도 있습니다.

종합해 보면, 작성 계획은 다음과 같이 정리됩니다.
1. PNG와 ICO만을 걸러내기 위한 구문 작성
2. 다이얼로그에서 전달한 파일을 불러오기 위한 구문 작성
3. 읽어낸 파일의 내용을 보여주기 위한 위젯 작성


이것을 간단하게 반영한 코드는 다음과 같습니다.


```go
package main

import (
	"fmt"
	"log"
	"os" //파일을 열고 닫기 위해 사용합니다.
	"runtime"
	"strings" //파일 확장자를 추출하기 위해 사용합니다.

	. "modernc.org/tk9.0"
)

func handleFileOpen() { //이미지를 열고 표시하기 위해 함수를 정의합니다.
	s := GetOpenFile() //GetOpenFile은 string 슬라이스 형태로 열린 파일을 반환합니다.
	for _, photo := range s { //따라서 슬라이스를 차례대로 인덱싱합니다.
        //이 예제에서는 다이얼로그가 한 번에 하나의 이미지만 선택할 수 있으니 s[0]과 같은 형태로 접근해도 좋습니다.
		formatCheck := strings.Split(photo, ".") //마침표를 델리미터로 해서 스트링을 쪼개어 슬라이스 형태로 받습니다.
		format := formatCheck[len(formatCheck)-1] //오른쪽 끝에 있는 문자열이 슬라이스 중 확장자라고 가정합니다.
        //간단한 예제의 경우 위와 같이 하여도 문제가 없지만, 보통은 파일 포맷을 확인하는 라이브러리를 사용하는 경우가 많습니다.
		if (strings.Compare(format, "png") == 0) || (strings.Compare(format, "ico") == 0) { //간단한 예제이니 편의 상 확장자는 소문자라고 가정합니다.
			picFile, err := os.Open(photo) //ICO 혹은 PNG 파일을 엽니다.
			if err != nil {
				log.Println("Error while opening photo, error is: ", err)
			}

			pictureRawData := make([]byte, 10000*10000) //10000*10000 픽셀까지를 범위로 합니다.
			picFile.Read(pictureRawData) //바이트 슬라이스에 이미지 정보를 저장합니다.

			imageLabel := Label(Image(NewPhoto(Data(pictureRawData)))) //읽어온 이미지 정보를 앞에서 언급한 방식으로 레이블 위젯으로 변경합니다.
			Pack(imageLabel,
				TExit(), //이 버튼을 누를 시 프로그램을 종료합니다. 사진이 여러 장일 시 어느 것이든 괜찮습니다.
				Padx("1m"), Pady("2m"), Ipadx("1m"), Ipady("1m")) //레이블 위젯을 최상위 창에 포함시킵니다. 이제 이 이미지가 창에 보일 것입니다.

		}
            picFile.Close() //작업이 완료된 파일은 닫아 줍니다.
	}
}

func main() {
	menubar := Menu() //간단한 메뉴 바를 추가합니다.

	fileMenu := menubar.Menu() //파일 메뉴를 추가합니다.
	fileMenu.AddCommand(Lbl("Open..."), Underline(0), Accelerator("Ctrl+O"), Command(handleFileOpen)) //앞서 확인한대로, <u>O</u>pen...와 같은 형태로 표시됩니다.
    //커맨드는 위에서 작성한 이미지 불러오기 함수로 대체합니다.
	Bind(App, "<Control-o>", Command(func() { fileMenu.Invoke(0) })) //이번에는 Open의 인덱스가 0입니다.
	fileMenu.AddCommand(Lbl("Exit"), Underline(1), Accelerator("Ctrl+Q"), ExitHandler())
	Bind(App, "<Control-q>", Command(func() { fileMenu.Invoke(1) })) //마찬가지로 인덱스가 1인 현재에는 Invoke에 들어가는 수도 바꿔 줍니다.
	menubar.AddCascade(Lbl("File"), Underline(0), Mnu(fileMenu)) //파일 메뉴라는 것을 알 수 있게 적절한 레이블을 달아줍니다.

	App.WmTitle(fmt.Sprintf("%s on %s", App.WmTitle(""), runtime.GOOS))
	App.Configure(Mnu(menubar), Width("10c"), Height("10c")).Wait() //창의 초기 크기가 커졌습니다. 직접 비교해 보세요.
}
```
위의 코드는 완성된 애플리케이션에 비하면 아주 단순하지만, 계획에 대한 해결책은 다음과 같습니다.

1. 확장자 문제는 strings 패키지의 스트링 비교 함수를 이용해 걸러낸다.
2. 파일을 열기 위해 os 패키지를 불러와서 파일을 열고, 읽고, 닫았다.
3. 읽어낸 이미지는 앞서 한 예제와 같이 레이블 위젯으로 표시된다.

그렇다면, 이제 코드를 컴파일해서 프로그램을 실행해 보겠습니다.

![go-tk-이미지-불러오기-실행결과](/assets/images/go-tk-helloworld-svg-etc/go-tk-image-open.png)

이와 같이 이미지가 잘 불러와지는 것을 볼 수 있습니다.



## 정리

이번 글에서는 Go 언어의 Tcl/Tk 라이브러리를 사용하여 다음과 같은 내용을 살펴보았습니다:

1. 기본적인 GUI 애플리케이션 생성 방법
2. 다양한 이미지 포맷(SVG, PNG, ICO) 처리 방법
4. 위젯 패킹과 레이아웃 관리 기법
5. 이미지 데이터 처리를 위한 내부 구조
6. 단축키 바인딩과 위젯 커맨드


Go 언어에서 Tcl/Tk를 활용하면 간단하면서도 효과적인 GUI 애플리케이션을 만들 수 있습니다. 이러한 기초를 바탕으로 더 복잡한 GUI 애플리케이션 개발에 도전해보시기 바랍니다.
