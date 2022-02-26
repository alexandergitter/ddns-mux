package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type Config struct {
	BindAddress string
	DDNSHost    string
	DDNSUser    string
	DDNSPass    string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func contains(s []interface{}, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func main() {
	var config Config
	var entries map[string]interface{}

	if len(os.Args) < 2 {
		panic("Missing config file argument")
	}

	_, err := toml.DecodeFile(os.Args[1], &config)
	check(err)

	_, err = toml.DecodeFile(os.Args[1], &entries)
	check(err)

	http.HandleFunc("/nic/update", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		myip := params.Get("myip")
		hostname := params.Get("hostname")
		username, password, _ := r.BasicAuth()

		fmt.Println("Received update request:", r.URL)

		if myip == "" || hostname == "" {
			fmt.Println("Error: Empty hostname or myip")
			io.WriteString(w, "nohost\n")
			return
		}

		if username == "" || password == "" {
			fmt.Println("Error: Empty username or password")
			io.WriteString(w, "badauth\n")
			return
		}

		if entries[username] == nil {
			fmt.Println("Error: Invalid username given:", username)
			io.WriteString(w, "badauth\n")
			return
		}

		entry := entries[username].(map[string]interface{})

		if entry["Password"] != password {
			fmt.Println("Error: Invalid password given for user", username)
			io.WriteString(w, "badauth\n")
			return
		}

		if !contains(entry["Domains"].([]interface{}), hostname) {
			fmt.Printf("Error: Domain %s not allowed for user %s\n", hostname, username)
			io.WriteString(w, "badauth\n")
			return
		}

		url := fmt.Sprintf("https://%s:%s@%s/nic/update?hostname=%s&myip=%s", config.DDNSUser, config.DDNSPass, config.DDNSHost, "domain", "127.0.0.1")
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error while requesting update:", err)
			io.WriteString(w, "dnserr\n")
			return
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error while reading response:", err)
			io.WriteString(w, "dnserr\n")
			return
		}

		fmt.Printf("Upstream response %v:\n%v\n", resp.Status, string(body))

		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	})

	fmt.Println("Listening on", config.BindAddress)
	http.ListenAndServe(config.BindAddress, nil)
}
