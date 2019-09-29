package main

import (
	"common"
	ec "errorcheck"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/cenkalti/rpc2"
	shell "github.com/ipfs/go-ipfs-api"
)

var sh *shell.Shell

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:5000")
	ec.CheckError(err, "Connect: ")

	clt := rpc2.NewClient(conn)

	clt.Handle("sendTask", sendTask)
	clt.Handle("getTaskAndRun", getTaskAndRun)

	sh = shell.NewShell("localhost:5001")

	go func() {
		clt.Run()
		<-clt.DisconnectNotify()
		fmt.Println("Disconnect")
	}()
	defer clt.Close()

	var rep string
	clt.Call("clientRegister", common.Register{ID: "1"}, &rep)
	fmt.Println("Register result", rep)

	time.Sleep(time.Second * 1)

	for {
		fmt.Println("Enter 'q' to quit")
		var input string
		fmt.Scanln(&input)

		if input == "q" {
			break
		}
	}

	clt.Call("unregister", common.Register{ID: "1"}, &rep)
	fmt.Println("Unregister result: ", rep)
}

func sendTask(client *rpc2.Client, args *common.Args, reply *string) error {
	*reply = "ok"
	return nil
}

//getTask 中的参数应为哈希码
func getTaskAndRun(client *rpc2.Client, args *common.Args, reply *string) error {
	err := sh.Get(args.Hash, "$HOME/.task")
	ec.CheckError(err, "Get task file: ")

	command := `./VirtualOS`
	cmd := exec.Command("/bin/bash", "-c", command)
	output, err := cmd.Output()
	ec.CheckError(err, "Excute VirtualOS: ")
	fmt.Println("VituralOS output:", string(output))

	//连接docker并运行镜像
	conn, err := net.Dial("tcp", "127.0.0.1:4399")
	ec.CheckError(err, "Connect: ")
	defer conn.Close()

	load := "PUT /path/$HOME/.task/" + args.Hash
	conn.Write([]byte(load))

	//TODO docker id 暂定为0
	askStatus := "GET /status/0"
	conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	for {
		buf := make([]byte, 64)
		conn.Write([]byte(askStatus))
		_, err = conn.Read(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				*reply = "docker time out"
				break
			}
		}
		if string(buf[:]) == "ok" {
			*reply = "ok"
			break
		} else if string(buf[:]) == "error" {
			*reply = "error"
			break
		}

		time.Sleep(time.Second * 120)
	}

	return nil
}
