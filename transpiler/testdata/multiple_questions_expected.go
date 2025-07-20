package main

import "fmt"

func process() error {
	first, err := getFirst()
	if err != nil {
		return err
	}
	second, err := getSecond()
	if err != nil {
		return err
	}
	fmt.Println(first, second)
	return nil
}

func getFirst() (string, error) {
	return "first", nil
}

func getSecond() (string, error) {
	return "second", nil
}
