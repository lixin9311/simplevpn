package main

import (
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/lixin9311/icli-go"
	"net"
	// modifed for beautiful private key encoding. unsafe for other user.
	"github.com/naoina/toml"
	"os"
	"time"
	//"io"
	"./simplevpn"
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
)

var (
	db        *bolt.DB
	users     = map[string]*simplevpn.User{}
	rsaLength = 2048
	bucket    = []byte("users")
	ip        = "10.0.0.1"
	port      = 1080
	comment   = "this is a comment."
)

func genKeyDer() (privateKeyDer, publicKeyDer []byte, err error) {
	genprivateKey, err := rsa.GenerateKey(rand.Reader, rsaLength)
	if err != nil {
		return
	}
	privateKeyDer = x509.MarshalPKCS1PrivateKey(genprivateKey)

	genpublicKey := genprivateKey.PublicKey
	publicKeyDer, err = x509.MarshalPKIXPublicKey(&genpublicKey)
	return
}

func parseDer(privateKeyDer []byte) string {
	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	}
	return string(pem.EncodeToMemory(&privateKeyBlock))
}

func opendb(args ...string) (err error) {
	params := flag.NewFlagSet(args[0], flag.ContinueOnError)
	params.SetOutput(os.Stdout)
	file := params.String("o", "user.db", "Database file to open.")
	mode := params.Int("m", 664, "Open mode.")
	timeout := params.Duration("t", time.Second, "Open timeout.")
	if err = params.Parse(args[1:]); err != nil {
		fmt.Println("Failed to parse flag: ", err)
		return
	}
	if db, err = bolt.Open(*file, os.FileMode(*mode), &bolt.Options{Timeout: *timeout}); err != nil {
		fmt.Println("Failed to open database: ", err)
		return
	}
	fmt.Println("Database opened.")
	return
}

func closedb(args ...string) (err error) {
	if db == nil {
		fmt.Println("Database already closed.")
		return fmt.Errorf("some err")
	}
	if err = db.Close(); err != nil {
		fmt.Println("Failed to close database: ", err)
		return
	}
	db = nil
	fmt.Println("Database closed.")
	return
}

func promtAndRead(promt, def string, reader *bufio.Reader) (result string, err error) {
	fmt.Printf("%s(default:%s):", promt, def)
	if result, err = reader.ReadString('\n'); err != nil {
		result = ""
		return
	} else if result == "\n" {
		result = def
		return
	}
	result = result[:len(result)-1]
	return
}

func newUser(args ...string) (err error) {
	user := new(simplevpn.User)
	input := bufio.NewReader(os.Stdin)
	if user.Name, err = promtAndRead("User Name", "", input); err != nil {
		return
	}
	if user.Email, err = promtAndRead("User Email", "lixin9311@gmail.com", input); err != nil {
		return
	}
	if user.Password, err = promtAndRead("User Password", "Password", input); err != nil {
		return
	}
	if usertype, err := promtAndRead("User Type", "Admin", input); err != nil {
		return err
	} else {
		user.Type = simplevpn.User_UserType(simplevpn.User_UserType_value[usertype])
	}
	if pri, pub, err := genKeyDer(); err != nil {
		fmt.Println("Failed to generate key: ", err)
		return err
	} else {
		user.Extension = pri
		user.PubBase64 = base64.StdEncoding.EncodeToString(pub)
		user.PriBase64 = base64.StdEncoding.EncodeToString(pri)
	}
	users[user.Name] = user
	fmt.Println("New user created.")
	return
}

func listUser(args ...string) (err error) {
	for _, v := range users {
		fmt.Println(v)
	}
	return nil
}

