package main

import (
	"fmt"
	"github.com/devfacet/gocmd/v3"
	"os"
	"tezos/internal/transaction"
)

var (
	version = "v1.0"
)

func main() {
	flags := struct {
		SendToAddress struct {
			Address string `short:"a" long:"address" required:"true" description:"Address of recipient"`
			Amount  string `short:"s" long:"amount" required:"true" description:"The amount of money to be sent"`
			Path    string `short:"p" long:"path" required:"true" description:"Path to the file with secret key"`
		} `command:"sendtoaddress" description:"Make transaction in Tezos blockchain and send money (amount) to recipient (address)"`
	}{}

	_, err := gocmd.HandleFlag("SendToAddress", func(cmd *gocmd.Cmd, args []string) error {
		if err := transaction.CreateTransaction(flags.SendToAddress.Address,
			flags.SendToAddress.Amount,
			flags.SendToAddress.Path); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Printf("failed to create transaction: %s\n", err.Error())
		os.Exit(1)
	}

	_, err = gocmd.New(gocmd.Options{
		Name:        "golang-cli-tezos-transaction",
		Description: "A simple cli Golang application that creates a transaction in the Tezos blockchain.",
		Version:     fmt.Sprintf("%s", version),
		Flags:       &flags,
		ConfigType:  gocmd.ConfigTypeAuto,
	})
	if err != nil {
		fmt.Printf("failed to create transaction: %s\n", err.Error())
		os.Exit(1)
	}
}
