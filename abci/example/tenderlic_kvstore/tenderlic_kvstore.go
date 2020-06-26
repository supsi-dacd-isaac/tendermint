package tenderlic_kvstore

import (
	"encoding/json"
	"github.com/op/go-logging"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/version"
	dbm "github.com/tendermint/tm-db"
	"os"
)

// Logger variables
var logger = logging.MustGetLogger("TenderLICKVstore")
var format = logging.MustStringFormatter(`%{level:.1s}[%{time:2006-01-02|15:04:05.000}] %{message}`)
var backend1 = logging.NewLogBackend(os.Stdout, "", 0)
var backend2 = logging.NewLogBackend(os.Stdout, "", 0)
var backend2Formatter = logging.NewBackendFormatter(backend2, format)
var backend1Leveled = logging.AddModuleLevel(backend1)

var (
	stateKey                         = []byte("stateKey")
	kvPairPrefixKey                  = []byte("kvPairKey:")
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

var _ types.Application = (*Application)(nil)

type Application struct {
	types.BaseApplication
	state State
}

func NewApplication() *Application {
	state := loadState(dbm.NewMemDB())
	return &Application{state: state}
}
