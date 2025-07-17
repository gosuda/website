---
id: e55cbf5b5116abf3a5fc62c1af99e19b
author: Yunjin Lee
title: Go에서 Tk로 파일 목록이 추가된 이미지 뷰어 만들기
description: Go 언어로 Tcl/Tk를 이용해 간단한 이미지 뷰어를 만들어 보세요.
language: ko
date: 2024-11-19T11:51:05.279238471Z
path: /blog/posts/creating-a-image-viewer-with-tk-in-go-z279238471Z
---

지난 게시물에서는 CGo-Free Tcl/Tk 라이브러리에 대해 간단하게 살펴 봤습니다.
이번 시간에는 지난 번의 예제를 응용하여 이미지 뷰어를 만들어 보도록 하겠습니다.

## 이미지 뷰어 계획

1. 지난 시간의 이미지 표시기는 이미지 삭제 기능이 없어 여러 이미지들을 불러올 수록
창의 크기가 모자랐습니다.
사용하지 않는 라벨을 삭제해 줍시다.
2. 이미지 표시기에 여러 이미지를 목록화할 것이라면 리스트를 만들어 줍시다.
3. 한 번에 여러 이미지를 불러오는 것이 좋습니다.
4. 보지 않을 이미지를 목록에서 빼는 것도 구현해 줍니다.
5. 다중 이미지 중 특정 이미지를 선택해서 보는 기능을 만들어 줍시다.


## 수정된 Tcl/Tk 9.0 라이브러리


기존 라이브러리에는 Listbox 구현이 미흡해서 이미지 목록을 보여주기 힘듭니다.
수정된 라이브러리를 다운로드해 줍시다.
Git CLI가 설치되어 있지 않다면 tarball이나 zip 파일로 다운로드받아도 좋습니다.

```bash
git clone https://github.com/gg582/tk9.0
```

먼저 추가된 기능들의 몇 가지를 살펴 봅시다.

일단 새로운 함수를 살펴보기에 앞서서, 기존 함수들은 어떻게 되어 있는지
tk.go의 1017행의 Destroy 함수를 통해 구조를 간단하게 파악해보도록 하겠습니다.

```go
func Destroy(options ...Opt) {
	evalErr(fmt.Sprintf("destroy %s", collect(options...)))
}
```
이 함수는 evalErr라는 함수에 Tcl 스크립트 형식으로 동작을 전달해 구현되어 있습니다.
그 말은, 원하는 기능을 구현하기 위해서는 해당하는 Tcl 스크립트의 형식으로 명령을 전달하기만 된다는 뜻입니다.

예시로 리스트박스에 항목을 추가하는 메서드를 구현해 봅시다.
먼저 Tcl 스크립팅을 위해 공식 매뉴얼에서 listbox에 사용 가능한 명령어 중 insert를 살펴 봅시다.

