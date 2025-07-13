---
id: 8wel7k12347894gsdjkfnru23kl78989
author: Yunjin Lee
title: Go에서 엔드포인트 응답 시간 단축-작업 큐 활용하기
description: 작업 큐를 이용하여 응답 시간을 단축해 봅시다.
language: ko
date: 2024-10-22T11:51:27.614782351Z
path: /blog/posts/go-handle-file-upload-with-concurrency-s844ce238
---

# 개요
Go를 처음 배울 때, 백엔드 서버를 구현하며 배우는 경우가 자주 있습니다. 이 때 RestAPI 등에서 파일 스트림을 받아서 서버에 업로드하는 예제를 만드는 경우를 예를 들어 봅시다.
Go언어 net/http 서버는 기본적으로 여러 요청을 동시에 처리하니 동시 다발적 업로드 자체에는 문제가 없습니다.
하지만 스트림 수신 이후 동작들을 모두 동기적으로 처리한다면 엔드포인트의 응답이 지연됩니다.
이러한 상황을 방지하기 위한 기법을 알아 봅시다.

# 원인

스트림을 수신하는 데에도 대체로 긴 시간이 소요되며, 큰 파일의 경우 단일 요청이 수 분동안 처리될 수 있습니다.
이러한 경우 조금이라도 수신 이후의 동작을 신속하게 처리하는 것이 중요합니다.
이 예제 시나리오는 스트림을 수신 후 임시파일로 저장, 컨테이너에 푸시하는 시나리오입니다.
이 때에 컨테이너에 임시파일을 푸시하는 부분을 워커 풀로 처리한다면, 응답 지연을 단축할 수 있습니다.
```
package file_upload

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const uploadTempDir = "/tmp/incus_uploads" // Host temporary directory

// UploadTask holds data for asynchronous file push.
type UploadTask struct {
	HostTempFilePath         string
	ContainerName            string
    HostFilename             string
	ContainerDestinationPath string
}

// UploadHandler processes file uploads. Saves to temp file, then queues for Incus push.
func UploadHandler(wr http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(wr, "POST method required.", http.StatusMethodNotAllowed)
		return
	}
	originalFilePath := req.Header.Get("X-File-Path")
    originalFilename := filepath.Base(req.Header.Get("X-Host-Path"))
	containerName := req.Header.Get("X-Container-Name")
	if originalFilePath == "" || containerName == "" {
		http.Error(wr, "Missing X-File-Path or X-Container-Name header.", http.StatusBadRequest)
		return
	}

	cleanContainerDestPath := filepath.Clean(originalFilePath)
	if !filepath.IsAbs(cleanContainerDestPath) {
		http.Error(wr, "File path must be absolute.", http.StatusBadRequest)
		return
	}

	// Create unique temporary file path on host
	tempFileName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(originalFilePath))
	hostTempFilePath := filepath.Join(uploadTempDir, tempFileName)

	if err := os.MkdirAll(uploadTempDir, 0755); err != nil {
		log.Printf("ERROR: Failed to create temp upload directory: %v", err)
		http.Error(wr, "Server error.", http.StatusInternalServerError)
		return
	}

	// Create and copy request body to temporary file (synchronous)
	outFile, err := os.Create(hostTempFilePath)
	if err != nil {
		log.Printf("ERROR: Failed to create temporary file: %v", err)
		http.Error(wr, "Server error.", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	bytesWritten, err := io.Copy(outFile, req.Body)
	if err != nil {
		outFile.Close()
		os.Remove(hostTempFilePath)
		log.Printf("ERROR: Failed to copy request body to temp file: %v", err)
		http.Error(wr, "File transfer failed.", http.StatusInternalServerError)
		return
	}
	log.Printf("Upload Info: Received %d bytes, saved to %s.", bytesWritten, hostTempFilePath)

	// Enqueue task for asynchronous Incus push
	task := UploadTask{
		HostTempFilePath:         hostTempFilePath,
		ContainerName:            containerName,
        HostFilename :            originalFilename,
		ContainerDestinationPath: cleanContainerDestPath,
	}
	EnqueueTask(task) //THIS PART
	log.Printf("Upload Info: Task enqueued for %s to %s.", originalFilePath, containerName)

	// Send immediate 202 Accepted response
	wr.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(wr, "File '%s' queued for processing on container '%s'.\n", originalFilename, containerName)
}
```

여기에서 THIS PART라고 적어둔 부분을 보면, 작업을 큐에 삽입한다는 것을 눈치채셨을 것입니다. 

