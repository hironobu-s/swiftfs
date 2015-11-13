package main

import "github.com/hironobu-s/objfs/objfs"

func main() {
	app := objfs.NewApp()
	app.RunAndExitOnError()
}
