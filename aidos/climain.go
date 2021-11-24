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
  	"strings"
  	"fmt"
    "github.com/AidosKuneen/gadk"
  	"log"
    "os/exec"
    "bufio"
  	"os"
    "io/ioutil"
    shellwords "github.com/mattn/go-shellwords"
  )

  func Pcheck(err error){
      if err != nil {
        panic(err)
      }
  }

  var Api gadk.API
  var Aconf *Conf

  func CLImain(passwd []byte, doimport bool, generate bool) {

    GlobalAccountName = "default"
    Aconf = ParseConf("aidosd.conf")

    Api = *gadk.NewAPI(Aconf.Node, nil)

    fmt.Println("******************************")
    fmt.Println("Welcome to aidosd (AZ9 Wallet)")
    fmt.Println("*****************************")
    fmt.Println("")
    fmt.Print("Checking for existing seed file (seed.enc)...")

    _, err := ioutil.ReadFile("seed.enc")
    seedFileExists := true
    if (err!=nil){
        fmt.Println("not found.")
        fmt.Println("")
        seedFileExists = false
    } else {
      fmt.Println("found.")
    }

    fmt.Print("Checking for existing legacy database file (aidosd.db)...")
    _, errDB := ioutil.ReadFile("aidosd.db")
    legacyDBFileExists := true
    if (errDB!=nil){
        fmt.Println("not found.")
        fmt.Println("")
        legacyDBFileExists = false
    }else {
      fmt.Println("found.")
    }

    if seedFileExists && (doimport || generate) {
        fmt.Println("")
        fmt.Println("")
        fmt.Println("######   WARNING  ######: aidosd called with -import or -generate, but seed.enc file already exists.")
        fmt.Println("")
        fmt.Println("Do you want to DELETE the existing seed.enc file and proceed with the import/generate ?")
        if !AskConfirm() {
          os.Exit(1)
        }
        os.Remove("seed.enc")
        os.Remove("transactions.dat") // if we remove the seed, we can also remove these if exists
        os.Remove("balances.dat") // if we remove the seed, we can also remove these if exists
        seedFileExists = false
    }

    seedimported := ""
    hasimportedseed := false

    if !seedFileExists && !doimport && !generate {
        if legacyDBFileExists {
            fmt.Println("")
            fmt.Println("************************************")
            fmt.Println("** No V2 Seed file (seed.enc) exists, but there is a legacy V1 database file present.")
            fmt.Println("** ")
            fmt.Println("** Do you want to import the seed from the old V1 database (aidosd.db) and initialize/")
            fmt.Println("** upgrade the existing wallet? (Enter \"no\" to abort, or \"yes\" to continue)")
            fmt.Println("************************************")
            if (AskConfirm()){
                fmt.Println("Extracting seed from old database.")
                seedimported, hasimportedseed = getSeedFromOldDB()
                if (!hasimportedseed){
                  fmt.Println("#############")
                  fmt.Println("No seed found in legacy DB. Either the legacy database is empty or corrupt.")
                  fmt.Println("")
                  log.Fatal("aborted")
                }
                // // continuing
                doimport = true
                //
              }  else {
                  fmt.Println("")
                  fmt.Println("No seed imported from Legacy DB.")
                  fmt.Println("")
                  fmt.Println("To set up a new seed/import a known seed please run aidosd with parameter")
                  fmt.Println("  \"-import\" or \"-generate\"")
                  fmt.Println("'aidosd -import' will prompt you for your 81 char seed and then perform a full scan")
                  fmt.Println("'aidosd -generate' will generate a new seed for you and set up a new wallet")
                  os.Exit(1)
              }
        } else {
            fmt.Println("Please run aidosd with parameter \"-import\" or \"-generate\"")
            fmt.Println("'aidosd -import' will prompt you for your 81 char seed and then perform a full scan")
            fmt.Println("'aidosd -generate' will generate a new seed for you and set up a new wallet")
            os.Exit(1)
        }
    }

    if doimport {
        if hasimportedseed {
          ImportSeed(seedimported) // use seed provided
        } else {
          ImportSeed("") // prompt for seed
        }
        OpenWallet(passwd)
        ScanStoreAddressesAndBalancesINIT(false)
        AddressMap, AddressMapByIndex, LastUsedAddressIndex =  ReadAddressesFromFile()
        StoreConfirmedTransactions(AddressMap,  AddressMapByIndex)

    } else if generate {

        GenerateSeed()
        OpenWallet(passwd)
        ScanStoreAddressesAndBalancesINIT(generate)
        AddressMap, AddressMapByIndex, LastUsedAddressIndex =  ReadAddressesFromFile()

    } else {

      OpenWallet(passwd)
      AddressMap, AddressMapByIndex, LastUsedAddressIndex =  ReadAddressesFromFile()

    }

    if (len(AddressMap)<=0 || len(AddressMapByIndex) <= 0){
      fmt.Println("balances.dat file corrupt. Please remove and initialize wallet from seed.")
      os.Exit(1)
    }
  }

  func TransactionScanner() { // default

      fmt.Println("Loading Address Balances [Index 0 -",LastUsedAddressIndex+100,"]...")
      ScanAddressesForBalanceChanges(100)

      newTransactions := StoreConfirmedTransactions(AddressMap,  AddressMapByIndex)
      fmt.Println("Found new transactions:",len(newTransactions))

      if len(newTransactions) > 0 {
        fmt.Println("Loading transaction details from mesh... Please wait.")
        trytes, errTry:= Api.GetTrytes(newTransactions)
        if (errTry!=nil ){ // skip error line, likely unknown transaction
          fmt.Println(errTry)
        }

        for _, transaction := range trytes.Trytes { // get trytes for all new transactions, and extract bundle and value
            bundleHash := transaction.Bundle
            address := transaction.Address
            value := transaction.Value
            if (value > 0){
                fmt.Println("|---- Bundle: ",bundleHash)
                fmt.Println("|---- Address:",address)
                fmt.Println("|---- Value:  ",value)
                //
                if Aconf.Notify == "" {
              		log.Println("not calling notify shellscript as none is defined")
              	} else {
                  cmd := strings.Replace(Aconf.Notify, "%s", string(bundleHash), -1)
                  args, err := shellwords.Parse(cmd)
                  if err != nil {
                    log.Println(err)
                    continue
                  }
                  var out []byte
                  if len(args) == 1 {
                      out, err = exec.Command(args[0]).Output()
                  } else {
                      out, err = exec.Command(args[0], args[1:]...).Output()
                  }
                  if err != nil {
                      log.Println(err)
                      continue
                  }
                  log.Println("executed ", cmd, ",output:", string(out))
                }
            }
        }
      }

  }

  func AskConfirm() bool {
  	var input string

  	fmt.Printf("(Enter Yes or No ): ")
    reader := bufio.NewReader(os.Stdin)
    input, _ = reader.ReadString('\n')
    // convert CRLF to LF
    input = strings.Replace(input, "\n", "", -1)
    input = strings.Replace(input, "\r", "", -1)

  	input = strings.TrimSpace(input)
  	input = strings.ToLower(input)

  	if input == "n" || input == "no" {
  		return false
  	} else if input == "y" || input == "yes" {
  	   return true
    }
    return AskConfirm()
  }
