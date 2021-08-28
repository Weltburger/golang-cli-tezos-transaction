package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/go-resty/resty/v2"
	"github.com/goat-systems/go-tezos/v4/forge"
	"github.com/goat-systems/go-tezos/v4/keys"
	"github.com/goat-systems/go-tezos/v4/rpc"
	"github.com/pkg/errors"
	"math/big"
	"os"
	"strconv"
)

func main() {
	key, err := keys.FromBase58("edsk3T1UVpvwMnJUjh6FrDPPbF4MqJsQkzTQAC36t4VG3TU1W9C8Pu", keys.Ed25519)
	if err != nil {
		fmt.Printf("failed to import keys: %s\n", err.Error())
		os.Exit(1)
	}

	client, err := rpc.New("https://testnet-tezos.giganode.io")
	if err != nil {
		fmt.Printf("failed to initialize rpc client: %s\n", err.Error())
		os.Exit(1)
	}

	resp, counter, err := client.ContractCounter(rpc.ContractCounterInput{
		BlockID:    &rpc.BlockIDHead{},
		ContractID: key.PubKey.GetAddress(),
	})
	if err != nil {
		fmt.Printf("failed to get (%s) counter: %s\n", resp.Status(), err.Error())
		os.Exit(1)
	}
	counter++

	big.NewInt(0).SetString("10000000000000000000000000000", 10)

	/*reveal := rpc.Reveal{
		Kind:         "reveal",
		Source:       key.PubKey.GetAddress(),
		Fee:          "1269",
		Counter:      strconv.Itoa(counter),
		GasLimit:     "10000",
		StorageLimit: "0",
		PublicKey:    key.PubKey.GetPublicKey(),
		Metadata:     nil,
	}*/
	transaction := rpc.Transaction{
		Kind:        rpc.TRANSACTION,
		Source:      key.PubKey.GetAddress(),
		Fee:         "2941",
		GasLimit:    "26283",
		StorageLimit: "365",
		Counter:     strconv.Itoa(counter),
		Amount:      "1000000",
		Destination: "tz1aRnX4C1FLn4byjM3xqv7DerrUdUFLPszb",
	}

	resp, head, err := client.Block(&rpc.BlockIDHead{})
	if err != nil {
		fmt.Printf("failed to get (%s) head block: %s\n", resp.Status(), err.Error())
		os.Exit(1)
	}

	op, err := forge.Encode(head.Hash, /*reveal.ToContent(),*/ transaction.ToContent())
	if err != nil {
		fmt.Printf("failed to forge transaction: %s\n", err.Error())
		os.Exit(1)
	}

	signature, err := key.SignHex(op)
	if err != nil {
		fmt.Printf("failed to sign operation: %s\n", err.Error())
		os.Exit(1)
	}
///////////////////////////////////////////////////////////////////////////////////////
/*	resp, ophash, err := client.InjectionOperation(rpc.InjectionOperationInput{
		Operation: signature.AppendToHex(op),
		//Async: false,
	})
	if err != nil {
		fmt.Printf("failed to inject (%s): %s\n", resp.Status(), err.Error())
		os.Exit(1)
	}*/
///////////////////////////////////////////////////////////////////////////////////////

	rsty := resty.New()

	resp, ophash, err := injectionOperation(rsty, rpc.InjectionOperationInput{
		Operation: signature.AppendToHex(op),
		ChainID: head.Hash,
	})
	if err != nil {
		fmt.Printf("failed to inject (%s): %s\n", resp.Status(), err.Error())
		os.Exit(1)
	}


	fmt.Println(ophash)
}

func injectionOperation(r *resty.Client, input rpc.InjectionOperationInput) (*resty.Response, string, error) {
	err := validator.New().Struct(input)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to inject operation: invalid input")
	}

	v, err := json.Marshal(input.Operation)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to inject operation")
	}
	resp, err := post(r, "/injection/operation", v, input.ChainID)
	if err != nil {
		return resp, "", errors.Wrap(err, "failed to inject operation")
	}

	var opstring string
	err = json.Unmarshal(resp.Body(), &opstring)
	if err != nil {
		return resp, "", errors.Wrap(err, "failed to inject operation: failed to parse json")
	}

	return resp, opstring, nil
}

func post(r *resty.Client, path string, body []byte, chainID string) (*resty.Response, error) {
	resp, err := r.R().
		SetQueryParams(map[string]string{
			"ChainID": chainID,
	}).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(fmt.Sprintf("%s%s", "https://testnet-tezos.giganode.io", path))

	if err != nil {
		return resp, err
	}

	return resp, err
}
