package tenderlic_kvstore

import (
	"bytes"
	"fmt"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	"strconv"
	"strings"
)

func (app *Application) SetKVOnDB(key []byte, value []byte) types.ResponseDeliverTx {
	app.state.db.Set(prefixKey(key), value)
	app.state.Size++
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}

func (app *Application) DeliverTxMint(keyInfo []string, amountRaw []byte) types.ResponseDeliverTx {
	var keyBalance = []byte(fmt.Sprintf("lict-balance_%s", keyInfo[1]))
	storedBalance := app.ReadBalanceInDB(keyBalance)
	amount, _ := strconv.Atoi(string(amountRaw))
	return app.SetKVOnDB(keyBalance, []byte(fmt.Sprintf("%d", storedBalance+amount)))
}

func (app *Application) DeliverTxSelfBurn(amountRaw []byte) types.ResponseDeliverTx {
	var mac, _ = app.GetMacAddr()
	var keyBalance = []byte(fmt.Sprintf("lict-balance_%s", app.CalcSHA512Hash(mac[0])))
	storedBalance := app.ReadBalanceInDB(keyBalance)
	amount, _ := strconv.Atoi(string(amountRaw))
	return app.SetKVOnDB(keyBalance, []byte(fmt.Sprintf("%d", storedBalance-amount)))
}
func (app *Application) DeliverTxPowerMeasure(dtStr string, valueStr []byte) types.ResponseDeliverTx {
	var mac, _ = app.GetMacAddr()
	return app.SetKVOnDB([]byte(fmt.Sprintf("meter_%s_%s", app.CalcSHA512Hash(mac[0]), dtStr)), valueStr)
}

func (app *Application) DeliverTxTransfer(keyInfo []string, amount []byte) types.ResponseDeliverTx {
	var mac, _ = app.GetMacAddr()
	var keySenderBalance = []byte(fmt.Sprintf("lict-balance_%s", app.CalcSHA512Hash(mac[0])))
	var keyReceiverBalance = []byte(fmt.Sprintf("lict-balance_%s", keyInfo[1]))
	amountToTransfer, _ := strconv.Atoi(string(amount))

	var senderBalance = app.ReadBalanceInDB(keySenderBalance)
	var receiverBalance = app.ReadBalanceInDB(keyReceiverBalance)

	resSender := app.SetKVOnDB(keySenderBalance, []byte(fmt.Sprintf("%d", senderBalance-amountToTransfer)))
	resReceiver := app.SetKVOnDB(keyReceiverBalance, []byte(fmt.Sprintf("%d", receiverBalance+amountToTransfer)))

	if resSender.Code == code.CodeTypeOK && resReceiver.Code == code.CodeTypeOK {
		return types.ResponseDeliverTx{Code: code.CodeTypeOK}
	} else {
		return types.ResponseDeliverTx{Code: code.CodeTypeBadRequest}
	}
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}

func (app *Application) ReadBalanceInDB(key []byte) int {
	var balance = 0
	if flag, _ := app.state.db.Has(prefixKey(key)); flag == true {
		var balanceRaw, _ = app.state.db.Get(prefixKey(key))
		balance, _ = strconv.Atoi(string(balanceRaw))
	}
	return balance
}

func (app *Application) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	var key, value []byte
	var keyInfo []string

	parts := bytes.Split(req.Tx, []byte("="))
	if len(parts) == 2 {
		key, value = parts[0], parts[1]
		keyInfo = strings.Split(string(key), "_")
		cmd := keyInfo[0]

		if cmd == "lict-mint" {
			return app.DeliverTxMint(keyInfo, value)
		} else if cmd == "lict-selfburn" {
			return app.DeliverTxSelfBurn(value)
		} else if cmd == "lict-transfer" {
			return app.DeliverTxTransfer(keyInfo, value)
		} else if cmd == "meter" {
			return app.DeliverTxPowerMeasure(keyInfo[1], value)
		} else {
			return app.SetKVOnDB(key, value)
		}
	} else {
		return types.ResponseDeliverTx{Code: code.CodeTypeBadRequest}
	}
}
