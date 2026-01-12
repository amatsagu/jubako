package main

import "github.com/amatsagu/lumo"

func main() {
	lumo.EnableDebug()
	lumo.EnableStackOnWarns()

	lumo.Info("Hello world!")
	lumo.Close()
}
