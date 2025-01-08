package main

import (
	"encoding/base64"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/deroproject/derohe/dvm"
	"github.com/deroproject/derohe/rpc"
	"github.com/sirupsen/logrus"

	"github.com/docopt/docopt-go"
	"github.com/ybbus/jsonrpc"

	"github.com/civilware/Gnomon/indexer"
	"github.com/civilware/Gnomon/structures"
)

var walletRPCClient jsonrpc.RPCClient
var derodRPCClient jsonrpc.RPCClient
var scid string
var entrypoint string
var ringsize uint64

const version = "v0.0.1"

var command_line string = `Go-DERO-SCAction
SC interactions and executions for the DERO blockchain https://github.com/deroproject/derohe

Usage:
  Go-DERO-SCAction [options]
  Go-DERO-SCAction -h | --help

Options:
  -h --help     Show this screen.
  --daemon-rpc-address=<127.0.0.1:40402>     Connect to daemon rpc.
  --wallet-rpc-address=<127.0.0.1:40403>     Connect to wallet rpc.
  --rpc-userpwd=<"">     Defines username & pwd to run rpc against.
  --operation=<"action">     Defines whether the call will be a scaction or scinstall defined as 'action' or 'install'
  --scid=<"">    Defines scid.
  --entrypoint=<"">     Defines entrypoint.
  --ringsize=<2>     Defines ringsize
  --loopcalls     Defines if the specified function call will be looped and re-called every 60s or not.
  --installfile     Defines the installation file to use when operation=install.
  --debug     Enables debug logging`

// local logger
var logger *logrus.Entry

func main() {
	// tview testing
	//box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")

	/*
		app := tview.NewApplication()
		t := tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetChangedFunc(func() {
				app.Draw()
			})
		t.SetBorder(true)
		numSelections := 0
		t.SetDoneFunc(func(key tcell.Key) {
			currentSelection := t.GetHighlights()
			if key == tcell.KeyEnter {
				if len(currentSelection) > 0 {
					t.Highlight()
				} else {
					t.Highlight("0").ScrollToHighlight()
				}
			} else if len(currentSelection) > 0 {
				index, _ := strconv.Atoi(currentSelection[0])
				if key == tcell.KeyTab {
					index = (index + 1) % numSelections
				} else if key == tcell.KeyBacktab {
					index = (index - 1 + numSelections) % numSelections
				} else {
					return
				}
				t.Highlight(strconv.Itoa(index)).ScrollToHighlight()
			}
		})
		if err := app.SetRoot(t, true).SetFocus(t).Run(); err != nil {
			panic(err)
		}
	*/
	// end tview testing

	var err error
	params := make(map[string]interface{})
	var loopcalls bool

	ringsize = uint64(2)

	// Inspect argument(s)
	arguments, err := docopt.ParseArgs(command_line, nil, version)

	if err != nil {
		log.Fatalf("[Main] Error while parsing arguments err: %s", err)
	}

	// setup logging
	indexer.InitLog(arguments, os.Stdout)
	//indexer.InitLog(arguments, t)
	logger = structures.Logger.WithFields(logrus.Fields{})

	// Set variables from arguments
	daemon_rpc_endpoint := "127.0.0.1:40402"
	if arguments["--daemon-rpc-address"] != nil {
		daemon_rpc_endpoint = arguments["--daemon-rpc-address"].(string)
	}

	wallet_rpc_endpoint := "127.0.0.1:40403"
	if arguments["--wallet-rpc-address"] != nil {
		wallet_rpc_endpoint = arguments["--wallet-rpc-address"].(string)
	}

	if arguments["--ringsize"] != nil {
		rs, err := strconv.ParseInt(arguments["--ringsize"].(string), 10, 64)
		if err != nil {
			logger.Fatalf("[Main] ERROR while converting --ringsize to int64")
		}
		ringsize = uint64(rs)
	}

	if arguments["--loopcalls"] != nil && arguments["--loopcalls"].(bool) {
		loopcalls = true
	}

	operation := "action"
	if arguments["--operation"] != nil {
		oa := arguments["--operation"].(string)
		if oa == "action" || oa == "install" {
			operation = oa
		} else {
			logger.Errorf("[Main] Defined some operation other than 'action' or 'install' incorrectly - %s. Re-run.", oa)
			return
		}
	}

	logger.Printf("[Main] Using wallet RPC endpoint %s", wallet_rpc_endpoint)

	scid = ""
	if arguments["--scid"] != nil {
		scid = arguments["--scid"].(string)
	}

	entrypoint = ""
	if arguments["--entrypoint"] != nil {
		entrypoint = arguments["--entrypoint"].(string)
	}

	pass := ""
	if arguments["--rpc-userpwd"] != nil {
		pass = arguments["--rpc-userpwd"].(string)
	}

	// wallet/derod rpc clients
	// TODO: Add ws/xswd support
	walletRPCClient = jsonrpc.NewClientWithOpts("http://"+wallet_rpc_endpoint+"/json_rpc", &jsonrpc.RPCClientOpts{
		CustomHeaders: map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(pass)),
		},
	})

	derodRPCClient = jsonrpc.NewClient("http://" + daemon_rpc_endpoint + "/json_rpc")

	switch operation {
	case "action":
		if loopcalls {
			for {
				err = scaction(params)
				if err != nil {
					logger.Errorf("[Main] ERROR - %v", err)
				} else {
					logger.Printf("[Main] SCID '%s' function call '%s' completed successfully.", scid, entrypoint)
				}
				logger.Printf("[Main] Round completed. Sleeping 1 minute for next round...")
				time.Sleep(60 * time.Second)
			}
		} else {

			/*
				// TODO: Addtl file params for specific function types e.g. updating a signature from a .sign file etc.
				var sig string
				s, _ := os.ReadFile("contract.bas.sign")
				sig = string(s)
				params["SC_SIG"] = sig
			*/

			err = scaction(params)
			if err != nil {
				logger.Errorf("[Main] ERROR - %v", err)
				return
			}
			logger.Printf("[Main] SCID '%s' function call '%s' completed successfully.", scid, entrypoint)
		}
	case "install":
		var installFile, code string
		if arguments["--installfile"] != nil {
			installFile = arguments["--installfile"].(string)
		}
		s, err := os.ReadFile(installFile)
		if err != nil {
			logger.Errorf("[Main] Defined installfile '%s' could not be found or other err: %v", installFile, err)
			return
		}
		code = string(s)

		_, pos, err := dvm.ParseSmartContract(code)
		if err != nil {
			logger.Errorf("[Main] Install SC ERR: %s %s", err, pos)
			return
		}

		err = scinstall(code)
		if err != nil {
			logger.Errorf("[Main] Install SC ERR: %s", err)
		}
	}
}