[insert 명령어](https://www.tcl.tk/man/tcl8.6/TkCmd/listbox.htm#M29)

insert 명령어의 설명 페이지를 보면, insert는 특정 인덱스에 나열된 항목들을 삽입합니다.
그렇다면 이를 구현하기 위해서 
```go
func (l *ListboxWidget) AddItem(index int, items string) {
	evalErr(fmt.Sprintf("%s insert %d %s", l.fpath, index, items))
}
```
와 같은 코드를 작성할 수 있습니다.


이제 대략적인 구현 원리를 알았으니, Listbox를 위한 추가 기능들부터 설명하겠습니다.

### 리스트: 삽입/삭제

```go
package main
import . "modernc.org/tk9.0"

func main() {
    length := 3
    l := Listbox()
    l.AddItem(0, "item1 item2 item3")
    b1 := TButton(Txt("Delete Multiple Items, index (0-1)"), Command( func(){
        if length >= 2 {
            l.DeleteItems(0,1)
            length-=2
        }
    }))
    b2 := TButton(Txt("Delete One Item, index (0)"), Command( func () {
        if length > 0 {
            l.DeleteOne(0) 
            length-=1
        }
    }))
    Pack(TExit(),l,b1,b2)
    App.Wait()
}

```


위 프로그램에서 AddItem은 스페이스바로 구분된 서로 다른 아이템들을 인덱스 0부터 차례대로 넣습니다.
item1-item3는 차례대로 0, 1, 2 인덱스를 갖게 됩니다.
항목 삭제가 어떻게 동작하는지 예제를 실행시켜 알아봅니다.


### 리스트: 선택된 항목 가져오기


이제 Listbox에서 선택된 항목들을 가지고 온 후 확인해 보겠습니다.

```go
package main

import . "modernc.org/tk9.0"

func main() {
    l := Listbox()
    l.SelectMode("multiple")
    l.AddItem(0, "item1 item2 item3")
    btn := TButton(Txt("Print Selected"), Command( func() {
        sel := l.Selected()
        for _, i := range sel {
            println(l.GetOne(i))
        }
    }))

    Pack(TExit(), l, btn)
    App.Wait()
}
```

Selected 메서드는 현재 Listbox에서 선택된 모든 항목들의 인덱스를 가져옵니다.
GetOne 메서드는 해당 인덱스에 해당하는 항목의 값을 가져옵니다.
콘솔에 출력되는 결과로 알 수 있습니다.
참고로 유사 메서드인 Get 메서드는 시작과 끝 인덱스를 받아 범위 내 항목의 값을 모두 가져옵니다.

다음은 리스트박스의 색상을 바꿔 보도록 하겠습니다.

먼저 아래의 예제를 살펴봅시다.

```go
package main

import . "modernc.org/tk9.0"

func main() {
    l := Listbox()
    l.Background("blue")
    l.Foreground("yellow")
    l.SelectBackground("black")
    l.SelectForeground("white")
    l.Height(20)
    l.Width(6)
    l.AddItem(0, "item1 item2 item3")
    l.ItemForeground(0,"red")
    l.ItemBackground(0,"green")
    l.ItemSelectBackground(0,"white")
    l.ItemSelectForeground(0,"black")
    l.Relief("ridged")
    Pack(TExit(),l)
    App.Wait()
}
```

![색상 적용 결과](/assets/images/go-tk-imageviewer/listbox_color.png)

위의 코드에서 작성한 대로, 높이가 늘어났습니다.
또한, 색상이 잘 적용된 것을 볼 수 있습니다.
여기서 특정 항목에만 색깔을 다르게 하는 옵션이 지정되어 있어
첫째 줄만 색상이 다르게 적용된 것을 알 수 있습니다.

또한, 큰 차이는 없지만 Relief 메서드를 이용하면 flat, groove, raise, ridge, solid, sunken 중에
위젯 테두리의 스타일을 변경할 수 있습니다.


## 이미지 뷰어 예제

그럼 앞서 배운 위젯을 이용해서 이미지 뷰어를 만들어 보도록 하겠습니다.
예제 프로그램은 다음과 같습니다.

```go
package main

import (
    "fmt"
    "log"
    "os"
    "runtime"
    "strings"
    "path"

    . "modernc.org/tk9.0"
)

var pbuttons []*TButtonWidget
var extensions []FileType
var pbutton *TButtonWidget = nil
var listbox, listbox2 *ListboxWidget
var cur *LabelWidget = nil
var imagesLoaded []*LabelWidget
func PhotoName(fileName string) string {
        fileName = path.Base(fileName)
        return fileName[:len(fileName)-len(path.Ext(fileName))]
}

func handleFileOpen() {
    res := GetOpenFile(Multiple(true),Filetypes(extensions)) //다중 선택을 활성화하고 필터를 켭니다.
    s := make([]string,0,1000)
    for _, itm := range res {
        if itm != "" {
            tmp := strings.Split(itm," ")
            s = append(s,tmp...)
        }
    }

    for _, photo := range s {
        formatCheck := strings.Split(photo, ".")
        format := formatCheck[len(formatCheck)-1]

        if (strings.Compare(format, "png") == 0) || (strings.Compare(format, "ico") == 0) {
            picFile, err := os.Open(photo)
            if err != nil {
                log.Println("Error while opening photo, error is: ", err)
            }

            pictureRawData := make([]byte, 10000*10000)
            picFile.Read(pictureRawData)

            imageLabel := Label(Image(NewPhoto(Data(pictureRawData))))
                        imagesLoaded = append(imagesLoaded,imageLabel)
            var deleteTestButton *TButtonWidget
            deleteTestButton = TButton(
                Txt("Unshow Image"),
            Command(func() {
                GridForget(imageLabel.Window)
                GridForget(deleteTestButton.Window)
            }))

            pbuttons = append(pbuttons,deleteTestButton)

                        listbox.AddItem(len(imagesLoaded)-1,PhotoName(photo))
                        listbox2.AddItem(len(imagesLoaded)-1,PhotoName(photo))
            picFile.Close()
        }
    }
}

func DeleteSelected () {
    s:=listbox.Selected()
    if len(s) == 0 {
        return
        }
    for _, i := range s {
        listbox.DeleteOne(i)
        listbox2.DeleteOne(i)
        if len(imagesLoaded)-1>i {
            continue
        }
        if cur == imagesLoaded[i] {
            pbutton = nil
            cur = nil
        }
        Destroy(imagesLoaded[i])
        Destroy(pbuttons[i])
                imagesLoaded = append(imagesLoaded[:i],imagesLoaded[i+1:]...)
        pbuttons = append(pbuttons[:i], pbuttons[i+1:]...)
    }
}

func SelectImage() {
        s:=listbox2.Selected()
        if len(s) == 0 {
                return
        }

    if len(imagesLoaded) -1 < s[0] {
        return
    }
    if imagesLoaded[s[0]] == nil {
        return
    }
    if cur != nil {
            GridForget(cur.Window)
    }
    if pbutton != nil {
        GridForget(pbutton.Window)
    }

        Grid(imagesLoaded[s[0]], Row(0), Column(2))
    Grid(pbuttons[s[0]], Row(0), Column(3))
    cur = imagesLoaded[s[0]]
    pbutton = pbuttons[s[0]]
}

func SelectIndex(index int) {

    if len(imagesLoaded) -1 <index {
        return
    }
    if imagesLoaded[index] == nil {
        return
    }
    if cur != nil {
            GridForget(cur.Window)
    }
    if pbutton != nil {
        GridForget(pbutton.Window)
    }

        Grid(imagesLoaded[index], Row(0), Column(2))
    Grid(pbuttons[index], Row(0), Column(3))
    cur = imagesLoaded[index]
    pbutton = pbuttons[index]
}

func main() {
    menubar := Menu()
    //DefaultTheme("awdark","themes/awthemes-10.4.0")
    //테마를 사용하고 싶을 때에는 테마 명과 경로를 지정해 줍니다.
    fileMenu := menubar.Menu()
    extensions = make([]FileType,0,1)
    extensions = append(extensions, FileType{ "Supported Images", []string {".png",".ico"}, "" } )
    //필터에 png와 ico를 넣어 줍니다.
    fileMenu.AddCommand(Lbl("Open..."), Underline(0), Accelerator("Ctrl+O"), Command(func () {
        handleFileOpen()
        SelectIndex(len(imagesLoaded)-1)
    } ))
    Bind(App, "<Control-o>", Command(func() { fileMenu.Invoke(0) }))
    fileMenu.AddCommand(Lbl("Exit"), Underline(1), Accelerator("Ctrl+Q"), ExitHandler())
    Bind(App, "<Control-q>", Command(func() { fileMenu.Invoke(1) }))
    menubar.AddCascade(Lbl("File"), Underline(0), Mnu(fileMenu))
        imagesLoaded = make([]*LabelWidget, 0, 10000)
    pbuttons = make([]*TButtonWidget,0,10000)
    var scrollx, scroll, scroll2, scrollx2 *TScrollbarWidget
        listbox = Listbox(Yscrollcommand(func(e *Event) { e.ScrollSet(scroll)}) , Xscrollcommand( func(e *Event) { e.ScrollSet(scrollx)}))
        listbox2 = Listbox(Yscrollcommand(func(e *Event) { e.ScrollSet(scroll2)}), Xscrollcommand(func(e *Event) { e.ScrollSet(scrollx2)}))
        listbox.SelectMode("multiple")
        listbox2 = Listbox()
        listbox.Background("white")
        listbox.SelectBackground("blue")
        listbox.SelectForeground("yellow")
        listbox2.Background("grey")
        listbox2.SelectBackground("green")
    listbox2.SelectForeground("blue")
    listbox2.SelectBackground("brown")
        listbox.Height(5)
        listbox.Width(4)
        listbox2.Height(5)
        listbox2.Width(4)
        delBtn := Button(Txt("Delete"), Command(func () { DeleteSelected() }))
        selBtn := Button(Txt("Select"), Command(func () { SelectImage() }))
        scroll = TScrollbar(Command(func(e *Event) { e.Yview(listbox) }))
        scrollx = TScrollbar(Orient("horizontal"),Command(func(e *Event) { e.Xview(listbox) }))
    scroll2 = TScrollbar(Command(func(e *Event) { e.Yview(listbox2) }))
        scrollx2 = TScrollbar(Orient("horizontal"),Command(func(e *Event) { e.Xview(listbox2) }))
        Grid(listbox,Row(1),Column(0), Sticky("nes"))
        Grid(scroll,Row(1),Column(1), Sticky("nes"))
    Grid(scrollx,Row(2),Column(0),  Sticky("nes"))
        Grid(delBtn,Row(3),Column(0), Sticky("nes"))
        Grid(listbox2,Row(1),Column(2), Sticky("nes"))
        Grid(scroll2,Row(1),Column(3), Sticky("nes"))
    Grid(scrollx2,Row(2),Column(2), Sticky("nes"))
        Grid(selBtn,Row(3),Column(2), Sticky("nes"))
    App.WmTitle(fmt.Sprintf("%s on %s", App.WmTitle(""), runtime.GOOS))
    App.Configure(Mnu(menubar), Width("80c"), Height("60c")).Wait()
}

```

![이미지 뷰어 실행 결과](/assets/images/go-tk-imageviewer/imageviewer.png)

이 예제에서는 구현을 간단하게 하기 위해 모든 이미지 위젯을 불러올 때 미리 만들어 두며,
중복 파일을 확인하지 않습니다.
앞서 말씀드린 문제점을 개선할 수도 있고,
주석 처리된 부분인 DefaultTheme 메서드를 이용하여 테마를 변경해 볼 수도 있습니다.
이러한 부분을 개선한 프로그램을 새로 만들어 보면서 연습해 보시기 바랍니다.


## 정리

이번 글에서는 Go 언어의 Tcl/Tk 라이브러리의 명령 호출이 어떤 식으로 동작하는지 알아보고,
리스트박스가 추가된 이미지 뷰어를 만들어 보았습니다.

1. Tcl/Tk 라이브러리의 명령 호출 방식
2. 리스트박스 사용 방법
3. 리스트박스 위젯의 속성 변경
4. 이미지 뷰어 작성

이와 같은 방식으로 다른 라이브러리들의 수정에도 도전해 보고, 추가한 기능으로 완성된 프로그램을 작성해 보시기 바랍니다.
