package main

import (
	//"github.com/lixin9311/icli-go"
	"flag"
	"fmt"
	"github.com/naoina/toml"
	"io/ioutil"
	"os"
)

var (
	file = flag.String("c", "config.toml", "Config file.")
	enc  = flag.Bool("e", false, "no encrypt")
	udp  = flag.Bool("u", false, "udp")
)

func readConf(path string) (*Config, error) {
	fd, err := os.Open(*file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	buf, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}
	config := new(Config)
	if err := toml.Unmarshal(buf, config); err != nil {
		return nil, err
	}
	return config, nil
}
func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	flag.Parse()
	conf, err := readConf(*file)
	if err != nil {
		fmt.Println("Failed to read config:", err)
		panic(err)
	}
	if conf.User.Role == "client" {
		err = runClient(conf)
		if err != nil {
			fmt.Println("Client instance exited with error:", err)
			panic(err)
		} else {
			fmt.Println("Client instance exited without error.")
		}
	} else if conf.User.Role == "server" {
		err = runServer(conf)
		if err != nil {
			fmt.Println("Server instance exited with error:", err)
			panic(err)
		} else {
			fmt.Println("Server instance exited without error.")
		}
	} else {
		fmt.Println("Unexpected instance role:", conf.User.Role)
	}
	return
}
