package remote_serve

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/zbrumen/remote-serve/net"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestListenAndServe(t *testing.T) {
	_, err := net.NewServer(":4000", map[string]string{
		"username": "password",
	})
	if err != nil {
		t.Fatal(err)
	} else {
		go func() {
			if err := ListenAndServe("remote://username:password@localhost:4000/"+base64.URLEncoding.EncodeToString([]byte(":5000")), http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				fmt.Println("Recieved request for", r.URL.Path)
				fmt.Println("Method:", r.Method)
				rw.Write([]byte("Secret"))
			})); err != nil {
				t.Fatal(err)
			}
		}()
		// wait for everything to setup
		time.Sleep(time.Second)
		req, err := http.NewRequest(http.MethodGet, "http://localhost:5000", nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		} else {
			defer resp.Body.Close()
			out, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			} else {
				if string(out) != "Secret" {
					t.Fatal("Secret does not match")
				} else {
					fmt.Println(string(out))
				}
			}
		}
	}
}
