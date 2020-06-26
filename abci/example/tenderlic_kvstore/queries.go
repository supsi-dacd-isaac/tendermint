package tenderlic_kvstore

import (
	"fmt"
	"strings"

	"github.com/tendermint/tendermint/abci/types"
)

func (app *Application) checkQueryAllowance(storedData string, meterId string) (res bool) {
	storedDataArr := strings.Split(storedData, ",")

	if app.Contains(storedDataArr, meterId) {
		return true
	} else {
		logger.Warning(fmt.Sprintf("Access unauthorized: Meter \"%s\" not allowed to perform queries", meterId))
		return false
	}
}

func (app *Application) GetBalance(reqQuery types.RequestQuery, meter string) (resQuery types.ResponseQuery) {
	value, _ := app.state.db.Get(prefixKey(reqQuery.Data))

	if value == nil {
		logger.Warning(fmt.Sprintf("Query \"%s\" has no value", string(prefixKey(reqQuery.Data))))
		resQuery.Log = "Value not stored"
		resQuery.Value = nil
	} else {
		if app.CheckAdmin() {
			resQuery.Log = "Stored value"
			resQuery.Value = value
		} else {
			// get the MAC hash
			if app.checkMacAddress(meter) {
				resQuery.Log = "Stored value"
				resQuery.Value = value
			} else {
				logger.Warning(fmt.Sprintf("ACCESS DENIED! You are not allowed to query the balance of meter %s", meter))
				resQuery.Log = "Not authorized"
				resQuery.Value = nil
			}
		}
	}
	return resQuery
}

func (app *Application) GetTimeRelatedValues(reqQuery types.RequestQuery, meter string) (resQuery types.ResponseQuery) {
	allowedMeters := app.GetAllowedMeters()
	value, _ := app.state.db.Get(prefixKey(reqQuery.Data))

	if app.checkQueryAllowance(string(allowedMeters), meter) == false {
		resQuery.Value = nil
		resQuery.Log = "Meter not authorized"
	} else {
		if value == nil {
			resQuery.Log = "Not available value"
		} else {
			resQuery.Log = "Stored value"
		}
		resQuery.Value = value
	}
	return resQuery
}

func (app *Application) GetSingleValue(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	value, _ := app.state.db.Get(prefixKey(reqQuery.Data))

	if value == nil {
		logger.Warning(fmt.Sprintf("Query \"%s\" has no value", string(prefixKey(reqQuery.Data))))
		resQuery.Log = "does not exist"
	} else {
		logger.Info(fmt.Sprintf("%s = %s", reqQuery.String(), value))
		resQuery.Log = "exists"
	}
	resQuery.Value = value
	return resQuery
}

func (app *Application) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	// Set the logger
	app.SetLogger()

	rawStr := strings.ReplaceAll(string(prefixKey(reqQuery.Data)), "kvPairKey:", "")
	keyInfo := strings.Split(rawStr, "_")

	logger.Info(fmt.Sprintf("Performing query: %s", reqQuery.String()))

	if keyInfo[0] == "admin" || keyInfo[0] == "allowed" {
		resQuery = app.GetSingleValue(reqQuery)
	} else if keyInfo[0] == "lict-balance" {
		resQuery = app.GetBalance(reqQuery, keyInfo[1])
	} else if len(keyInfo) == 3 {
		resQuery = app.GetTimeRelatedValues(reqQuery, keyInfo[1])
	} else {
		logger.Warning(fmt.Sprintf("Query %s is not well formed", reqQuery.String()))
		resQuery.Log = "Query not well formed"
		resQuery.Value = nil
	}
	return resQuery
}
