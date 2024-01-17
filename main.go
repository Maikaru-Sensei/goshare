package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/fatih/color"
	"goshare/ipfs"
	"os"
	"strings"
)

// IpfsConnector represents the connection to IPFS
type IpfsConnector struct {
	Connector *ipfs.Connector
}

// Command defines the interface for the command pattern
type Command interface {
	Execute(ctx context.Context) error
}

// Invoker represents the client that triggers the command
type Invoker struct {
	command Command
}

func (i *Invoker) SetCommand(command Command) {
	i.command = command
}

func (i *Invoker) ExecuteCommand(ctx context.Context) error {
	if i.command == nil {
		return fmt.Errorf("command not set")
	}
	return i.command.Execute(ctx)
}

// IpfsConfigCommand is a concrete command for configuring IPFS
type IpfsConfigCommand struct {
	ipfsConnector *IpfsConnector
	repository    string
}

func (c *IpfsConfigCommand) Execute(ctx context.Context) error {
	connector, err := ipfs.CreateNode(ctx, c.repository)
	if err != nil {
		return err
	}
	c.ipfsConnector.Connector = connector
	return nil
}

// IpfsAddCommand is a concrete command for adding a file to IPFS
type IpfsAddCommand struct {
	ipfsConnector *IpfsConnector
	filePath      string
}

func (c *IpfsAddCommand) Execute(ctx context.Context) error {
	if c.ipfsConnector.Connector == nil {
		return fmt.Errorf("run config command first")
	}
	return c.ipfsConnector.Connector.AddFile(ctx, c.filePath)
}

// IpfsGetCommand is a concrete command for getting a file from IPFS
type IpfsGetCommand struct {
	ipfsConnector *IpfsConnector
	cid           string
	outputPath    string
}

func (c *IpfsGetCommand) Execute(ctx context.Context) error {
	if c.ipfsConnector.Connector == nil {
		return fmt.Errorf("run config command first")
	}
	return c.ipfsConnector.Connector.GetFile(ctx, c.cid, c.outputPath)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ipfsConnector := &IpfsConnector{}
	invoker := &Invoker{}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("goshare> ")
		scanner.Scan()
		command := scanner.Text()

		switch command {
		case "help":
			fmt.Println("Available commands:")
			fmt.Println(" - config <repository>")
			fmt.Println(" - add <file-path>")
			fmt.Println(" - get <cid> <output-path>")
			fmt.Println(" - exit")
		case "exit":
			fmt.Println("Exiting the app. Goodbye!")
			os.Exit(0)
		default:
			err := handleIpfsCommand(command, ctx, ipfsConnector, invoker)
			if err != nil {
				color.Red(err.Error())
			}
		}
	}
}

func handleIpfsCommand(command string, ctx context.Context, ipfsConnector *IpfsConnector, invoker *Invoker) error {
	ipfsCommand := strings.Fields(command)
	switch ipfsCommand[0] {
	case "config":
		configCommand := &IpfsConfigCommand{
			ipfsConnector: ipfsConnector,
			repository:    ipfsCommand[1],
		}
		invoker.SetCommand(configCommand)
	case "add":
		if ipfsConnector.Connector == nil {
			return fmt.Errorf("run config command first")
		}
		addCommand := &IpfsAddCommand{
			ipfsConnector: ipfsConnector,
			filePath:      ipfsCommand[1],
		}
		invoker.SetCommand(addCommand)
	case "get":
		if ipfsConnector.Connector == nil {
			return fmt.Errorf("run config command first")
		}
		getCommand := &IpfsGetCommand{
			ipfsConnector: ipfsConnector,
			cid:           ipfsCommand[1],
			outputPath:    ipfsCommand[2],
		}
		invoker.SetCommand(getCommand)
	default:
		fmt.Printf("invalid command: %s\n", command)
		return nil
	}

	return invoker.ExecuteCommand(ctx)
}
