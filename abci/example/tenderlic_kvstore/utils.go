package tenderlic_kvstore

import (
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"github.com/op/go-logging"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/version"
	"net"
)

// Logger variables

func (app *Application) CalcSHA512Hash(input string) string {
	h := sha512.New()
	h.Write([]byte(input))
	macHash := h.Sum(nil)
	return fmt.Sprintf("%x", macHash)
}

func (app *Application) SetLogger() {
	backend1Leveled.SetLevel(logging.CRITICAL, "")
	logging.SetBackend(backend1Leveled, backend2Formatter)
}

func (app *Application) Contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func (app *Application) GetMacAddr() ([]string, error) {
	netInts, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var macs []string
	for _, netInt := range netInts {
		mac := netInt.HardwareAddr.String()
		if mac != "" {
			macs = append(macs, mac)
		}
	}
	return macs, nil
}

func (app *Application) GetAllowedMeters() []byte {
	keyFlag := []byte("allowed")
	allowedMeters, err := app.state.db.Get(prefixKey(keyFlag))
	if err != nil {
		panic(err)
	}
	return allowedMeters
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

func (app *Application) Commit() types.ResponseCommit {
	// Using a memdb - just return the big endian size of the db
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Size)
	app.state.AppHash = appHash
	app.state.Height++
	saveState(app.state)
	return types.ResponseCommit{Data: appHash}
}
