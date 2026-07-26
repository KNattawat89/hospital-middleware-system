package main

import (
	"fmt"
	"sync"
)

var wg sync.WaitGroup

func main() {

	tasks := []struct {
		name     string
		dir      string
		filename string
	}{
		{"patient", "core/patient", "repo.go"},
		{"staff", "core/staff", "repo.go"},
	}

	for _, task := range tasks {
		wg.Go(func() {
			if err := generateRepo(task.name, task.dir, task.filename); err != nil {
				fmt.Printf("Failed to generate %s: %v\n", task.name, err)
			}
		})
	}

	wg.Wait()
}
