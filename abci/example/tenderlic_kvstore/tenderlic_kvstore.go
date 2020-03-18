package tenderlic_kvstore

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/op/go-logging"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/kv"
	"github.com/tendermint/tendermint/version"
	dbm "github.com/tendermint/tm-db"
)

// Logger variables
var logger = logging.MustGetLogger("TenderLICKVstore")
var format = logging.MustStringFormatter(`%{level:.1s}[%{time:2006-01-02|15:04:05.000}] %{message}`)
var backend1 = logging.NewLogBackend(os.Stderr, "", 0)
var backend2 = logging.NewLogBackend(os.Stderr, "", 0)
var backend2Formatter = logging.NewBackendFormatter(backend2, format)
var backend1Leveled = logging.AddModuleLevel(backend1)

var (
	stateKey        = []byte("stateKey")
	kvPairPrefixKey = []byte("kvPairKey:")

	ProtocolVersion version.Protocol = 0x1
)

type State struct {
	db      dbm.DB
	Size    int64  `json:"size"`
	Height  int64  `json:"height"`
	AppHash []byte `json:"app_hash"`
}

func loadState(db dbm.DB) State {
	var state State
	state.db = db
	stateBytes, err := db.Get(stateKey)
	if err != nil {
		panic(err)
	}
	if len(stateBytes) == 0 {
		return state
	}
	err = json.Unmarshal(stateBytes, &state)
	if err != nil {
		panic(err)
	}
	return state
}

func saveState(state State) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	state.db.Set(stateKey, stateBytes)
}

func prefixKey(key []byte) []byte {
	return append(kvPairPrefixKey, key...)
}

//---------------------------------------------------

var _ types.Application = (*Application)(nil)

type Application struct {
	types.BaseApplication

	state State
}

func NewApplication() *Application {
	state := loadState(dbm.NewMemDB())
	return &Application{state: state}
}

func (app *Application) Info(req types.RequestInfo) (resInfo types.ResponseInfo) {
	return types.ResponseInfo{
		Data:             fmt.Sprintf("{\"size\":%v}", app.state.Size),
		Version:          version.ABCIVersion,
		AppVersion:       ProtocolVersion.Uint64(),
		LastBlockHeight:  app.state.Height,
		LastBlockAppHash: app.state.AppHash,
	}
}

// tx is either "key=value" or just arbitrary bytes
func (app *Application) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	var key, value []byte
	parts := bytes.Split(req.Tx, []byte("="))
	if len(parts) == 2 {
		key, value = parts[0], parts[1]
	} else {
		key, value = req.Tx, req.Tx
	}

	app.state.db.Set(prefixKey(key), value)
	app.state.Size++

	events := []types.Event{
		{
			Type: "app",
			Attributes: []kv.Pair{
				{Key: []byte("creator"), Value: []byte("Cosmoshi Netowoko")},
				{Key: []byte("key"), Value: key},
			},
		},
	}

	return types.ResponseDeliverTx{Code: code.CodeTypeOK, Events: events}
}

func (app *Application) getMacAddr() []string {
	ifas, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var as []string
	for _, ifa := range ifas {
		a := ifa.HardwareAddr.String()
		if a != "" {
			as = append(as, a)
			break
		}
	}
	return as
}

func (app *Application) checkTime(dtStr string) bool {
	var tmp = strings.Split(dtStr, "T")
	var tmpT = strings.Split(tmp[1], "Z")
	tmpT = strings.Split(tmpT[0], ":")

	var m, errM = strconv.Atoi(tmpT[1])
	var s, errS = strconv.Atoi(tmpT[2])
	if errM == nil && errS == nil && m%15 == 0 && s == 0 {
		return true
	} else {
		return false
	}
}

func (app *Application) SetLogger() {
	backend1Leveled.SetLevel(logging.CRITICAL, "")
	logging.SetBackend(backend1Leveled, backend2Formatter)
}

func (app *Application) checkSigner(data []string) (res bool) {
	nodeNameRaw, _ := ioutil.ReadFile(fmt.Sprintf("%s/name", os.Getenv("HOME")))
	nodeName := strings.TrimSuffix(string(nodeNameRaw), "\n")

	if data[0] != "allowed" {
		if nodeName != data[1] {
			logger.Warning(fmt.Sprintf("Meter \"%s\" is trying to cheat signing a transaction as / doing a query about meter \"%s\"", nodeName, data[1]))
			return false
		}
	}
	return true
}

