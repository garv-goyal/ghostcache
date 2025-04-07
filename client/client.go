package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	coordinatorURL := "http://localhost:9000"
	key := "foo"
	value := "bar"

	// Set key
	err := setKey(coordinatorURL, key, value)
	if err != nil {
		fmt.Println("Set error:", err)
		os.Exit(1)
	}

	// Get key
	val, err := getKey(coordinatorURL, key)
	if err != nil {
		fmt.Println("Get error:", err)
		os.Exit(1)
	}

	fmt.Printf("Got value for key '%s': %s\n", key, val)
}

func setKey(coordinatorURL, key, value string) error {
	nodeURL, err := lookupNode(coordinatorURL, key)
	if err != nil {
		return err
	}

	payload := map[string]string{
		"key":   key,
		"value": value,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(nodeURL+"/set", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func getKey(coordinatorURL, key string) (string, error) {
	nodeURL, err := lookupNode(coordinatorURL, key)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(fmt.Sprintf("%s/get?key=%s", nodeURL, key))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func lookupNode(coordinatorURL, key string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/where?key=%s", coordinatorURL, key))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(body)), nil
}
