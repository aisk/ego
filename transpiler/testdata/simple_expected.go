package main

import "fmt"

func process() error {
	result, err := someFunc()
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

func someFunc() (string, error) {
	return "hello", nil
}
