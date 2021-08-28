package models

import (
	"fmt"
	"os"
	"strconv"
)

type AccountInfo struct {
	Balance  uint64 `json:"balance"`
	Revealed bool   `json:"revealed"`
	Counter  int64  `json:"counter"`
}

func (ai *AccountInfo) CheckBalance(money string) bool {
	amount, err := strconv.Atoi(money)
	if err != nil {
		fmt.Printf("could not convert amount (string) into int: %s\n", err.Error())
		os.Exit(1)
	}

	if ai.Balance < (uint64(amount) + uint64(5000)) {
		return false
	}

	return true
}