이제 작업 큐가 어떻게 동작하는지 보도록 합시다.

```go
package file_upload

import (
	"log"
	"sync"
)

var taskQueue chan UploadTask
var once sync.Once

// InitWorkQueue initializes the in-memory task queue.
func InitWorkQueue() {
	once.Do(func() {
		taskQueue = make(chan UploadTask, 100)
		log.Println("Upload Info: Work queue initialized.")
	})
}

// EnqueueTask adds an UploadTask to the queue.
func EnqueueTask(task UploadTask) {
	if taskQueue == nil {
		log.Fatal("ERROR: Task queue not initialized.")
	}
	taskQueue <- task
	log.Printf("Upload Info: Queue: Task enqueued. Size: %d", len(taskQueue))
}

// DequeueTask retrieves an UploadTask from the queue, blocking if empty.
func DequeueTask() UploadTask {
	if taskQueue == nil {
		log.Fatal("ERROR: Task queue not initialized.")
	}
	task := <-taskQueue
	log.Printf("Upload Info: Queue: Task dequeued. Size: %d", len(taskQueue))
	return task
}

// GetQueueLength returns current queue size.
func GetQueueLength() int {
	if taskQueue == nil {
		return 0
	}
	return len(taskQueue)
}
```

예시로 제공된 작업 큐는 단순하게 구현되어 있습니다. 이 작업 큐는 큐에 들어가 있는 작업들을 채널에서 빼내는 단순한 구조를 가지고 있습니다. 

아래는 파일 업로드 이후 컨테이너에 푸시하기 위한 워커 메서드입니다.
현재 메서드는 좋은 반응성과 쉬운 구현을 위해 무한 루프이나 용도에 따라 알고리즘을 추가하여도 무방합니다.
```go
func StartFilePushWorker() {
	for {
		task := DequeueTask()
		log.Printf("Upload Info: Worker processing task for %s from %s.", task.ContainerName, task.HostFilename)

		// Defer cleanup of the temporary file
		defer func(filePath string) {
			if err := os.Remove(filePath); err != nil {
				log.Printf("ERROR: Worker: Failed to remove temp file '%s': %v", filePath, err)
			} else {
				log.Printf("Upload Info: Worker: Cleaned up temp file: %s", filePath)
			}
		}(task.HostTempFilePath)

		// Process task with retries for transient Incus errors
		for i := 0; i <= MaxRetries; i++ {
			err := processUploadTask(task) //separate upload task
			if err == nil {
				log.Printf("SUCCESS: Worker: Task completed for %s.", task.ContainerName)
				break
			}

			isTransient := true
			if err.Error() == "incus: container not found" { // Example permanent error
				isTransient = false
			}

			if isTransient && i < MaxRetries {
				log.Printf("WARNING: Worker: Task failed for %s (attempt %d/%d): %v. Retrying.",
					task.ContainerName, i+1, MaxRetries, err)
				time.Sleep(RetryDelay)
			} else {
				log.Printf("ERROR: Worker: Task permanently failed for %s after %d attempts: %v.",
					task.ContainerName, i+1, err)
				break
			}
		}
	}
}
```
먼저 이 함수는 계속해서 루프를 돌면서 태스크를 큐로부터 받아오고자 시도합니다. 이후, 재시도 범위 내에서 스트림-임시파일이 아닌, 임시파일-컨테이너로의 업로드 단계를 시도합니다.

# 이점

이러한 처리 방식의 이점은, **스트림 업로드만 정상이라면**이후 처리되는 작업에 대한 지연 시간을 줄일 수 있으며, 동시다발적인 컨테이너 작업으로 인한 자원 고갈을 예방할 수 있습니다.  현재의 코드에서 보이듯, 동시에 진행 가능한 컨테이너 작업은 채널의 수로 제한됩니다. 이와 같이 실용적으로 Go의 병렬 처리를 사용 가능한 예제를 알아보았습니다. 보다 많은 예시를 보고 싶다면, 아래의 링크를 방문해 주세요.
[예제를 포함한 모듈](https://github.com/gg582/linux_virt_unit)
[예제를 활용한 프로젝트](https://github.com/gg582/incuspeed)
프로젝트 자체에는 부수적인 구성요소가 많으니, 워커 자체에 대한 학습은 main.go에서 어떻게 워커 init 함수를 호출하는지만 간략히 보시고 넘어가면 됩니다. 모듈에는 다른 형태의 워커도 포함하고 있으니 참고해 주세요.

