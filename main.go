package main

import (
	"fmt"
	"ispace/common/id"
	"ispace/serve"
)

func main() {

	for range 100 {
		i := id.Next().Int64()
		fmt.Println(i, id.PathOfId(i))
	}
	serve.Serve()
}
