package go_rcon

import (
	"testing"
	"time"
)

func Test(t *testing.T) {
	client, err := Open("", "", true)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 10)

	resp, err := client.Write("RefreshList")
	if err != nil {
		panic(err)
	}

	if !resp.Successful {
		t.Error("test failed")
	}
}
