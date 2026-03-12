package resgister

import (
	"fmt"
	//import anonymously to register tasks to the list
	_ "github.com/iceymoss/go-task/internal/tasks/ai"
	_ "github.com/iceymoss/go-task/internal/tasks/network"
)

func init() {
	fmt.Println("tasks registered")
}
