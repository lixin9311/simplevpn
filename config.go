package main

type Config struct {
	User   user
	Server server
}

type user struct {
	Password string
	Role     string
	Method   string
}

type server struct {
	Ip   string
	Port int
}