func scaction(params map[string]interface{}) (err error) {
	// Get gas estimate based on updatecode function to calculate appropriate storage fees to append
	var rpcArgs = rpc.Arguments{}
	rpcArgs = append(rpcArgs, rpc.Argument{Name: "entrypoint", DataType: "S", Value: entrypoint})

	if len(params) > 0 {
		for k, v := range params {
			switch rval := v.(type) {
			case int:
				rpcArgs = append(rpcArgs, rpc.Argument{Name: k, DataType: "U", Value: uint64(rval)})
			case int64:
				rpcArgs = append(rpcArgs, rpc.Argument{Name: k, DataType: "U", Value: uint64(rval)})
			case float64:
				rpcArgs = append(rpcArgs, rpc.Argument{Name: k, DataType: "U", Value: uint64(rval)})
			case uint64:
				rpcArgs = append(rpcArgs, rpc.Argument{Name: k, DataType: "U", Value: uint64(rval)})
			default:
				rpcArgs = append(rpcArgs, rpc.Argument{Name: k, DataType: "S", Value: rval.(string)})
			}
		}
	}

	var transfers []rpc.Transfer

	return sendtx(rpcArgs, transfers)
}

func scinstall(code string) (err error) {
	var addr rpc.GetAddress_Result
	var txnp rpc.Transfer_Params
	var str rpc.Transfer_Result

	if ringsize != 2 {
		logger.Errorf("[scinstall] Must use ringsize 2 to install a SC.")
		return
	}

	err = walletRPCClient.CallFor(&addr, "GetAddress")
	if addr.Address == "" {
		logger.Errorf("[GetAddress] Failed - %v", err)
		return
	}

	var scRpc rpc.Arguments
	scRpc = append(scRpc, rpc.Argument{Name: "SC_ACTION", DataType: "U", Value: rpc.SC_INSTALL})

	txnp.SC_Code = code
	txnp.SC_RPC = scRpc
	txnp.Ringsize = ringsize
	err = walletRPCClient.CallFor(&str, "Transfer", txnp)
	if err != nil {
		logger.Errorf("[sendtx] err: %v", err)
	} else {
		logger.Printf("[sendtx] Tx sent successfully - txid: %v", str.TXID)
	}
	return
}

func sendtx(rpcArgs rpc.Arguments, transfers []rpc.Transfer) (err error) {
	var gasstr rpc.GasEstimate_Result
	var addr rpc.GetAddress_Result
	var txnp rpc.Transfer_Params
	var str rpc.Transfer_Result

	err = walletRPCClient.CallFor(&addr, "GetAddress")
	if addr.Address == "" {
		logger.Errorf("[GetAddress] Failed - %v", err)
		return
	}
	gasRpc := rpcArgs
	gasRpc = append(gasRpc, rpc.Argument{Name: "SC_ACTION", DataType: "U", Value: rpc.SC_CALL})
	gasRpc = append(gasRpc, rpc.Argument{Name: "SC_ID", DataType: "H", Value: string([]byte(scid))})

	var gasestimateparams rpc.GasEstimate_Params
	if len(transfers) > 0 {
		if ringsize > 2 {
			gasestimateparams = rpc.GasEstimate_Params{SC_RPC: gasRpc, Ringsize: ringsize, Signer: "", Transfers: transfers}
		} else {
			gasestimateparams = rpc.GasEstimate_Params{SC_RPC: gasRpc, Ringsize: ringsize, Signer: addr.Address, Transfers: transfers}
		}
	} else {
		if ringsize > 2 {
			gasestimateparams = rpc.GasEstimate_Params{SC_RPC: gasRpc, Ringsize: ringsize, Signer: ""}
		} else {
			gasestimateparams = rpc.GasEstimate_Params{SC_RPC: gasRpc, Ringsize: ringsize, Signer: addr.Address}
		}
	}
	err = derodRPCClient.CallFor(&gasstr, "DERO.GetGasEstimate", gasestimateparams)
	if err != nil {
		logger.Errorf("[getGasEstimate] gas estimate err %s", err)
		return
	} else {
		logger.Printf("[getGasEstimate] gas estimate results: %v", gasstr)
	}

	txnp.SC_RPC = gasRpc
	if len(transfers) > 0 {
		txnp.Transfers = transfers
	}
	txnp.Ringsize = ringsize
	txnp.Fees = gasstr.GasStorage

	err = walletRPCClient.CallFor(&str, "Transfer", txnp)
	if err != nil {
		logger.Errorf("[sendtx] err: %v", err)
	} else {
		logger.Printf("[sendtx] Tx sent successfully - txid: %v", str.TXID)
	}

	return
}