func updateToDB(args ...string) (err error) {
	if db == nil {
		err = fmt.Errorf("Database not opened yes.\n")
		fmt.Println("Failed to write database: ", err)
		return err
	}
	tx, err := db.Begin(true)
	if err != nil {
		fmt.Println("Failed to start transaction: ", err)
		return err
	}
	defer tx.Rollback()
	b, err := tx.CreateBucketIfNotExists(bucket)
	if err != nil {
		fmt.Println("Failed to create db bucket: ", err)
		return err
	}
	for name, val := range users {
		data, err := val.Marshal()
		if err != nil {
			fmt.Println("Failed to encode user: ", err)
			return err
		}
		if err := b.Put([]byte(name), data); err != nil {
			fmt.Printf("Failed to put User(Name:%s) into database: %v\n", name, err)
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		fmt.Println("Failed to commit database: ", err)
		return err
	}
	fmt.Println("Updated databse successfully.")
	return nil
}

func readFromDB(args ...string) (err error) {
	if db == nil {
		err = fmt.Errorf("Database not opened yes.\n")
		fmt.Println("Failed to read database: ", err)
		return err
	}
	tx, err := db.Begin(false)
	if err != nil {
		fmt.Println("Failed to start transaction: ", err)
		return err
	}
	defer tx.Rollback()
	b := tx.Bucket(bucket)
	if b == nil {
		err = fmt.Errorf("Bucket[%s] do not exist.", string(bucket))
		fmt.Println("Failed to open bucket: ", err)
		return err
	}
	err = b.ForEach(func(name, v []byte) error {
		user := new(simplevpn.User)
		err := user.Unmarshal(v)
		if err != nil {
			fmt.Println("Failed to unmarshal data: ", err)
			return err
		}
		users[string(name)] = user
		return nil
	})
	if err != nil {
		fmt.Println("Database iteration failed: ", err)
		return
	}
	fmt.Println("Read database successfully.")
	return
}

func genConfig(args ...string) (err error) {
	params := flag.NewFlagSet(args[0], flag.ContinueOnError)
	params.SetOutput(os.Stdout)
	file := params.String("o", "client.conf", "Config file to save.")
	mode := params.Int("m", 664, "Open mode.")
	username := params.String("n", "", "Selected user.")
	if err = params.Parse(args[1:]); err != nil {
		fmt.Println("Failed to parse flag: ", err)
		return
	}
	user, ok := users[*username]
	if !ok {
		err = fmt.Errorf("User[%s] not found.", username)
		fmt.Println("Failed to find user.")
		return err
	}
	config := new(simplevpn.Config)
	config.User.Name = user.Name
	config.User.Password = user.Password
	config.User.Email = user.Email
	config.User.PrivateKey = parseDer(user.Extension)
	config.Server.Ip = ip
	config.Server.Port = port
	config.Server.Comment = comment
	data, err := toml.Marshal(*config)
	if err != nil {
		fmt.Println("Failed to generate config: ", err)
		return err
	}
	fd, err := os.OpenFile(*file, os.O_RDWR|os.O_CREATE|os.O_EXCL, os.FileMode(*mode))
	if err != nil {
		fmt.Println("Failed to open config file: ", err)
		return err
	}
	defer fd.Close()
	n, err := fd.Write(data)
	if err != nil {
		fmt.Println("Filed to write file: ", err)
		return err
	} else if n != len(data) {
		return fmt.Errorf("Wrote length mismatched: %d/%d bytes wrote", n, len(data))
	}
	fmt.Println("Config file successfully generated.")
	return nil
}

func exit(args ...string) error {
	return icli.ExitIcli
}

func help(args ...string) error {
	return icli.PrintDesc
}

func errorHandler(e error) error {
	return nil
}

func test(args ...string) error {
	fmt.Println("asdfasdfadsfdsafdhhhhhhhhhhhhhhhasdfdsafadsfsadasdjfhkjdshfkdshflkjdsahfhdsalfhsafjhkfhldsfhdslhfdsalhfdsahflkdsahflkdsahflkhfjlkdsafhkdsa")
	return nil
}

func listenPort(args ...string) error {
	conn, err := net.Listen("tcp", "8080")
	if err != nil {
		fmt.Println("Failed to listen tcp port:8080", err)
		return err
	}
}

func main() {
	defer closedb()
	cmds := []icli.CommandOption{
		{"opendb", "Open a database", opendb},
		{"closedb", "Close a database", closedb},
		{"newuser", "Generate a new user.", newUser},
		{"listuser", "List all user in cache.", listUser},
		{"update", "Update the database.", updateToDB},
		{"read", "Read the database.", readFromDB},
		{"genconf", "Generate config file.", genConfig},
		{"exit", "Exit.", exit},
		{"help", "Help.", help},
		{"test", "Help.", test},
	}
	icli.AddCmd(cmds)
	icli.SetPromt("SimpleVPN >")
	icli.Start(errorHandler)
}
