package main

func someFunc() (int, error) {
	return 1, nil
}

func callSomeFunc() (bool, error) {
	if result, err := someFunc(); err != nil {
		return false, err
	} else if result > 0 {
		return true, nil
	} else if result, err := someFunc(); err != nil {
		return false, err
	} else if result {
		return true, nil
	}
	if result, err := someFunc(); err != nil {
		return false, err
	} else if result {
		return true, nil
	}

	return false, nil
}
