// Backdoor is a program that allows to inject Google Cloud Service Account credentials into a running container.
//

package main

import (
	"encoding/base64"
	"os"

	"github.com/lemon-mint/vbox"
)

func main() {
	if os.Getenv("BUILD_BACKDOOR") != "" {
		content := os.Getenv("BUILD_BACKDOOR")
		key_env := os.Getenv("BUILD_BACKDOOR_KEY")

		if key_env == "" {
			panic("BUILD_BACKDOOR_KEY is not set")
		}

		key, err := base64.RawURLEncoding.DecodeString(key_env)
		if err != nil {
			panic(err)
		}

		b := vbox.NewBlackBox(key)

		data, ok := b.Base64Open(content)
		if !ok {
			panic("failed to decrypt")
		}

		err = os.MkdirAll("~/.gcloud", 0755)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile("~/.gcloud/service-account.json", data, 0644)
		if err != nil {
			panic(err)
		}
	}
}
