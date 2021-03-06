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
  	"encoding/json"
  	"errors"
  	"github.com/AidosKuneen/gadk"
  	"sync"
  	"time"
    "fmt"
  )

  var privileged bool
  var pmutex sync.RWMutex

  var send_mutex sync.RWMutex

  func send(acc string, conf *Conf, trs []gadk.Transfer) (gadk.Trytes, error) {
  	var mwm int64 = 15
    //
    send_mutex.Lock()
  	defer send_mutex.Unlock()
    //
    inputs := []gadk.AddressInfo{}
    addressDirty := []gadk.Address{}
    // find out how much input we need;
    total := int64(0)
    //
    errTrytes,_ := gadk.ToTrytes("9ERROR999")
    //
    for _, tr := range trs {
      total += tr.Value
      addressDirty = append(addressDirty,tr.Address)
    }
    //
    fmt.Println("*sending : preparing seed ")
    seedTrytes, errS := gadk.ToTrytes(Seed)
    if (errS != nil){
      return errTrytes, errS
    }
    // now collect inputs that will cover these. and check balances as we go against the live mesh, so be signature
    fmt.Println("*sending : collecting input addresses ")

    inputsum := int64(0)
    for addr, bal := range AddressMap {
      if (bal > 0) {
        balances, berr := Api.GetBalances([]gadk.Address{addr},100) // get bal from mesh.. dont trust the local cache...too important
        if (berr != nil){
          return errTrytes, berr
        }
        bal2 := balances.Balances[0]
        if (bal2 > 0){
          inputsum += bal2
          var input gadk.AddressInfo
          input.Seed = seedTrytes
          input.Index = GetIndexForAddress(addr)
          input.Security = 2
          addressDirty = append(addressDirty,addr)
          inputs = append(inputs, input)
          fmt.Println("*sending : added input address.")

          if inputsum >= total {
            break // we got enough.
          }
        }
      }
    }

    if (total > inputsum){ // we dont have enough ADK
       fmt.Println("insufficient balance for sending. ",total,inputsum)
       return errTrytes, errors.New("insufficient balance for sending")
    }
    fmt.Println("*sending : generating remainder address.")

    balanceAddress := GetNewAddress() // only gget this if everything ekse is done, to avoid wasting addresses
    addressDirty = append(addressDirty,balanceAddress)
    fmt.Println("*sending : prepare transfers.")

    bundle, errT := gadk.PrepareTransfers(&Api, seedTrytes, trs, inputs, balanceAddress, 2)
    if errT != nil {
    		return errTrytes, errT
    }
    fmt.Println("*sending : get pow engine.")
    engine, pow := gadk.GetBestPoW()
    fmt.Println("*sending : get pow engine (",engine,"), doing pow and sending trytes.")

    fmt.Println("Now calling gadk.SendTrytes")
    errST1 := gadk.SendTrytes(&Api, 1, []gadk.Transaction(bundle), mwm, pow) // note this only Broadcasts (but API takes care of that, still stores)

    if errST1 != nil {
        fmt.Println("Error: ",errST1)
    		return errTrytes, errST1
    }
    // update new balances input and Output
    fmt.Println("*sending : updating address caches:")

    for _, addr := range addressDirty{
      fmt.Println("*sending : updating...")
      balances, berr := Api.GetBalances([]gadk.Address{addr},100) // get bal from mesh.. dont trust the local cache...too important
      if (berr != nil){
          // ognore. next check balance will fix it
      } else {
        AddressMap[addr] = balances.Balances[0]
      }
    }
    fmt.Println("*sending : storing new balances:")
    StoreBalances(AddressMap,AddressMapByIndex,LastUsedAddressIndex)

    fmt.Println("*sending : returning bundle hash:",bundle.Hash())

    return bundle.Hash(), nil // success
  }

  func sendmany(conf *Conf, req *Request, res *Response) error {
  	pmutex.RLock()
  	if !privileged {
  		pmutex.RUnlock()
  		return errors.New("not priviledged")
  	}
  	pmutex.RUnlock()
  	mutex.Lock()
  	defer mutex.Unlock()
  	data, ok := req.Params.([]interface{})
  	if !ok {
  		return errors.New("invalid params")
  	}
  	if len(data) < 2 || len(data) > 5 {
  		return errors.New("invalid param length (less than 2 or more than 5)")
  	}
  	acc, ok := data[0].(string)
  	if !ok {
  		return errors.New("invalid account")
  	}
  	target := make(map[string]float64)
  	switch data[1].(type) {
  	case string:
  		t := data[1].(string)
  		if err := json.Unmarshal([]byte(t), &target); err != nil {
  			return err
  		}
  	case map[string]interface{}:
  		t := data[1].(map[string]interface{})
  		for k, v := range t {
  			f, ok := v.(float64)
  			if !ok {
  				return errors.New("param must be a  map string")
  			}
  			target[k] = f
  		}
  	default:
  		return errors.New("param must be a  map string")
  	}
  	trs := make([]gadk.Transfer, len(target))
  	i := 0
  	var err error
  	for k, v := range target {
  		trs[i].Address, err = gadk.ToAddress(k)
  		if err != nil {
  			return err
  		}
  		trs[i].Value = int64(v * 100000000)
  		trs[i].Tag = gadk.Trytes(conf.Tag)
  		i++
  	}
  	res.Result, err = send(acc, conf, trs)
  	return err
  }

  //done
  func sendfrom(conf *Conf, req *Request, res *Response) error {
  	var err error
  	pmutex.RLock()
  	if !privileged {
  		pmutex.RUnlock()
  		return errors.New("not priviledged")
  	}
  	pmutex.RUnlock()
  	mutex.Lock()
  	defer mutex.Unlock()
  	data, ok := req.Params.([]interface{})
  	if !ok {
  		return errors.New("invalid params")
  	}
  	if len(data) < 3 || len(data) > 6 {
  		return errors.New("invalid params")
  	}
  	acc, ok := data[0].(string)
  	if !ok {
  		return errors.New("invalid account")
  	}
  	var tr gadk.Transfer
  	tr.Tag = gadk.Trytes(conf.Tag)
  	adrstr, ok := data[1].(string)
  	if !ok {
  		return errors.New("invalid address")
  	}
  	tr.Address, err = gadk.ToAddress(adrstr)
  	if err != nil {
  		return err
  	}
  	value, ok := data[2].(float64)
  	if !ok {
  		return errors.New("invalid value")
  	}
  	tr.Value = int64(value * 100000000)
  	res.Result, err = send(acc, conf, []gadk.Transfer{tr})
  	return err
  }

 //done
  func sendtoaddress(conf *Conf, req *Request, res *Response) error {
  	var err error
    pmutex.RLock()
    if !privileged {
      pmutex.RUnlock()
      return errors.New("not priviledged")
    }
    pmutex.RUnlock()
    mutex.Lock()
    defer mutex.Unlock()
  	var tr gadk.Transfer
  	tr.Tag = gadk.Trytes(conf.Tag)

  	data, ok := req.Params.([]interface{})
  	if !ok {
  		return errors.New("invalid params")
  	}
  	if len(data) > 5 || len(data) < 2 {
  		return errors.New("invalid params")
  	}
  	adrstr, ok := data[0].(string)
  	if !ok {
  		return errors.New("invalid address")
  	}
  	value, ok := data[1].(float64)
  	if !ok {
  		return errors.New("invalid value")
  	}
  	tr.Address, err = gadk.ToAddress(adrstr)
  	if err != nil {
  		return err
  	}
    tr.Value = int64(value * 100000000)
  	res.Result, err = send("*", conf, []gadk.Transfer{tr})
  	return err
  }


  func walletpassphrase(conf *Conf, req *Request, res *Response) error {
  	pmutex.RLock()
  	if privileged {
  		pmutex.RUnlock()
  		return nil
  	}
  	pmutex.RUnlock()
  	data, ok := req.Params.([]interface{})
  	if !ok {
  		return errors.New("invalid params")
  	}
  	if len(data) != 2 {
  		return errors.New("invalid param length")
  	}
  	pwd, ok := data[0].(string)
  	if !ok {
  		return errors.New("invalid password")
  	}
  	sec, ok := data[1].(float64)
  	if !ok {
  		return errors.New("invalid time")
  	}
  	if !LoadSeedAPIUnlock([]byte(pwd)){
      return errors.New("invalid password")
    }
  	go func() {
  		pmutex.Lock()
  		privileged = true
  		pmutex.Unlock()
  		time.Sleep(time.Second * time.Duration(sec))
  		pmutex.Lock()
  		privileged = false
  		pmutex.Unlock()
  	}()
  	return nil
  }
