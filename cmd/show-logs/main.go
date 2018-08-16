package main

import (
	"log"
	"path/filepath"
	"fmt"
	"io/ioutil"
)

func main() {
	path := filepath.Join("build", "events.log")

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("could not open %s: %s", path, err.Error())
	}

	fmt.Println("*********************************** [ begin log ] ***********************************")
	fmt.Print(string(contents))
	fmt.Println("************************************ [ end log ] ************************************")
}
