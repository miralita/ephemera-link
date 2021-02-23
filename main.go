package main

func main() {
	cfg := ReadConfig()
	storage := NewStorage(cfg)
	defer storage.Clear()
	app := NewApp(cfg, storage)
	app.Run()
}
