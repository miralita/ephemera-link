package main

import "log"

func main() {
	cfg := ReadConfig()
	//log.Printf("%+v", cfg)
	storage := NewStorage(cfg)
	defer storage.Clear()
	app := NewApp(cfg, storage)
	app.Run()
}
