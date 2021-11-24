  // Copyright (c) 2017 Aidos Developer

  // Permission is hereby granted, free of charge, to any person obtaining a copy
  // of this software and associated documentation files (the "Software"), to deal
  // in the Software without restriction, including without limitation the rights
  // to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  // copies of the Software, and to permit persons to whom the Software is
  // furnished to do so, subject to the following conditions:

  // The above copyright notice and this permission notice shall be included in
  // all copies or substantial portions of the Software.

  // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
  // THE SOFTWARE.

  package aidos

  import (
  	"errors"
    "fmt"
  	"github.com/AidosKuneen/gadk"
    "time"
  )

  func importwallet(conf *Conf, req *Request, res *Response) error {
  	//
    fmt.Println("CALLED importwallet but not implemented. Run aidosd -import from command line")
    //
    res.Result = "CALLED importwallet but not implemented. Run aidosd -import from command line"
  	return nil
  }

  func getnewaddress(conf *Conf, req *Request, res *Response) error {
    res.Result =  GetNewAddress().WithChecksum()
    return nil
  }

  func listaddressgroupings(conf *Conf, req *Request, res *Response) error {
  	filemutex.Lock()
    defer filemutex.Unlock()
  	var result [][][]interface{}
  	var r0 [][]interface{}

    for idx, addr := range AddressMapByIndex {
        r1 := make([]interface{}, 3)
        r1[0] = addr.WithChecksum()
        r1[1] =  float64(AddressMap[addr]) / 100000000 // convert to adk = float64(v) / 100000000
        r1[2] = GlobalAccountName
        r0 = append(r0, r1)
        if idx >= LastUsedAddressIndex {
          break
        }
    }
    result = append(result, r0)
  	res.Result = result

  	return nil
  }

  func getbalance(conf *Conf, req *Request, res *Response) error {

     data, ok := req.Params.([]interface{})
  	 if !ok {
  	 	return errors.New("param must be slice")
  	 }
  	 adrstr := "*"
     var singleAddress gadk.Address

     switch len(data) {
     	case 3:
     		fallthrough
     	case 2:
     		n, okk := data[1].(float64)
     		if !okk {
     			return errors.New("invalid number")
     		}
     		if n == 0 {
     			return errors.New("unconfirmed transactions not supported")
     		}
     		fallthrough
      case 1:
   	 	adrstr, ok = data[0].(string)
       if len(adrstr) < 81 && adrstr != "*" {
         ok = false
       } else if adrstr != "*" {
         var errA error
         singleAddress, errA = gadk.ToAddress(adrstr)
         if errA != nil || len(singleAddress) < 81 {
           ok = false
         }
       }
   	 	if !ok {
   	 		return errors.New("invalid address")
   	 	}
     	case 0:
     	default:
     		return errors.New("invalid params")
     	}

    //
    filemutex.Lock()
    defer filemutex.Unlock()

  	var total int64
  	if adrstr == "*" {
  	 		for _, v := range AddressMap {
  	 			total += v
  	 		}
  	}  else {
      total = AddressMap[singleAddress]
    }
    //
    //
  	res.Result =  float64(total) / 100000000
  	// 	return nil
  	// })
  	return nil
  }

  func listaccounts(conf *Conf, req *Request, res *Response) error {
    //
  	result := make(map[string]float64)
    //
    filemutex.Lock()
    defer filemutex.Unlock()

  	var total int64
  	for _, v := range AddressMap {
  	 			total += v
  	}
    //
    result[GlobalAccountName] = float64(total) / 100000000
    res.Result = result
  	return nil
  }

  type info struct {
  	IsValid      bool    `json:"isvalid"`
  	Address      string  `json:"address"`
  	ScriptPubKey string  `json:"scriptPubkey"`
  	IsMine       bool    `json:"ismine"`
  	IsWatchOnly  *bool   `json:"iswatchonly,omitempty"`
  	IsScript     *bool   `json:"isscript,omitempty"`
  	Pubkey       *string `json:"pubkey,omitempty"`
  	IsCompressed *bool   `json:"iscompressed,omitempty"`
  	Account      *string `json:"account,omitempty"`
  }

  //only 'isvalid' params is valid, others may be incorrect.
  func validateaddress(conf *Conf, req *Request, res *Response) error {
  	// mutex.RLock()
  	// defer mutex.RUnlock()
  	data, ok := req.Params.([]interface{})
  	if !ok {
  	 	return errors.New("invalid params")
  	}
  	if len(data) != 1 {
  	 	return errors.New("length of param must be 1")
  	}
  	adrstr, ok := data[0].(string)
  	if !ok {
  	 	return errors.New("invalid address")
  	}

    var singleAddress gadk.Address
    var errA error
    singleAddress, errA = gadk.ToAddress(adrstr)

    if errA != nil || len(singleAddress) < 81 {
       return errors.New("invalid address")
    }
    var inMap bool
    _, inMap = AddressMap[singleAddress]

    valid := false
    if len(singleAddress)==81 {
      valid = true
    }

  	infoi := info{
  	 	IsValid: valid,
  	 	Address: string(singleAddress),
  	 	IsMine:  false,
  	}
  	t := false
    if (inMap) {
    	empty := ""
  		infoi.IsMine = true
  		infoi.Account = &GlobalAccountName
  	 	infoi.IsWatchOnly = &t
  	 	infoi.IsScript = &t
  	 	infoi.Pubkey = &empty
  	 	infoi.IsCompressed = &t
    }
  	// }
  	res.Result = &infoi
  	return nil
  }

  func settxfee(conf *Conf, req *Request, res *Response) error {
  	res.Result = true
  	return nil
  }

  type details struct {
  	Account   string      `json:"account"`
  	Address   gadk.Trytes `json:"address"`
  	Category  string      `json:"category"`
  	Amount    float64     `json:"amount"`
  	Vout      int64       `json:"vout"`
  	Fee       float64     `json:"fee"`
  	Abandoned *bool       `json:"abandoned,omitempty"`
  }

  type tx struct {
  	Amount            float64     `json:"amount"`
  	Fee               float64     `json:"fee"`
  	Confirmations     int         `json:"confirmations"`
  	Blockhash         *string     `json:"blockhash,omitempty"`
  	Blockindex        *int64      `json:"blockindex,omitempty"`
  	Blocktime         *int64      `json:"blocktime,omitempty"`
  	Txid              gadk.Trytes `json:"txid"`
  	Walletconflicts   []string    `json:"walletconflicts"`
  	Time              int64       `json:"time"`
  	TimeReceived      int64       `json:"timereceived"`
  	BIP125Replaceable string      `json:"bip125-replaceable"`
  	Details           []*details  `json:"details"`
  	Hex               string      `json:"hex"`
  }

  func gettransaction(conf *Conf, req *Request, res *Response) error {
  	filemutex.Lock()
    defer filemutex.Unlock()
  	//
  	data, ok := req.Params.([]interface{})
  	if !ok {
  	 	return errors.New("invalid params")
  	}
  	bundlestr := ""
  	switch len(data) {
  	 case 2:
  	 case 1:
  		bundlestr, ok = data[0].(string)
  		if !ok {
  	 		return errors.New("invalid txid")
  	 	}
  	 default:
  	 	return errors.New("invalid params")
  	 }
  	 //
     //
     // load transactions from Mesh
     var dt *transaction

  	 var detailss []*details = []*details{}
  	 bundle := gadk.Trytes(bundlestr)

     resp, errF := Aconf.api.FindTransactions(&gadk.FindTransactionsRequest{
       Bundles: []gadk.Trytes{bundle},
     })
     if errF != nil {
        return errF
    }

     trytes, errT := Aconf.api.GetTrytes(resp.Hashes);
     if errT != nil {
  			return errT
  	}

    nconf := 0

    if (len(trytes.Trytes)>0){
      nconf = 1000
    }

    dt = getBlankTransaction()
    var amount int64

    for _, tr := range trytes.Trytes {
      // address in wallet?
      var inMap bool
      _, inMap = AddressMap[tr.Address]
			if tr.Value != 0 && inMap {
            //dtx = getBlankTransaction()
            cat := "receive"
            if (tr.Value < 0) {
              cat = "send"
            }
            d := &details{
                  Account:   GlobalAccountName,
                  Address:   gadk.Trytes(tr.Address), //"insert address",
                  Category:  cat,
                  Amount:    float64(tr.Value) / 100000000,
                  Abandoned: nil,
            }
            detailss = append(detailss, d)
            amount += tr.Value
			}
		}

  	res.Result = &tx{
  	 	Amount:            float64(amount) / 100000000,
  	 	Confirmations:     nconf,
  	 	Blocktime:         dt.Blocktime,
  	 	Blockhash:         dt.Blockhash,
  	 	Blockindex:        dt.Blockindex,
  	 	Txid:              bundle,
  	 	Walletconflicts:   []string{},
  	 	Time:              dt.Time,
  	 	TimeReceived:      dt.TimeReceived,
  	 	BIP125Replaceable: "no",
  	 	Details:           detailss,
  	}
  	return nil
  }

  type transaction struct {
  	Account  *string     `json:"account"`
  	Address  gadk.Trytes `json:"address"`
  	Category string      `json:"category"`
  	Amount   float64     `json:"amount"`
  	// Label             string      `json:"label"`
  	Vout          int64   `json:"vout"`
  	Fee           float64 `json:"fee"`
  	Confirmations int     `json:"confirmations"`
  	Trusted       *bool   `json:"trusted,omitempty"`
  	// Generated         bool        `json:"generated"`
  	Blockhash       *string     `json:"blockhash,omitempty"`
  	Blockindex      *int64      `json:"blockindex,omitempty"`
  	Blocktime       *int64      `json:"blocktime,omitempty"`
  	Txid            gadk.Trytes `json:"txid"`
  	Walletconflicts []string    `json:"walletconflicts"`
  	Time            int64       `json:"time"`
  	TimeReceived    int64       `json:"timereceived"`
  	// Comment           string      `json:"string"`
  	// To                string `json:"to"`
  	// Otheraccount      string `json:"otheraccount"`
  	BIP125Replaceable string `json:"bip125-replaceable"`
  	Abandoned         *bool  `json:"abandoned,omitempty"`
  }

  //do not support over 1000 txs.
  func listtransactions(conf *Conf, req *Request, res *Response) error {
    // not implemented
  	res.Result = "NOT IMPLEMENTED"
  	return nil
  }

  func getBlankTransaction()(*transaction) {
    f := false
    blank := ""
    tm := time.Now().Unix()
    zero := int64(0)
  	dt := &transaction{
  		Address:           "ADDRESSGOESHERE", //tr.Address.WithChecksum(),
  		Amount:            float64(0) / 100000000,
  		Txid:              "BUNDLEGOESHERE",
  		Walletconflicts:   []string{},
  		Time:              tm,
  		TimeReceived:      tm,
  		BIP125Replaceable: "no",
      Account:           &GlobalAccountName,
      Blockhash:         &blank,
  		Blocktime:         &tm,
  		Blockindex:        &zero,
  		Confirmations: 100000,
  		Trusted: &f,
  	  Category: "receive",
  		Abandoned: nil,
  	}
  	return dt
  }