func (app *Application) checkTransactionAllowance(storedData []byte, txData []string) (res bool) {
	metersList := strings.Split(string(storedData), ",")
	code := true

	if txData[0] != "allowed" {
		if app.contains(metersList, txData[1]) {
			code = true
		} else {
			logger.Warning(fmt.Sprintf("Access unauthorized: Meter \"%s\" not allowed to perform transactions", txData[1]))
			code = false
		}
	} else {
		code = true
	}
	return code
}

func (app *Application) checkQueryAllowance(storedData string, qData string) (res bool) {
	storedDataArr := strings.Split(storedData, ",")
	qDataArr := strings.Split(qData, "_")

	if app.contains(storedDataArr, qDataArr[1]) {
		return true
	} else {
		logger.Warning(fmt.Sprintf("Access unauthorized: Meter \"%s\" not allowed to perform queries", qDataArr[1]))
		return false
	}
}

func (app *Application) contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func (app *Application) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	// Set the logger
	app.SetLogger()

	data := strings.Split(req.String(), "=")
	var flagAllowed = data[0]
	data = strings.Split(data[0], "_")

	//logger.Info(fmt.Sprintf("Checking transaction %s", req.String()))
	logger.Info(fmt.Sprintf("Checking transaction "))

	if flagAllowed == "tx:\"allowed" {
		logger.Info(fmt.Sprintf("Allowance transaction"))
	} else if len(data) == 3 { // Check the data format

		// Check the signature
		/*if app.checkSigner(data) == false {
			return types.ResponseCheckTx{Code: code.CodeTypeCheatingMeter, GasWanted: 1}
		}*/

		// Check the time format
		/*ts := data[2]
		if app.checkTime(ts) == false {
			return types.ResponseCheckTx{Code: code.CodeTypeBadTimeFormat, GasWanted: 1}
		}*/

		// Check the meter allowance
		keyFlag := []byte("allowed")
		allowedMeters, err := app.state.db.Get(prefixKey(keyFlag))
		if err != nil {
			panic(err)
		}

		if app.checkTransactionAllowance(allowedMeters, data) == false {
			return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized, GasWanted: 1}
		}
	} else {
		logger.Error(fmt.Sprintf("Bad request"))
		return types.ResponseCheckTx{Code: code.CodeTypeBadRequest, GasWanted: 1}
	}

	logger.Info(fmt.Sprintf("Transaction OK"))
	return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

func (app *Application) Commit() types.ResponseCommit {
	// Using a memdb - just return the big endian size of the db
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Size)
	app.state.AppHash = appHash
	app.state.Height++
	saveState(app.state)
	return types.ResponseCommit{Data: appHash}
}

// Returns an associated value or nil if missing.
func (app *Application) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	// Set the logger
	app.SetLogger()

	data := strings.Split(string(prefixKey(reqQuery.Data)), ":")
	key := data[1]

	logger.Info(fmt.Sprintf("Performing query: %s", reqQuery.String()))

	// Check if there is an allowance query
	if key == "allowed" {
		logger.Info(fmt.Sprintf("Allowance query: %s", resQuery.String()))
		value, _ := app.state.db.Get(prefixKey(reqQuery.Data))

		if value == nil {
			logger.Warning(fmt.Sprintf("Query \"%s\" has no value", string(prefixKey(reqQuery.Data))))
			resQuery.Log = "does not exist"
		} else {
			logger.Info(fmt.Sprintf("%s = %s", reqQuery.String(), value))
			resQuery.Log = "exists"
		}
		resQuery.Value = value
	} else {
		// Check the signature
		data = strings.Split(data[1], "_")

		if len(data) == 3 {
			// Check the signature
			if app.checkSigner(data) == false {
				logger.Warning(fmt.Sprintf("Access denied"))
				resQuery.Log = "Access denied"
				resQuery.Value = nil
			} else {
				keyFlag := []byte("allowed")
				allowedMeters, _ := app.state.db.Get(prefixKey(keyFlag))
				value, _ := app.state.db.Get(prefixKey(reqQuery.Data))

				if app.checkQueryAllowance(string(allowedMeters), key) == false {
					resQuery.Value = nil
					resQuery.Log = "Not authorized"
				} else {
					if value == nil {
						//logger.Warning(fmt.Sprintf("%s = N/A", reqQuery.String()))
						resQuery.Log = "Not available value"
					} else {
						//logger.Info(fmt.Sprintf("%s = %s", reqQuery.String(), value))
						resQuery.Log = "Stored value"
					}
					resQuery.Value = value
				}
			}
		} else {
			logger.Warning(fmt.Sprintf("Query %s is not well formed", reqQuery.String()))
			resQuery.Log = "does not exist"
			resQuery.Value = nil
		}
	}

	return resQuery
}
