package main

import (
	"fmt"
	"net/http"
	"bufio"
)

func main() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost/1", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", "Bearer zort")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	fmt.Println(resp.Status)

	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < 5; i++ {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
