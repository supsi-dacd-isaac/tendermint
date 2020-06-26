package tenderlic_kvstore

import (
	"bytes"
	"fmt"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	"strconv"
	"strings"
)

func (app *Application) checkMeterAllowance(allowedMetersRawData []byte, meterId string) (res bool) {
	metersList := strings.Split(string(allowedMetersRawData), ",")

	if app.Contains(metersList, meterId) {
		return true
	} else {
		logger.Warning(fmt.Sprintf("Access unauthorized: Meter \"%s\" not in the allowed list of meters", meterId))
		return false
	}
}

func (app *Application) CheckAdmin() bool {
	if flag, _ := app.state.db.Has(prefixKey([]byte("admin"))); flag == true {
		hashAdminOnChain, _ := app.state.db.Get(prefixKey([]byte("admin")))
		return app.checkMacAddress(string(hashAdminOnChain))
	} else {
		return true
	}
}

func (app *Application) checkMacAddress(macToCheck string) bool {
	macs, _ := app.GetMacAddr()
	if macToCheck == app.CalcSHA512Hash(macs[0]) {
		return true
	} else {
		return false
	}
}

func (app *Application) CheckTxAdmin() types.ResponseCheckTx {
	logger.Info(fmt.Sprintf("Admin setting transaction"))
	if !app.CheckAdmin() {
		logger.Warning("ACCESS DENIED! You are not the community admin!")
		return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	} else {
		logger.Info(fmt.Sprintf("Set new admin"))
		return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
	}
}

func (app *Application) CheckTxAllowed() types.ResponseCheckTx {
	logger.Info(fmt.Sprintf("Allowance meters setting transaction"))

	if !app.CheckAdmin() {
		logger.Warning("ACCESS DENIED! You are not the community admin, you can not change the allowed meters list")
		return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	} else {
		logger.Info(fmt.Sprintf("Set new meters list"))
		return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
	}
}

func (app *Application) CheckTxPowerMeasure(keyInfo []string) types.ResponseCheckTx {
	return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

func (app *Application) CheckTxMint(keyInfo []string, valueInfo string) types.ResponseCheckTx {
	logger.Info(fmt.Sprintf("Tokens mint transaction"))
	if !app.CheckAdmin() {
		logger.Warning("ACCESS DENIED! You are not the community admin, you can not mint tokens")
		return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	} else {
		// Check the meter allowance
		allowedMeters := app.GetAllowedMeters()
		if app.checkMeterAllowance(allowedMeters, keyInfo[1]) == false {
			return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
		} else {
			amount, _ := strconv.Atoi(valueInfo)
			if amount <= 0 {
				logger.Warning(fmt.Sprintf("The amount must be positive"))
				return types.ResponseCheckTx{Code: code.CodeNotPositiveAmount, GasWanted: 1}
			} else {
				logger.Info(fmt.Sprintf("Successfull minting"))
				return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
			}
		}
	}
}

func (app *Application) CheckTxSelfBurn(valueInfo string) types.ResponseCheckTx {
	logger.Info(fmt.Sprintf("Tokens burn transaction"))

	amount, _ := strconv.Atoi(valueInfo)
	if amount <= 0 {
		logger.Warning(fmt.Sprintf("The amount must be positive"))
		return types.ResponseCheckTx{Code: code.CodeNotPositiveAmount, GasWanted: 1}
	} else {
		logger.Info(fmt.Sprintf("Successfull burning"))
		return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
	}
}
func (app *Application) CheckTxTransfer(keyInfo []string, valueInfo string) types.ResponseCheckTx {
	logger.Info(fmt.Sprintf("Tokens transfer transaction"))
	receiver := keyInfo[1]
	allowedMeters := app.GetAllowedMeters()
	// Check if the receiver is an allowed meter
	if app.checkMeterAllowance(allowedMeters, receiver) == false {
		return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
	} else {
		amount, _ := strconv.Atoi(valueInfo)
		if amount <= 0 {
			logger.Warning(fmt.Sprintf("The amount must be positive"))
			return types.ResponseCheckTx{Code: code.CodeNotPositiveAmount, GasWanted: 1}
		} else {
			mac, _ := app.GetMacAddr()
			keySenderBalance := []byte(fmt.Sprintf("lict-balance_%s", app.CalcSHA512Hash(mac[0])))
			senderBalance := app.ReadBalanceInDB(keySenderBalance)
			if amount > senderBalance {
				logger.Warning(fmt.Sprintf("The amount to transfer exceeds the sender balance"))
				return types.ResponseCheckTx{Code: code.CodeExceedingAmount, GasWanted: 1}
			} else {
				logger.Info(fmt.Sprintf("Successfull transfer"))
				return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
			}
		}
	}
}

func (app *Application) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	// Set the logger
	app.SetLogger()

	// Initialize the response
	resp := types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}

	data := bytes.Split(req.Tx, []byte("="))
	if len(data) == 2 {
		var keyInfo = strings.Split(string(data[0]), "_")
		var valueInfo = string(data[1])

		logger.Info(fmt.Sprintf("Checking transaction %s", req.String()))
		logger.Info(fmt.Sprintf("Checking transaction "))

		if keyInfo[0] == "admin" {
			resp = app.CheckTxAdmin()
		} else if keyInfo[0] == "allowed" {
			resp = app.CheckTxAllowed()
		} else if keyInfo[0] == "lict-mint" {
			resp = app.CheckTxMint(keyInfo, valueInfo)
		} else if keyInfo[0] == "lict-selfburn" {
			resp = app.CheckTxSelfBurn(valueInfo)
		} else if keyInfo[0] == "lict-transfer" {
			resp = app.CheckTxTransfer(keyInfo, valueInfo)
		} else if keyInfo[0] == "meter" && len(keyInfo) == 2 {
			resp = app.CheckTxPowerMeasure(keyInfo)
		} else {
			logger.Error(fmt.Sprintf("Bad request"))
			resp = types.ResponseCheckTx{Code: code.CodeTypeBadRequest, GasWanted: 1}
		}

		if resp.Code == code.CodeTypeOK {
			logger.Info(fmt.Sprintf("Transaction OK"))
		}
		return resp
	} else {
		logger.Error(fmt.Sprintf("Bad request"))
		return types.ResponseCheckTx{Code: code.CodeTypeBadRequest, GasWanted: 1}
	}
}
