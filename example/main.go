package main

import (
	"fmt"
	"github.com/kofemann/linux_ftrace"
	"os"
	"os/signal"
)

func main() {

	eventTrace := ftrace.NewEventTrace("sunrpc/xprt_transmit")
	eventTrace.Enable()

	c := eventTrace.EventSource()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
loop:
	for {
		select {
		case l, ok := <-c:
			fmt.Println(l.String())
			if !ok {
				break loop
			}
		case <-signalChan:
			fmt.Println("\nInterrupted...")
			eventTrace.Disable()
		}

	}

}
