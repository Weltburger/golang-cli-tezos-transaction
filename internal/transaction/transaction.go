package transaction

import (
	"blockwatch.cc/tzgo/tezos"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/go-resty/resty/v2"
	"github.com/goat-systems/go-tezos/v4/forge"
	"github.com/goat-systems/go-tezos/v4/keys"
	"github.com/goat-systems/go-tezos/v4/rpc"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"tezos/pkg/models"
)

func CreateTransaction(address, amount, path string) error {
	_, err := tezos.ParseAddress(address)
	if err != nil {
		return errors.Wrap(err, "invalid address")
	}

	file, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "invalid path")
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.Wrap(err, "cannot read the file")
	}

	secretKey := fmt.Sprintf("%s", data)

	key, err := keys.FromBase58(secretKey, keys.Ed25519)
	if err != nil {
		fmt.Printf("failed to import keys: %s\n", err.Error())
		return err
	}

	accountInfo, err := getAccountInfo(key.PubKey.GetAddress())
	if err != nil {
		fmt.Printf("failed to get account info: %s\n", err.Error())
		return err
	}

	if !accountInfo.CheckBalance(amount) {
		fmt.Printf("you do not have enough money for this transaction!\n")
		return err
	}

	client, err := rpc.New("https://testnet-tezos.giganode.io")
	if err != nil {
		fmt.Printf("failed to initialize rpc client: %s\n", err.Error())
		return err
	}

	accountInfo.Counter++

	resp, head, err := client.Block(&rpc.BlockIDHead{})
	if err != nil {
		fmt.Printf("failed to get (%s) head block: %s\n", resp.Status(), err.Error())
		return err
	}

	transaction := rpc.Transaction{
		Kind:         rpc.TRANSACTION,
		Source:       key.PubKey.GetAddress(),
		Fee:          "5000",
		GasLimit:     "26283",
		StorageLimit: "365",
		Counter:      strconv.FormatInt(accountInfo.Counter, 10),
		Amount:       amount,
		Destination:  address,
	}
	var op string
	if accountInfo.Revealed {
		op, err = forge.Encode(head.Hash, transaction.ToContent())
		if err != nil {
			fmt.Printf("failed to forge transaction: %s\n", err.Error())
			return err
		}
	} else {
		reveal := rpc.Reveal{
			Kind:         "reveal",
			Source:       key.PubKey.GetAddress(),
			Fee:          "1269",
			Counter:      strconv.FormatInt(accountInfo.Counter, 10),
			GasLimit:     "10000",
			StorageLimit: "0",
			PublicKey:    key.PubKey.GetPublicKey(),
			Metadata:     nil,
		}
		transaction.Counter = strconv.FormatInt(accountInfo.Counter+1, 10)
		op, err = forge.Encode(head.Hash, reveal.ToContent(), transaction.ToContent())
		if err != nil {
			fmt.Printf("failed to forge transaction: %s\n", err.Error())
			return err
		}
	}

	signature, err := key.SignHex(op)
	if err != nil {
		fmt.Printf("failed to sign operation: %s\n", err.Error())
		return err
	}

	rsty := resty.New()

	resp, ophash, err := injectionOperation(rsty, rpc.InjectionOperationInput{
		Operation: signature.AppendToHex(op),
		ChainID:   head.Hash,
	})
	if err != nil {
		fmt.Printf("failed to inject (%s): %s\n", resp.Status(), err.Error())
		return err
	}

	fmt.Println("Done processing. Transaction has been created. Operation hash:", ophash)
	return nil
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

	return resp, nil
}

func getAccountInfo(address string) (*models.AccountInfo, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.granadanet.tzkt.io/v1/accounts/%s", address))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ai := new(models.AccountInfo)
	body, _ := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(body, ai); err != nil {
		return nil, err
	}

	return ai, nil
}
